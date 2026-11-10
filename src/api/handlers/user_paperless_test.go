package handlers

import "testing"

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
