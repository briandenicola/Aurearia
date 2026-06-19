package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeploymentDocsContainPublicExposureBetaChecklist(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "docs", "deployment.md"))
	if err != nil {
		t.Fatalf("failed to read deployment docs: %v", err)
	}
	doc := string(content)

	required := map[string][]string{
		"external nginx checks": {"Public Exposure Beta Acceptance Checklist", "External nginx checks", "X-Forwarded-For", "X-Forwarded-Proto"},
		"backup restore drill":  {"Backup restore drill", "SQLite", "uploads"},
		"WebAuthn HTTPS":        {"WebAuthn HTTPS", "WEBAUTHN_RP_ID", "WEBAUTHN_ORIGIN", "HTTPS"},
		"private media":         {"Private media", "/uploads/*", "/api/showcase/:slug/uploads/*"},
		"agent port privacy":    {"Agent port privacy", "8081", "not published", "AGENT_INTERNAL_SERVICE_TOKEN"},
	}

	for name, tokens := range required {
		for _, token := range tokens {
			if !strings.Contains(doc, token) {
				t.Fatalf("deployment docs missing %s token %q", name, token)
			}
		}
	}
}
