package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestGeocodeService(t *testing.T, handler http.HandlerFunc) *GeocodeService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &GeocodeService{client: server.Client(), baseURL: server.URL}
}

func TestGeocodeService_Search_ReturnsCandidates(t *testing.T) {
	svc := newTestGeocodeService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != nominatimUserAgent {
			t.Errorf("expected identifying User-Agent, got %q", got)
		}
		if got := r.URL.Query().Get("q"); got != "Rome" {
			t.Errorf("expected query q=Rome, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"display_name":"Rome, Italy","lat":"41.9028","lon":"12.4964"}]`))
	})

	candidates, err := svc.Search("Rome")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].DisplayName != "Rome, Italy" || candidates[0].Lat != 41.9028 || candidates[0].Lng != 12.4964 {
		t.Fatalf("unexpected candidate: %+v", candidates[0])
	}
}

func TestGeocodeService_Search_EmptyQueryReturnsNoCandidatesWithoutRequest(t *testing.T) {
	called := false
	svc := newTestGeocodeService(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte(`[]`))
	})

	candidates, err := svc.Search("   ")
	if err != nil {
		t.Fatalf("expected no error for blank query, got %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates for blank query, got %+v", candidates)
	}
	if called {
		t.Fatalf("expected no HTTP request for a blank query")
	}
}

func TestGeocodeService_Search_NoMatchReturnsEmptyNotError(t *testing.T) {
	svc := newTestGeocodeService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	})

	candidates, err := svc.Search("Nonexistentville")
	if err != nil {
		t.Fatalf("expected no error for zero matches, got %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected zero candidates, got %+v", candidates)
	}
}

func TestGeocodeService_Search_ServerErrorReturnsEmptyNotError(t *testing.T) {
	svc := newTestGeocodeService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	candidates, err := svc.Search("Rome")
	if err != nil {
		t.Fatalf("expected upstream failure to be treated as no-results, got error %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected zero candidates, got %+v", candidates)
	}
}

func TestGeocodeService_Search_SkipsUnparsableCoordinates(t *testing.T) {
	svc := newTestGeocodeService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"display_name":"Bad","lat":"not-a-number","lon":"12.5"},{"display_name":"Good","lat":"1.5","lon":"2.5"}]`))
	})

	candidates, err := svc.Search("Rome")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(candidates) != 1 || candidates[0].DisplayName != "Good" {
		t.Fatalf("expected only the parsable candidate to survive, got %+v", candidates)
	}
}
