package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
)

// DescribePaperlessError turns a paperless-ngx failure into a message that
// says what went wrong and what to check. It never includes the API token:
// errors from this client carry URLs and status lines, not request headers.
func DescribePaperlessError(err error) string {
	if err == nil {
		return ""
	}
	var (
		dnsErr      *net.DNSError
		paperless   *PaperlessError
		certErr     *tls.CertificateVerificationError
		unknownAuth x509.UnknownAuthorityError
		hostnameErr x509.HostnameError
		recordErr   tls.RecordHeaderError
		netErr      net.Error
	)
	switch {
	case errors.Is(err, ErrPaperlessNotConfigured):
		return err.Error()
	case errors.Is(err, ErrPaperlessTagNotFound):
		return err.Error() + ". Check the tag names in Settings → Connections."
	case errors.Is(err, ErrOutboundTargetBlocked):
		return "The paperless-ngx URL points at a blocked address (localhost or link-local). Use the host's LAN name or IP."
	case errors.As(err, &paperless):
		switch paperless.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return "paperless-ngx rejected the API token. Copy a fresh token from paperless-ngx → My Profile."
		case http.StatusNotAcceptable:
			return fmt.Sprintf("paperless-ngx refused the API version Aurearia requested for the %s (HTTP 406). Response: %s", paperless.Operation, paperless.BodyExcerpt)
		case http.StatusNotFound:
			return fmt.Sprintf("paperless-ngx returned 404 for the %s. Check the URL is the paperless-ngx root, without /api or another path.", paperless.Operation)
		default:
			msg := fmt.Sprintf("paperless-ngx returned HTTP %d for the %s.", paperless.StatusCode, paperless.Operation)
			if paperless.BodyExcerpt != "" {
				msg += " Response: " + paperless.BodyExcerpt
			}
			return msg
		}
	case errors.As(err, &dnsErr):
		return fmt.Sprintf("The Aurearia server could not resolve %q. If paperless-ngx runs in Docker, put both containers on the same Docker network, or use the host's LAN IP and published port.", dnsErr.Name)
	case errors.Is(err, syscall.ECONNREFUSED):
		return "Connection refused: nothing is listening at that host and port. Check the port, and that it is published if paperless-ngx runs in Docker."
	case errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, syscall.ENETUNREACH):
		return "The paperless-ngx host is unreachable from the Aurearia server. Check the address and any firewall between them."
	case errors.As(err, &certErr), errors.As(err, &unknownAuth), errors.As(err, &hostnameErr):
		return "paperless-ngx's TLS certificate was not accepted by the Aurearia server. Use a certificate it trusts, or an http:// URL on your LAN."
	case errors.As(err, &recordErr), strings.Contains(err.Error(), "server gave HTTP response to HTTPS client"):
		return "TLS handshake failed: the URL uses https:// but paperless-ngx answered with plain HTTP. Try http://."
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return "Timed out connecting to paperless-ngx. Check the host and port, and that the Aurearia server can reach it."
	case err.Error() == "invalid paperless-ngx URL":
		return "The paperless-ngx URL is invalid. Use an absolute http:// or https:// URL."
	default:
		return "Could not reach paperless-ngx: " + err.Error()
	}
}
