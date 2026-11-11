package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Paperless task states as reported by paperless-ngx's /api/tasks/ endpoint.
const (
	PaperlessTaskPending = "PENDING"
	PaperlessTaskStarted = "STARTED"
	PaperlessTaskSuccess = "SUCCESS"
	PaperlessTaskFailure = "FAILURE"
)

// PaperlessTask is the subset of a paperless-ngx consumption task we need.
type PaperlessTask struct {
	Status     string
	DocumentID *int
	Result     string
}

// PaperlessClient talks to a user's paperless-ngx instance.
type PaperlessClient interface {
	PostDocument(ctx context.Context, baseURL, token, title, filename, contentType string, body io.Reader) (string, error)
	GetTask(ctx context.Context, baseURL, token, taskID string) (*PaperlessTask, error)
	GetDocumentFile(ctx context.Context, baseURL, token string, documentID int, kind PaperlessFileKind) (*PaperlessFile, error)
	// Ping makes one authenticated request so URL, network path and token are
	// all exercised.
	Ping(ctx context.Context, baseURL, token string) error
}

// PaperlessFileKind selects which rendition of a document to fetch.
type PaperlessFileKind string

const (
	// PaperlessFilePreview is paperless-ngx's viewable rendition (archived PDF, or the original image).
	PaperlessFilePreview PaperlessFileKind = "preview"
	// PaperlessFileThumbnail is the small page-1 thumbnail paperless-ngx generates.
	PaperlessFileThumbnail PaperlessFileKind = "thumb"
)

// PaperlessFile is a streamed document rendition. Callers must close Body.
type PaperlessFile struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

// PaperlessError carries a non-2xx paperless response without echoing tokens.
type PaperlessError struct {
	Operation   string
	StatusCode  int
	BodyExcerpt string
}

func (e *PaperlessError) Error() string {
	return fmt.Sprintf("paperless-ngx %s failed: status=%d body=%q", e.Operation, e.StatusCode, e.BodyExcerpt)
}

type HTTPPaperlessClient struct {
	client *http.Client
}

// paperlessBlockedPrefixes are refused even though private ranges are allowed:
// "this network", benchmarking, reserved and IPv6 discard/documentation space.
var paperlessBlockedPrefixes = mustParseOutboundCIDRs(
	"0.0.0.0/8",
	"198.18.0.0/15",
	"240.0.0.0/4",
	"100::/64",
)

// IsDisallowedPaperlessIP reports whether a resolved paperless-ngx address is
// refused. Unlike the general outbound guard, private LAN (RFC 1918, IPv6 ULA)
// and CGNAT/Tailscale (100.64.0.0/10) addresses are allowed, because a
// self-hosted paperless almost always lives there. Loopback, link-local
// (including the 169.254.169.254 cloud metadata endpoint), multicast and
// unspecified addresses stay blocked.
func IsDisallowedPaperlessIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, prefix := range paperlessBlockedPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func paperlessDialContext(resolver OutboundResolver) func(context.Context, string, string) (net.Conn, error) {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		var candidates []net.IP
		if ip := net.ParseIP(host); ip != nil {
			candidates = []net.IP{ip}
		} else {
			resolved, err := resolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve paperless-ngx host: %w", err)
			}
			for _, addr := range resolved {
				candidates = append(candidates, net.IP(addr.Unmap().AsSlice()))
			}
		}
		var dialErr error
		for _, ip := range candidates {
			if IsDisallowedPaperlessIP(ip) {
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			dialErr = err
		}
		if dialErr != nil {
			return nil, dialErr
		}
		return nil, fmt.Errorf("%w: paperless-ngx host resolves only to loopback, link-local or reserved addresses", ErrOutboundTargetBlocked)
	}
}

func newPaperlessHTTPClient(resolver OutboundResolver) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = paperlessDialContext(resolver)
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("paperless-ngx redirected too many times")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("%w: unsupported redirect scheme", ErrOutboundTargetBlocked)
			}
			return nil
		},
	}
}

