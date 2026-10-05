package models

import "testing"

func TestApiKeyCapabilityParsingUsesExactTokens(t *testing.T) {
	for _, tc := range []struct {
		name         string
		capabilities string
		wantRead     bool
		wantWrite    bool
		wantCopilot  bool
	}{
		{name: "read", capabilities: "read", wantRead: true},
		{name: "read copilot", capabilities: "read,copilot", wantRead: true, wantCopilot: true},
		{name: "read write", capabilities: "read,write", wantRead: true, wantWrite: true},
		{name: "all", capabilities: "read,write,copilot", wantRead: true, wantWrite: true, wantCopilot: true},
		{name: "write implies read", capabilities: "write", wantRead: true, wantWrite: true},
		{name: "readwrite malformed", capabilities: "readwrite"},
		{name: "copilot substring malformed", capabilities: "read,copilotx", wantRead: true},
		{name: "xwritex malformed", capabilities: "xwritex"},
		{name: "notread malformed", capabilities: "notread"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiKey := ApiKey{Capabilities: tc.capabilities}
			if got := apiKey.HasRead(); got != tc.wantRead {
				t.Fatalf("HasRead() = %v, want %v", got, tc.wantRead)
			}
			if got := apiKey.HasWrite(); got != tc.wantWrite {
				t.Fatalf("HasWrite() = %v, want %v", got, tc.wantWrite)
			}
			if got := apiKey.HasCopilot(); got != tc.wantCopilot {
				t.Fatalf("HasCopilot() = %v, want %v", got, tc.wantCopilot)
			}
		})
	}
}
