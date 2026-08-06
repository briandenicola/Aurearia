package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestHTTPParcelAppClient_ListDeliveriesPreservesFailureBody(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"_success":false,"_error_message":"","detail":"invalid api key"}`))
	}))
	defer server.Close()

	client := NewHTTPParcelAppClient()
	client.baseURL = server.URL

	_, err := client.ListDeliveries(context.Background(), "test-key")
	if err == nil {
		t.Fatalf("expected ParcelApp error")
	}
	var parcelErr *ParcelAppError
	if !errors.As(err, &parcelErr) {
		t.Fatalf("error type = %T, want *ParcelAppError", err)
	}
	if parcelErr.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", parcelErr.StatusCode)
	}
	if !strings.Contains(parcelErr.BodyExcerpt, "invalid api key") {
		t.Fatalf("body excerpt = %q, want invalid api key", parcelErr.BodyExcerpt)
	}
	if requestedPath != "/deliveries/?filter_mode=recent" {
		t.Fatalf("requested path = %q, want recent deliveries filter", requestedPath)
	}
}

func TestHTTPParcelAppClient_ListDeliveriesAcceptsActualSuccessField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"deliveries":[{"tracking_number":"9402150105800000607499","carrier_code":"usps","status_code":2,"events":[]}]}`))
	}))
	defer server.Close()

	client := NewHTTPParcelAppClient()
	client.baseURL = server.URL

	deliveries, err := client.ListDeliveries(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("ListDeliveries returned error for actual ParcelApp success field: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("deliveries len = %d, want 1", len(deliveries))
	}
	if deliveries[0].TrackingNumber != "9402150105800000607499" {
		t.Fatalf("tracking = %q, want parsed tracking number", deliveries[0].TrackingNumber)
	}
}

func TestHTTPParcelAppClient_ListDeliveries_AcceptsLargePayload(t *testing.T) {
	longAdditional := strings.Repeat("X", 9000)
	body := `{"success":true,"deliveries":[{"tracking_number":"9402150105800000607499","carrier_code":"dhlgm","status_code":2,"description":"Mark Weldon","events":[{"event":"DEPARTURE ORIGIN DHL ECOMMERCE FACILITY","date":"August 5 2026 03:41","location":"Avenel, NJ, UNITED","additional":"` + longAdditional + `"}]}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewHTTPParcelAppClient()
	client.baseURL = server.URL

	deliveries, err := client.ListDeliveries(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("ListDeliveries returned error for large payload: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("deliveries len = %d, want 1", len(deliveries))
	}
	if deliveries[0].StatusCode != 2 {
		t.Fatalf("status code = %d, want 2", deliveries[0].StatusCode)
	}
}
