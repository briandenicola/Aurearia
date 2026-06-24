package services

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCRuntimeConfig struct {
	Provider     *oidc.Provider
	OAuth2Config oauth2.Config
}

type OIDCDiscoveryFactory func(ctx context.Context, issuerURL string) (*oidc.Provider, error)

func NewDefaultOIDCDiscoveryFactory() OIDCDiscoveryFactory {
	return oidc.NewProvider
}
