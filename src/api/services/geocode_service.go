package services

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	nominatimBaseURL      = "https://nominatim.openstreetmap.org/search"
	nominatimUserAgent    = "Aurearia/1.0 (self-hosted coin collection app)"
	nominatimMaxResults   = 3
	geocodeRequestTimeout = 8 * time.Second
)

// GeocodeCandidate is a single geocoding result for a place name.
type GeocodeCandidate struct {
	DisplayName string  `json:"displayName"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

// GeocodeService looks up coordinates for a place name via OpenStreetMap
// Nominatim. Only the user-typed name is ever sent to it - no coin,
// collection, or account data.
type GeocodeService struct {
	client  *http.Client
	baseURL string
}

// NewGeocodeService creates a new GeocodeService pointed at the public
// Nominatim endpoint.
func NewGeocodeService() *GeocodeService {
	return &GeocodeService{
		client:  &http.Client{Timeout: geocodeRequestTimeout},
		baseURL: nominatimBaseURL,
	}
}

// NewGeocodeServiceForTest creates a GeocodeService pointed at an arbitrary
// base URL (e.g. an httptest.Server), for tests that need to stand in for
// Nominatim without making real network calls.
func NewGeocodeServiceForTest(baseURL string) *GeocodeService {
	return &GeocodeService{
		client:  &http.Client{Timeout: geocodeRequestTimeout},
		baseURL: baseURL,
	}
}

// Search returns up to a few geocoding candidates for a place name, ordered
// by Nominatim's relevance ranking. Returns an empty, non-nil slice (not an
// error) when Nominatim has no match, so callers can tell "no results" apart
// from a real failure and fall back to manual pin placement either way.
func (s *GeocodeService) Search(query string) ([]GeocodeCandidate, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []GeocodeCandidate{}, nil
	}

	reqURL := s.baseURL + "?" + url.Values{
		"q":      {trimmed},
		"format": {"jsonv2"},
		"limit":  {strconv.Itoa(nominatimMaxResults)},
	}.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", nominatimUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []GeocodeCandidate{}, nil
	}

	var raw []struct {
		DisplayName string `json:"display_name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	candidates := make([]GeocodeCandidate, 0, len(raw))
	for _, r := range raw {
		lat, errLat := strconv.ParseFloat(r.Lat, 64)
		lng, errLng := strconv.ParseFloat(r.Lon, 64)
		if errLat != nil || errLng != nil {
			continue
		}
		candidates = append(candidates, GeocodeCandidate{
			DisplayName: r.DisplayName,
			Lat:         lat,
			Lng:         lng,
		})
	}
	return candidates, nil
}
