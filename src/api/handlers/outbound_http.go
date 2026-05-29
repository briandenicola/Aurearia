package handlers

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

var errOutboundTargetBlocked = errors.New("outbound target blocked")

type outboundResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

var defaultOutboundHTTPClient = newRestrictedHTTPClient(net.DefaultResolver)

var outboundHTTPClientFactory = func() *http.Client {
	return defaultOutboundHTTPClient
}

var blockedIPPrefixes = mustParseCIDRs(
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

func validateOutboundURL(rawURL string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme")
	}
	hostname := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if hostname == "" || hostname == "localhost" {
		return nil, fmt.Errorf("%w: disallowed hostname", errOutboundTargetBlocked)
	}
	if ip := net.ParseIP(hostname); ip != nil && isDisallowedIP(ip) {
		return nil, fmt.Errorf("%w: disallowed address", errOutboundTargetBlocked)
	}
	return parsed, nil
}

func newRestrictedHTTPClient(resolver outboundResolver) *http.Client {
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = restrictedDialContext(resolver)

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after too many redirects")
			}
			_, err := validateOutboundURL(req.URL.String())
			return err
		},
	}
}

func restrictedDialContext(resolver outboundResolver) func(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		if ip := net.ParseIP(host); ip != nil {
			if isDisallowedIP(ip) {
				return nil, fmt.Errorf("%w: disallowed address", errOutboundTargetBlocked)
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
			ipAddr := addr.Unmap()
			ip := net.IP(ipAddr.AsSlice())
			if isDisallowedIP(ip) {
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
		return nil, fmt.Errorf("%w: host resolves only to disallowed addresses", errOutboundTargetBlocked)
	}
}

func isOutboundTargetBlockedError(err error) bool {
	return errors.Is(err, errOutboundTargetBlocked)
}

func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	for _, prefix := range blockedIPPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}

func mustParseCIDRs(values ...string) []*net.IPNet {
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
