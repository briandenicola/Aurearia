package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

const shipmentCarrierHTTPTimeout = 15 * time.Second

type USPSShipmentClientConfig struct {
	BaseURL      string
	APIKey       string
	APIKeyHeader string
}

type UPSShipmentClientConfig struct {
	BaseURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
}

type FedExShipmentClientConfig struct {
	BaseURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
}

func USPSShipmentClientConfigFromSettings(settings *SettingsService) USPSShipmentClientConfig {
	return USPSShipmentClientConfig{
		BaseURL:      settings.GetSetting(SettingUSPSAPIBaseURL),
		APIKey:       settings.GetSetting(SettingUSPSAPIKey),
		APIKeyHeader: settings.GetSetting(SettingUSPSAPIKeyHeader),
	}
}

func UPSShipmentClientConfigFromSettings(settings *SettingsService) UPSShipmentClientConfig {
	return UPSShipmentClientConfig{
		BaseURL:      settings.GetSetting(SettingUPSAPIBaseURL),
		TokenURL:     settings.GetSetting(SettingUPSTokenURL),
		ClientID:     settings.GetSetting(SettingUPSClientID),
		ClientSecret: settings.GetSetting(SettingUPSClientSecret),
		Scope:        settings.GetSetting(SettingUPSScope),
	}
}

func FedExShipmentClientConfigFromSettings(settings *SettingsService) FedExShipmentClientConfig {
	return FedExShipmentClientConfig{
		BaseURL:      settings.GetSetting(SettingFedExAPIBaseURL),
		TokenURL:     settings.GetSetting(SettingFedExTokenURL),
		ClientID:     settings.GetSetting(SettingFedExClientID),
		ClientSecret: settings.GetSetting(SettingFedExClientSecret),
		Scope:        settings.GetSetting(SettingFedExScope),
	}
}

type USPSShipmentCarrierClient struct {
	baseURL      string
	apiKey       string
	apiKeyHeader string
	httpClient   *http.Client
}

func NewUSPSShipmentCarrierClient(config USPSShipmentClientConfig, httpClient *http.Client) (*USPSShipmentCarrierClient, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	apiKey := strings.TrimSpace(config.APIKey)
	if baseURL == "" {
		return nil, fmt.Errorf("USPS base URL is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("USPS API key is required")
	}
	header := strings.TrimSpace(config.APIKeyHeader)
	if header == "" {
		header = "X-API-Key"
	}
	return &USPSShipmentCarrierClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiKey:       apiKey,
		apiKeyHeader: header,
		httpClient:   withDefaultShipmentHTTPClient(httpClient),
	}, nil
}

func (c *USPSShipmentCarrierClient) Carrier() models.ShipmentCarrier {
	return models.ShipmentCarrierUSPS
}

