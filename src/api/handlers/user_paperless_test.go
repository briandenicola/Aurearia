package handlers

import (
	"strings"
	"testing"
)

func TestNormalizePaperlessURL(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"   ", "", false},
		{"https://paperless.example.com", "https://paperless.example.com", false},
		{"https://paperless.example.com/", "https://paperless.example.com", false},
		{" http://10.0.0.5:8000/paperless// ", "http://10.0.0.5:8000/paperless", false},
		{"javascript:alert(1)", "", true},
		{"ftp://paperless.example.com", "", true},
		{"paperless.example.com", "", true},
		{"https://user:pass@paperless.example.com", "", true},
		{"https://paperless.example.com/?q=1", "", true},
	}
	for _, tc := range cases {
		got, err := normalizePaperlessURL(tc.in)
		if (err != nil) != tc.wantErr {
			t.Fatalf("normalizePaperlessURL(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
		}
		if got != tc.want {
			t.Fatalf("normalizePaperlessURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizePaperlessTags(t *testing.T) {
	got, err := normalizePaperlessTags(" coins,brian ,, shopping ")
	if err != nil || got != "coins, brian, shopping" {
		t.Fatalf("got %q %v", got, err)
	}
	if got, err := normalizePaperlessTags(""); err != nil || got != "" {
		t.Fatalf("empty should clear: %q %v", got, err)
	}
	if _, err := normalizePaperlessTags(strings.Repeat("a", 501)); err == nil {
		t.Fatal("expected a length error")
	}
}