// NewHTTPPaperlessClient talks to the user's paperless-ngx through a dialer
// that allows LAN/Tailscale addresses but blocks loopback, link-local and
// cloud-metadata targets. Every dial (including after redirects) is checked.
func NewHTTPPaperlessClient() *HTTPPaperlessClient {
	return &HTTPPaperlessClient{client: newPaperlessHTTPClient(nil)}
}

func paperlessEndpoint(baseURL, path string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return "", errors.New("invalid paperless-ngx URL")
	}
	return base + path, nil
}

func (c *HTTPPaperlessClient) do(req *http.Request, token, operation string) ([]byte, error) {
	req.Header.Set("Authorization", "Token "+token)
	// No version pin: paperless-ngx answers with its own default API version.
	// Pinning an old one breaks when paperless drops it (3.x accepts only 9
	// and 10 and answers 406 to anything else).
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paperless-ngx %s request failed: %w", operation, err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("paperless-ngx %s read failed: %w", operation, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		excerpt := strings.TrimSpace(string(payload))
		if len(excerpt) > 300 {
			excerpt = excerpt[:300]
		}
		return nil, &PaperlessError{Operation: operation, StatusCode: resp.StatusCode, BodyExcerpt: excerpt}
	}
	return payload, nil
}

func (c *HTTPPaperlessClient) PostDocument(ctx context.Context, baseURL, token, title, filename, contentType string, body io.Reader) (string, error) {
	endpoint, err := paperlessEndpoint(baseURL, "/api/documents/post_document/")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if title != "" {
		if err := writer.WriteField("title", title); err != nil {
			return "", err
		}
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="document"; filename=%q`, filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, body); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	payload, err := c.do(req, token, "upload")
	if err != nil {
		return "", err
	}
	// paperless-ngx answers with the consumption task UUID as a JSON string.
	var taskID string
	if err := json.Unmarshal(payload, &taskID); err != nil {
		taskID = strings.Trim(strings.TrimSpace(string(payload)), `"`)
	}
	if taskID == "" {
		return "", errors.New("paperless-ngx upload returned no task id")
	}
	return taskID, nil
}

func (c *HTTPPaperlessClient) Ping(ctx context.Context, baseURL, token string) error {
	endpoint, err := paperlessEndpoint(baseURL, "/api/documents/?page_size=1")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	_, err = c.do(req, token, "connection test")
	return err
}

// paperlessTaskPayload covers both task formats. API v9 reports the new
// document as related_document; v10 (paperless-ngx 3.x) reports
// related_document_ids and result_data.document_id instead. Result fields are
// raw so a type change cannot fail the whole parse.
type paperlessTaskPayload struct {
	Status             string            `json:"status"`
	Result             json.RawMessage   `json:"result"`
	ResultMessage      json.RawMessage   `json:"result_message"`
	RelatedDocument    json.RawMessage   `json:"related_document"`
	RelatedDocumentIDs []json.RawMessage `json:"related_document_ids"`
	ResultData         json.RawMessage   `json:"result_data"`
}

type paperlessTaskResultData struct {
	DocumentID json.RawMessage `json:"document_id"`
	Message    string          `json:"message"`
	Error      string          `json:"error"`
}

func (p paperlessTaskPayload) documentID() *int {
	if id := parsePaperlessDocumentID(p.RelatedDocument); id != nil {
		return id
	}
	for _, raw := range p.RelatedDocumentIDs {
		if id := parsePaperlessDocumentID(raw); id != nil {
			return id
		}
	}
	var data paperlessTaskResultData
	if len(p.ResultData) > 0 && json.Unmarshal(p.ResultData, &data) == nil {
		if id := parsePaperlessDocumentID(data.DocumentID); id != nil {
			return id
		}
	}
	// Last resort: paperless's own result text, "Success. New document id 4949 created".
	if match := paperlessNewDocumentIDPattern.FindStringSubmatch(p.message()); match != nil {
		if id, err := strconv.Atoi(match[1]); err == nil && id > 0 {
			return &id
		}
	}
	return nil
}

