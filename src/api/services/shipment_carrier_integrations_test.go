package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestUSPSShipmentCarrierClient_GetTracking(t *testing.T) {
	var gotAPIKeyHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tracking/94001000" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAPIKeyHeader = r.Header.Get("X-API-Key")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":             "In Transit",
			"estimatedDeliveryAt": "2026-08-06T10:00:00Z",
			"events": []map[string]any{
				{
					"eventId":     "evt-1",
					"status":      "Out for Delivery",
					"timestamp":   "2026-08-05T12:00:00Z",
					"location":    "Boston, MA",
					"description": "Out for delivery",
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewUSPSShipmentCarrierClient(USPSShipmentClientConfig{
		BaseURL:      server.URL,
		APIKey:       "test-usps-key",
		APIKeyHeader: "X-API-Key",
	}, server.Client())
	if err != nil {
		t.Fatalf("new usps client: %v", err)
	}

	snapshot, err := client.GetTracking(context.Background(), "94001000")
	if err != nil {
		t.Fatalf("get tracking: %v", err)
	}
	if gotAPIKeyHeader != "test-usps-key" {
		t.Fatalf("api key header = %q, want test-usps-key", gotAPIKeyHeader)
	}
	if snapshot.CurrentStatus != models.ShipmentStatusInTransit {
		t.Fatalf("current status = %s, want %s", snapshot.CurrentStatus, models.ShipmentStatusInTransit)
	}
	if len(snapshot.Events) != 1 {
		t.Fatalf("events len = %d, want 1", len(snapshot.Events))
	}
	if snapshot.Events[0].Status != models.ShipmentStatusOutForDelivery {
		t.Fatalf("event status = %s, want %s", snapshot.Events[0].Status, models.ShipmentStatusOutForDelivery)
	}
}

func TestUPSShipmentCarrierClient_GetTracking_WithOAuth(t *testing.T) {
	var tokenCalls int32
	var trackingAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			atomic.AddInt32(&tokenCalls, 1)
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if r.Form.Get("grant_type") != "client_credentials" {
				t.Fatalf("grant_type = %q", r.Form.Get("grant_type"))
			}
			if r.Form.Get("client_id") != "ups-client" || r.Form.Get("client_secret") != "ups-secret" {
				t.Fatalf("unexpected ups token credentials")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "ups-token-123",
				"expires_in":   300,
				"token_type":   "bearer",
			})
		case "/tracking/1Z999":
			trackingAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"current_status": "Delivered",
				"delivered_at":   "2026-08-06T14:31:00Z",
				"events": []map[string]any{
					{
						"id":          "deliv-1",
						"status":      "Delivered",
						"occurred_at": "2026-08-06T14:31:00Z",
						"location":    "Providence, RI",
						"description": "Delivered",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewUPSShipmentCarrierClient(UPSShipmentClientConfig{
		BaseURL:      server.URL,
		TokenURL:     server.URL + "/oauth/token",
		ClientID:     "ups-client",
		ClientSecret: "ups-secret",
		Scope:        "tracking",
	}, server.Client())
	if err != nil {
		t.Fatalf("new ups client: %v", err)
	}

	snapshot, err := client.GetTracking(context.Background(), "1Z999")
	if err != nil {
		t.Fatalf("get tracking: %v", err)
	}
	if trackingAuth != "Bearer ups-token-123" {
		t.Fatalf("authorization = %q, want bearer token", trackingAuth)
	}
	if snapshot.CurrentStatus != models.ShipmentStatusDelivered {
		t.Fatalf("status = %s, want delivered", snapshot.CurrentStatus)
	}

	_, err = client.GetTracking(context.Background(), "1Z999")
	if err != nil {
		t.Fatalf("second get tracking: %v", err)
	}
	if atomic.LoadInt32(&tokenCalls) != 1 {
		t.Fatalf("token endpoint called %d times, want 1 (cached token)", tokenCalls)
	}
}

func TestFedExShipmentCarrierClient_GetTracking_WithOAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "fedex-token-abc",
				"expires_in":   120,
			})
		case "/tracking/123456":
			if r.Header.Get("Authorization") != "Bearer fedex-token-abc" {
				t.Fatalf("missing fedex bearer token")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"statusDescription": "Delivery Exception",
				"events": []map[string]any{
					{
						"event_key":   "fx-1",
						"status":      "Delivery Exception",
						"timestamp":   "2026-08-07T09:15:00Z",
						"location":    "Hartford, CT",
						"description": "Address issue",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewFedExShipmentCarrierClient(FedExShipmentClientConfig{
		BaseURL:      server.URL,
		TokenURL:     server.URL + "/oauth/token",
		ClientID:     "fedex-client",
		ClientSecret: "fedex-secret",
	}, server.Client())
	if err != nil {
		t.Fatalf("new fedex client: %v", err)
	}

	snapshot, err := client.GetTracking(context.Background(), "123456")
	if err != nil {
		t.Fatalf("get tracking: %v", err)
	}
	if snapshot.CurrentStatus != models.ShipmentStatusException {
		t.Fatalf("status = %s, want exception", snapshot.CurrentStatus)
	}
}

func TestCarrierConfigFromSettings(t *testing.T) {
	svc, db := newTestSettingsService(t)
	db.Create(&models.AppSetting{Key: SettingUSPSAPIBaseURL, Value: "https://usps.example"})
	db.Create(&models.AppSetting{Key: SettingUSPSAPIKey, Value: "usps-key"})
	db.Create(&models.AppSetting{Key: SettingUPSAPIBaseURL, Value: "https://ups.example"})
	db.Create(&models.AppSetting{Key: SettingUPSTokenURL, Value: "https://ups.example/oauth/token"})
	db.Create(&models.AppSetting{Key: SettingUPSClientID, Value: "ups-id"})
	db.Create(&models.AppSetting{Key: SettingUPSClientSecret, Value: "ups-secret"})
	db.Create(&models.AppSetting{Key: SettingFedExAPIBaseURL, Value: "https://fedex.example"})
	db.Create(&models.AppSetting{Key: SettingFedExTokenURL, Value: "https://fedex.example/oauth/token"})
	db.Create(&models.AppSetting{Key: SettingFedExClientID, Value: "fedex-id"})
	db.Create(&models.AppSetting{Key: SettingFedExClientSecret, Value: "fedex-secret"})

	usps := USPSShipmentClientConfigFromSettings(svc)
	ups := UPSShipmentClientConfigFromSettings(svc)
	fedex := FedExShipmentClientConfigFromSettings(svc)

	if usps.BaseURL != "https://usps.example" || usps.APIKey != "usps-key" {
		t.Fatalf("unexpected usps config: %+v", usps)
	}
	if ups.TokenURL == "" || ups.ClientID != "ups-id" {
		t.Fatalf("unexpected ups config: %+v", ups)
	}
	if fedex.TokenURL == "" || fedex.ClientID != "fedex-id" {
		t.Fatalf("unexpected fedex config: %+v", fedex)
	}
}

func TestNewCarrierClients_RequireCredentials(t *testing.T) {
	_, err := NewUSPSShipmentCarrierClient(USPSShipmentClientConfig{}, &http.Client{Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "USPS base URL is required") {
		t.Fatalf("unexpected usps error: %v", err)
	}

	_, err = NewUPSShipmentCarrierClient(UPSShipmentClientConfig{
		BaseURL: "https://ups.example",
	}, &http.Client{Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "token URL is required") {
		t.Fatalf("unexpected ups error: %v", err)
	}

	_, err = NewFedExShipmentCarrierClient(FedExShipmentClientConfig{
		BaseURL:      "https://fedex.example",
		TokenURL:     "https://fedex.example/oauth/token",
		ClientID:     "fedex-id",
		ClientSecret: "",
	}, &http.Client{Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "client secret is required") {
		t.Fatalf("unexpected fedex error: %v", err)
	}
}

func TestUSPSShipmentCarrierClient_TrackingNumberEscaped(t *testing.T) {
	var gotRequestURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestURI = r.RequestURI
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "Pending"})
	}))
	defer server.Close()

	client, err := NewUSPSShipmentCarrierClient(USPSShipmentClientConfig{
		BaseURL:      server.URL,
		APIKey:       "k",
		APIKeyHeader: "X-API-Key",
	}, server.Client())
	if err != nil {
		t.Fatalf("new usps client: %v", err)
	}

	tracking := "9400 1000/ABC"
	_, err = client.GetTracking(context.Background(), tracking)
	if err != nil {
		t.Fatalf("get tracking: %v", err)
	}
	expectedSegment := "/tracking/" + url.PathEscape(tracking)
	if !strings.Contains(gotRequestURI, expectedSegment) {
		t.Fatalf("request uri = %q, expected segment %q", gotRequestURI, expectedSegment)
	}
}
