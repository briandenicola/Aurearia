package models

import "testing"

func TestApiKeyCapabilityParsingUsesExactTokens(t *testing.T) {
	for _, tc := range []struct {
		name         string
		capabilities string
		wantRead     bool
		wantWrite    bool
	}{
		{name: "read", capabilities: "read", wantRead: true, wantWrite: false},
		{name: "read write", capabilities: "read,write", wantRead: true, wantWrite: true},
		{name: "write implies read", capabilities: "write", wantRead: true, wantWrite: true},
		{name: "readwrite malformed", capabilities: "readwrite", wantRead: false, wantWrite: false},
		{name: "xwritex malformed", capabilities: "xwritex", wantRead: false, wantWrite: false},
		{name: "notread malformed", capabilities: "notread", wantRead: false, wantWrite: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiKey := ApiKey{Capabilities: tc.capabilities}
			if got := apiKey.HasRead(); got != tc.wantRead {
				t.Fatalf("HasRead() = %v, want %v", got, tc.wantRead)
			}
			if got := apiKey.HasWrite(); got != tc.wantWrite {
				t.Fatalf("HasWrite() = %v, want %v", got, tc.wantWrite)
			}
		})
	}
}