var paperlessNewDocumentIDPattern = regexp.MustCompile(`(?i)new document id (\d+)`)

func (p paperlessTaskPayload) message() string {
	for _, raw := range []json.RawMessage{p.Result, p.ResultMessage} {
		var text string
		if len(raw) > 0 && json.Unmarshal(raw, &text) == nil && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	var data paperlessTaskResultData
	if len(p.ResultData) > 0 && json.Unmarshal(p.ResultData, &data) == nil {
		if msg := strings.TrimSpace(data.Error); msg != "" {
			return msg
		}
		return strings.TrimSpace(data.Message)
	}
	return ""
}

// normalizePaperlessTaskStatus maps v9/v10 status spellings onto the states
// Aurearia acts on; anything unrecognised stays pending.
func normalizePaperlessTaskStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESS", "SUCCEEDED", "COMPLETED", "COMPLETE":
		return PaperlessTaskSuccess
	case "FAILURE", "FAILED", "ERROR", "REVOKED":
		return PaperlessTaskFailure
	case "STARTED", "RUNNING", "PROCESSING":
		return PaperlessTaskStarted
	default:
		return PaperlessTaskPending
	}
}

func (c *HTTPPaperlessClient) GetTask(ctx context.Context, baseURL, token, taskID string) (*PaperlessTask, error) {
	endpoint, err := paperlessEndpoint(baseURL, "/api/tasks/?task_id="+url.QueryEscape(taskID))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	payload, err := c.do(req, token, "task lookup")
	if err != nil {
		return nil, err
	}
	tasks, err := parsePaperlessTasks(payload)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		// Paperless can take a moment before the task row exists.
		return &PaperlessTask{Status: PaperlessTaskPending}, nil
	}
	return &PaperlessTask{
		Status:     normalizePaperlessTaskStatus(tasks[0].Status),
		Result:     tasks[0].message(),
		DocumentID: tasks[0].documentID(),
	}, nil
}

// parsePaperlessTasks accepts the task list as a plain array or as a
// paginated {"results": [...]} object.
func parsePaperlessTasks(payload []byte) ([]paperlessTaskPayload, error) {
	var tasks []paperlessTaskPayload
	if err := json.Unmarshal(payload, &tasks); err == nil {
		return tasks, nil
	}
	var page struct {
		Results *[]paperlessTaskPayload `json:"results"`
	}
	if err := json.Unmarshal(payload, &page); err == nil && page.Results != nil {
		return *page.Results, nil
	}
	excerpt := strings.TrimSpace(string(payload))
	if len(excerpt) > 200 {
		excerpt = excerpt[:200]
	}
	return nil, fmt.Errorf("paperless-ngx task lookup returned an unexpected payload: %s", excerpt)
}

// parsePaperlessDocumentID accepts related_document as a number or a numeric
// string; paperless-ngx has returned both across versions.
func parsePaperlessDocumentID(raw json.RawMessage) *int {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var asInt int
	if err := json.Unmarshal(raw, &asInt); err == nil && asInt > 0 {
		return &asInt
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if id, err := strconv.Atoi(strings.TrimSpace(asString)); err == nil && id > 0 {
			return &id
		}
	}
	return nil
}

func (c *HTTPPaperlessClient) GetDocumentFile(ctx context.Context, baseURL, token string, documentID int, kind PaperlessFileKind) (*PaperlessFile, error) {
	if documentID <= 0 || (kind != PaperlessFilePreview && kind != PaperlessFileThumbnail) {
		return nil, errors.New("invalid paperless-ngx document request")
	}
	endpoint, err := paperlessEndpoint(baseURL, fmt.Sprintf("/api/documents/%d/%s/", documentID, kind))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paperless-ngx %s request failed: %w", kind, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return nil, &PaperlessError{Operation: string(kind), StatusCode: resp.StatusCode, BodyExcerpt: strings.TrimSpace(string(excerpt))}
	}
	return &PaperlessFile{Body: resp.Body, ContentType: resp.Header.Get("Content-Type"), ContentLength: resp.ContentLength}, nil
}
