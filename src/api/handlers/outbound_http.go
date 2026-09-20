package handlers

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"github.com/briandenicola/ancient-coins-api/services"
)

var errOutboundTargetBlocked = services.ErrOutboundTargetBlocked

type outboundResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

var defaultOutboundHTTPClient = newRestrictedHTTPClient(net.DefaultResolver)

var outboundHTTPClientFactory = func() *http.Client {
	return defaultOutboundHTTPClient
}

func validateOutboundURL(rawURL string) (*url.URL, error) {
	return services.ValidateOutboundURL(rawURL)
}

func newRestrictedHTTPClient(resolver outboundResolver) *http.Client {
	return services.NewRestrictedHTTPClient(resolver, 30*time.Second, 10)
}

func restrictedDialContext(resolver outboundResolver) func(ctx context.Context, network, address string) (net.Conn, error) {
	return services.RestrictedDialContext(resolver)
}

func isOutboundTargetBlockedError(err error) bool {
	return services.IsOutboundTargetBlockedError(err)
}

func isDisallowedIP(ip net.IP) bool {
	return services.IsDisallowedOutboundIP(ip)
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