func (c *USPSShipmentCarrierClient) GetTracking(ctx context.Context, trackingNumber string) (ShipmentTrackingSnapshot, error) {
	normalizedTracking := strings.TrimSpace(trackingNumber)
	if normalizedTracking == "" {
		return ShipmentTrackingSnapshot{}, fmt.Errorf("tracking number is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/tracking/"+url.PathEscape(normalizedTracking), nil)
	if err != nil {
		return ShipmentTrackingSnapshot{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set(c.apiKeyHeader, c.apiKey)

	payload, err := executeShipmentTrackingRequest(c.httpClient, req)
	if err != nil {
		return ShipmentTrackingSnapshot{}, err
	}
	return normalizeTrackingSnapshot(c.Carrier(), normalizedTracking, payload), nil
}

type UPSShipmentCarrierClient struct {
	baseURL    string
	httpClient *http.Client
	token      *carrierOAuthTokenProvider
}

func NewUPSShipmentCarrierClient(config UPSShipmentClientConfig, httpClient *http.Client) (*UPSShipmentCarrierClient, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("UPS base URL is required")
	}
	tokenProvider, err := newCarrierOAuthTokenProvider(
		models.ShipmentCarrierUPS,
		config.TokenURL,
		config.ClientID,
		config.ClientSecret,
		config.Scope,
		withDefaultShipmentHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	return &UPSShipmentCarrierClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: withDefaultShipmentHTTPClient(httpClient),
		token:      tokenProvider,
	}, nil
}

func (c *UPSShipmentCarrierClient) Carrier() models.ShipmentCarrier {
	return models.ShipmentCarrierUPS
}

func (c *UPSShipmentCarrierClient) GetTracking(ctx context.Context, trackingNumber string) (ShipmentTrackingSnapshot, error) {
	return getTrackingWithOAuth(ctx, c.Carrier(), c.baseURL, c.httpClient, c.token, trackingNumber)
}

type FedExShipmentCarrierClient struct {
	baseURL    string
	httpClient *http.Client
	token      *carrierOAuthTokenProvider
}

func NewFedExShipmentCarrierClient(config FedExShipmentClientConfig, httpClient *http.Client) (*FedExShipmentCarrierClient, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("FedEx base URL is required")
	}
	tokenProvider, err := newCarrierOAuthTokenProvider(
		models.ShipmentCarrierFedEx,
		config.TokenURL,
		config.ClientID,
		config.ClientSecret,
		config.Scope,
		withDefaultShipmentHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	return &FedExShipmentCarrierClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: withDefaultShipmentHTTPClient(httpClient),
		token:      tokenProvider,
	}, nil
}

func (c *FedExShipmentCarrierClient) Carrier() models.ShipmentCarrier {
	return models.ShipmentCarrierFedEx
}

func (c *FedExShipmentCarrierClient) GetTracking(ctx context.Context, trackingNumber string) (ShipmentTrackingSnapshot, error) {
	return getTrackingWithOAuth(ctx, c.Carrier(), c.baseURL, c.httpClient, c.token, trackingNumber)
}

func getTrackingWithOAuth(
	ctx context.Context,
	carrier models.ShipmentCarrier,
	baseURL string,
	httpClient *http.Client,
	tokenProvider *carrierOAuthTokenProvider,
	trackingNumber string,
) (ShipmentTrackingSnapshot, error) {
	normalizedTracking := strings.TrimSpace(trackingNumber)
	if normalizedTracking == "" {
		return ShipmentTrackingSnapshot{}, fmt.Errorf("tracking number is required")
	}
	token, err := tokenProvider.Token(ctx)
	if err != nil {
		return ShipmentTrackingSnapshot{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/tracking/"+url.PathEscape(normalizedTracking), nil)
	if err != nil {
		return ShipmentTrackingSnapshot{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	payload, err := executeShipmentTrackingRequest(httpClient, req)
	if err != nil {
		return ShipmentTrackingSnapshot{}, err
	}
	return normalizeTrackingSnapshot(carrier, normalizedTracking, payload), nil
}

type carrierOAuthTokenProvider struct {
	carrier      models.ShipmentCarrier
	tokenURL     string
	clientID     string
	clientSecret string
	scope        string
	httpClient   *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func newCarrierOAuthTokenProvider(
	carrier models.ShipmentCarrier,
	tokenURL, clientID, clientSecret, scope string,
	httpClient *http.Client,
) (*carrierOAuthTokenProvider, error) {
	if strings.TrimSpace(tokenURL) == "" {
		return nil, fmt.Errorf("%s token URL is required", carrier)
	}
	if strings.TrimSpace(clientID) == "" {
		return nil, fmt.Errorf("%s client ID is required", carrier)
	}
	if strings.TrimSpace(clientSecret) == "" {
		return nil, fmt.Errorf("%s client secret is required", carrier)
	}
	return &carrierOAuthTokenProvider{
		carrier:      carrier,
		tokenURL:     strings.TrimSpace(tokenURL),
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		scope:        strings.TrimSpace(scope),
		httpClient:   withDefaultShipmentHTTPClient(httpClient),
	}, nil
}

func (p *carrierOAuthTokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.accessToken != "" && time.Until(p.expiresAt) > 30*time.Second {
		return p.accessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	if p.scope != "" {
		form.Set("scope", p.scope)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s token request failed: %w", p.carrier, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("%s token request failed with HTTP %d: %s", p.carrier, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("%s token response decode failed: %w", p.carrier, err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("%s token response missing access_token", p.carrier)
	}
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 300
	}

	p.accessToken = strings.TrimSpace(tokenResp.AccessToken)
	p.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return p.accessToken, nil
}

func executeShipmentTrackingRequest(httpClient *http.Client, req *http.Request) (map[string]any, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shipment tracking request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("shipment tracking request returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("shipment tracking response decode failed: %w", err)
	}
	return payload, nil
}

func normalizeTrackingSnapshot(carrier models.ShipmentCarrier, trackingNumber string, payload map[string]any) ShipmentTrackingSnapshot {
	now := time.Now().UTC()
	rawCurrentStatus := firstPayloadString(payload, "current_status", "status", "status_description", "statusDescription")
	currentStatus := NormalizeCarrierShipmentStatus(carrier, rawCurrentStatus)
	estimatedDeliveryAt, _ := firstPayloadTime(payload, "estimated_delivery_at", "estimatedDeliveryAt", "estimated_delivery", "estimatedDelivery")
	deliveredAt, _ := firstPayloadTime(payload, "delivered_at", "deliveredAt", "delivery_time", "deliveryTime")

	eventPayloads := payloadEvents(payload)
	events := make([]ShipmentTrackingEvent, 0, len(eventPayloads))
	for index, eventPayload := range eventPayloads {
		rawStatus := firstPayloadString(eventPayload, "status", "status_description", "statusDescription", "event_status", "eventStatus")
		occurredAt, hasOccurredAt := firstPayloadTime(eventPayload, "occurred_at", "occurredAt", "timestamp", "eventTime", "time")
		if !hasOccurredAt {
			occurredAt = &now
		}
		eventKey := firstPayloadString(eventPayload, "event_key", "eventKey", "id", "event_id", "eventId")
		if eventKey == "" {
			eventKey = fmt.Sprintf("%s:%s:%d:%d", carrier, strings.ToLower(rawStatus), occurredAt.Unix(), index)
		}
		rawPayload := marshalAnyJSON(eventPayload)
		events = append(events, ShipmentTrackingEvent{
			EventKey:     eventKey,
			Status:       NormalizeCarrierShipmentStatus(carrier, rawStatus),
			StatusSource: models.ShipmentStatusSourceAPI,
			OccurredAt:   *occurredAt,
			Location:     firstPayloadString(eventPayload, "location", "city", "facility"),
			Description:  firstPayloadString(eventPayload, "description", "message", "details"),
			RawStatus:    rawStatus,
			RawPayload:   rawPayload,
		})
	}

	return ShipmentTrackingSnapshot{
		Carrier:             carrier,
		TrackingNumber:      trackingNumber,
		CurrentStatus:       currentStatus,
		CurrentStatusSource: models.ShipmentStatusSourceAPI,
		RawCurrentStatus:    rawCurrentStatus,
		EstimatedDeliveryAt: estimatedDeliveryAt,
		DeliveredAt:         deliveredAt,
		Events:              events,
		SyncedAt:            now,
	}
}

func firstPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := topLevelValueCaseInsensitive(payload, key); ok {
			switch typed := value.(type) {
			case string:
				trimmed := strings.TrimSpace(typed)
				if trimmed != "" {
					return trimmed
				}
			case fmt.Stringer:
				trimmed := strings.TrimSpace(typed.String())
				if trimmed != "" {
					return trimmed
				}
			}
		}
		if value, ok := lookupNestedValueCaseInsensitive(payload, key); ok {
			switch typed := value.(type) {
			case string:
				trimmed := strings.TrimSpace(typed)
				if trimmed != "" {
					return trimmed
				}
			case fmt.Stringer:
				trimmed := strings.TrimSpace(typed.String())
				if trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

func firstPayloadTime(payload map[string]any, keys ...string) (*time.Time, bool) {
	for _, key := range keys {
		if value, ok := topLevelValueCaseInsensitive(payload, key); ok {
			parsed, hasParsed := parseTimestamp(value)
			if hasParsed {
				return &parsed, true
			}
		}
		if value, ok := lookupNestedValueCaseInsensitive(payload, key); ok {
			parsed, hasParsed := parseTimestamp(value)
			if hasParsed {
				return &parsed, true
			}
		}
	}
	return nil, false
}

func parseTimestamp(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return time.Time{}, false
		}
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, trimmed); err == nil {
				return parsed.UTC(), true
			}
		}
	case float64:
		return time.Unix(int64(typed), 0).UTC(), true
	case int64:
		return time.Unix(typed, 0).UTC(), true
	case json.Number:
		if asInt, err := typed.Int64(); err == nil {
			return time.Unix(asInt, 0).UTC(), true
		}
	}
	return time.Time{}, false
}

func payloadEvents(payload map[string]any) []map[string]any {
	eventKeys := map[string]bool{
		"events":         true,
		"event":          true,
		"activities":     true,
		"activity":       true,
		"trackingevents": true,
		"trackingevent":  true,
		"scans":          true,
		"scan":           true,
	}
	var events []map[string]any
	collectEventPayloads(payload, eventKeys, &events)
	return events
}

func collectEventPayloads(node any, eventKeys map[string]bool, out *[]map[string]any) {
	switch typed := node.(type) {
	case map[string]any:
		for key, value := range typed {
			normalizedKey := strings.ToLower(strings.TrimSpace(key))
			if eventKeys[normalizedKey] {
				if array, ok := value.([]any); ok {
					for _, entry := range array {
						if mapped, ok := entry.(map[string]any); ok {
							*out = append(*out, mapped)
						}
					}
				}
			}
			collectEventPayloads(value, eventKeys, out)
		}
	case []any:
		for _, value := range typed {
			collectEventPayloads(value, eventKeys, out)
		}
	}
}

func lookupNestedValueCaseInsensitive(payload map[string]any, wantedKey string) (any, bool) {
	wanted := strings.ToLower(strings.TrimSpace(wantedKey))
	var walk func(node any) (any, bool)
	walk = func(node any) (any, bool) {
		switch typed := node.(type) {
		case map[string]any:
			for key, value := range typed {
				if strings.ToLower(strings.TrimSpace(key)) == wanted {
					return value, true
				}
				if nestedValue, ok := walk(value); ok {
					return nestedValue, true
				}
			}
		case []any:
			for _, value := range typed {
				if nestedValue, ok := walk(value); ok {
					return nestedValue, true
				}
			}
		}
		return nil, false
	}
	return walk(payload)
}

func topLevelValueCaseInsensitive(payload map[string]any, wantedKey string) (any, bool) {
	wanted := strings.ToLower(strings.TrimSpace(wantedKey))
	for key, value := range payload {
		if strings.ToLower(strings.TrimSpace(key)) == wanted {
			return value, true
		}
	}
	return nil, false
}

func marshalAnyJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(encoded))
}

func withDefaultShipmentHTTPClient(httpClient *http.Client) *http.Client {
	if httpClient != nil {
		return httpClient
	}
	return &http.Client{Timeout: shipmentCarrierHTTPTimeout}
}
