package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var ErrOutboundTargetBlocked = errors.New("outbound target blocked")

type OutboundResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

var blockedOutboundIPPrefixes = mustParseOutboundCIDRs(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"198.18.0.0/15",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"::1/128",
	"fe80::/10",
	"fc00::/7",
	"ff00::/8",
)

func ValidateOutboundURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if !parsed.IsAbs() || parsed.Opaque != "" {
		return nil, fmt.Errorf("invalid absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("%w: credentials are not allowed", ErrOutboundTargetBlocked)
	}
	hostname := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if hostname == "" || hostname == "localhost" {
		return nil, fmt.Errorf("%w: disallowed hostname", ErrOutboundTargetBlocked)
	}
	if ip := net.ParseIP(hostname); ip != nil {
		return nil, fmt.Errorf("%w: IP literals are not allowed", ErrOutboundTargetBlocked)
	}
	return parsed, nil
}

func NewRestrictedHTTPClient(resolver OutboundResolver, timeout time.Duration, maxRedirects int) *http.Client {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if maxRedirects <= 0 {
		maxRedirects = 10
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = RestrictedDialContext(resolver)
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after too many redirects")
			}
			_, err := ValidateOutboundURL(req.URL.String())
			return err
		},
	}
}

func RestrictedDialContext(resolver OutboundResolver) func(context.Context, string, string) (net.Conn, error) {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if ip := net.ParseIP(host); ip != nil {
			if IsDisallowedOutboundIP(ip) {
				return nil, fmt.Errorf("%w: disallowed address", ErrOutboundTargetBlocked)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}
		resolved, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host: %w", err)
		}
		if len(resolved) == 0 {
			return nil, fmt.Errorf("failed to resolve host")
		}
		var dialErr error
		for _, addr := range resolved {
			ip := net.IP(addr.Unmap().AsSlice())
			if IsDisallowedOutboundIP(ip) {
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
		return nil, fmt.Errorf("%w: host resolves only to disallowed addresses", ErrOutboundTargetBlocked)
	}
}

func IsOutboundTargetBlockedError(err error) bool {
	return errors.Is(err, ErrOutboundTargetBlocked)
}

func IsDisallowedOutboundIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, prefix := range blockedOutboundIPPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseOutboundCIDRs(values ...string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			panic(err)
		}
		nets = append(nets, ipNet)
	}
	return nets
}
