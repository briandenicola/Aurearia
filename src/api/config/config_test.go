package config

import (
	"testing"
)

func TestAllowedOriginsExcludesDevOriginsInRelease(t *testing.T) {
	cfg := &Config{WebAuthnOrigin: "https://coins.example.com"}

	t.Setenv("GIN_MODE", "release")
	for _, origin := range cfg.AllowedOrigins() {
		if origin == "http://localhost:5173" || origin == "http://localhost:8080" {
			// corsMiddleware sends Access-Control-Allow-Credentials for any
			// matched origin, so a localhost allowance in production is a
			// credentialed-request hole.
			t.Fatalf("release mode must not allow dev origin %q", origin)
		}
	}
}

func TestAllowedOriginsIncludesDevOriginsOutsideRelease(t *testing.T) {
	cfg := &Config{WebAuthnOrigin: "http://localhost:8080"}

	t.Setenv("GIN_MODE", "debug")
	var found bool
	for _, origin := range cfg.AllowedOrigins() {
		if origin == "http://localhost:5173" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the Vite dev origin outside release mode")
	}
}

func TestExplicitCORSOriginsAlwaysWin(t *testing.T) {
	cfg := &Config{CORSOrigins: "https://a.example.com, https://b.example.com"}

	t.Setenv("GIN_MODE", "release")
	got := cfg.AllowedOrigins()
	if len(got) != 2 || got[0] != "https://a.example.com" || got[1] != "https://b.example.com" {
		t.Fatalf("expected the explicit list verbatim, got %v", got)
	}
}
