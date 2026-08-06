package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

const (
	defaultParcelAppBaseURL = "https://api.parcel.app/external"
	parcelAppCarrierCode    = "pholder"
)

type ParcelAppClient interface {
	ListDeliveries(ctx context.Context, apiKey string) ([]ParcelAppDelivery, error)
	AddDelivery(ctx context.Context, apiKey, trackingNumber, description string) error
}

type HTTPParcelAppClient struct {
	baseURL string
	client  *http.Client
}

type ParcelAppError struct {
	Operation    string
	StatusCode   int
	ErrorMessage string
	BodyExcerpt  string
}

func (e *ParcelAppError) Error() string {
	message := strings.TrimSpace(e.ErrorMessage)
	if message == "" {
		message = "no ParcelApp error message"
	}
	if e.BodyExcerpt != "" {
		return fmt.Sprintf("ParcelApp %s failed: status=%d error=%q body=%q", e.Operation, e.StatusCode, message, e.BodyExcerpt)
	}
	return fmt.Sprintf("ParcelApp %s failed: status=%d error=%q", e.Operation, e.StatusCode, message)
}

func NewHTTPParcelAppClient() *HTTPParcelAppClient {
	return &HTTPParcelAppClient{
		baseURL: defaultParcelAppBaseURL,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

type parcelAppListResponse struct {
	Success                bool                `json:"success"`
	UnderscoreSuccess      bool                `json:"_success"`
	ErrorMessage           string              `json:"error_message"`
	UnderscoreErrorMessage string              `json:"_error_message"`
	Deliveries             []ParcelAppDelivery `json:"deliveries"`
}

type parcelAppAddResponse struct {
	Success                bool   `json:"success"`
	UnderscoreSuccess      bool   `json:"_success"`
	ErrorMessage           string `json:"error_message"`
	UnderscoreErrorMessage string `json:"_error_message"`
}

type ParcelAppDelivery struct {
	CarrierCode          string           `json:"carrier_code"`
	Description          string           `json:"description"`
	StatusCode           int              `json:"status_code"`
	TrackingNumber       string           `json:"tracking_number"`
	Events               []ParcelAppEvent `json:"events"`
	DateExpected         string           `json:"date_expected"`
	TimestampExpected    int64            `json:"timestamp_expected"`
	TimestampExpectedEnd int64            `json:"timestamp_expected_end"`
}

type ParcelAppEvent struct {
	Event      string `json:"event"`
	Date       string `json:"date"`
	Location   string `json:"location"`
	Additional string `json:"additional"`
}

func (c *HTTPParcelAppClient) ListDeliveries(ctx context.Context, apiKey string) ([]ParcelAppDelivery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.baseURL, "/")+"/deliveries/?filter_mode=recent", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, parcelAPIError("list deliveries", resp.StatusCode, "", body)
	}

	body, _ := io.ReadAll(resp.Body)
	var parsed parcelAppListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode parcel deliveries: %w; body=%q", err, responseExcerpt(body))
	}
	if !parcelResponseSuccess(parsed.Success, parsed.UnderscoreSuccess) {
		return nil, parcelAPIError("list deliveries", resp.StatusCode, parcelResponseError(parsed.ErrorMessage, parsed.UnderscoreErrorMessage), body)
	}
	return parsed.Deliveries, nil
}

func (c *HTTPParcelAppClient) AddDelivery(ctx context.Context, apiKey, trackingNumber, description string) error {
	payload := map[string]interface{}{
		"tracking_number":        trackingNumber,
		"carrier_code":           parcelAppCarrierCode,
		"description":            description,
		"send_push_confirmation": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/add-delivery/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parcelAPIError("add delivery", resp.StatusCode, "", respBody)
	}

	var parsed parcelAppAddResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("decode parcel add delivery: %w; body=%q", err, responseExcerpt(respBody))
	}
	if !parcelResponseSuccess(parsed.Success, parsed.UnderscoreSuccess) {
		return parcelAPIError("add delivery", resp.StatusCode, parcelResponseError(parsed.ErrorMessage, parsed.UnderscoreErrorMessage), respBody)
	}
	return nil
}

func parcelResponseSuccess(success bool, underscoreSuccess bool) bool {
	return success || underscoreSuccess
}

func parcelResponseError(errorMessage string, underscoreErrorMessage string) string {
	if strings.TrimSpace(errorMessage) != "" {
		return errorMessage
	}
	return underscoreErrorMessage
}

func parcelAPIError(operation string, statusCode int, errorMessage string, body []byte) error {
	return &ParcelAppError{
		Operation:    operation,
		StatusCode:   statusCode,
		ErrorMessage: strings.TrimSpace(errorMessage),
		BodyExcerpt:  responseExcerpt(body),
	}
}

func responseExcerpt(body []byte) string {
	excerpt := strings.Join(strings.Fields(string(body)), " ")
	if len(excerpt) > 300 {
		return excerpt[:300]
	}
	return excerpt
}

func parcelDeliveryToSnapshot(delivery ParcelAppDelivery) ShipmentTrackingSnapshot {
	status := parcelStatusCodeToShipmentStatus(delivery.StatusCode)
	snapshot := ShipmentTrackingSnapshot{
		Carrier:             models.ShipmentCarrierParcel,
		TrackingNumber:      strings.TrimSpace(delivery.TrackingNumber),
		CurrentStatus:       status,
		CurrentStatusSource: models.ShipmentStatusSourceAPI,
		EstimatedDeliveryAt: parcelExpectedTime(delivery),
	}
	if status == models.ShipmentStatusDelivered {
		if deliveredAt := parcelLatestEventTime(delivery.Events); deliveredAt != nil {
			snapshot.DeliveredAt = deliveredAt
		}
	}
	for idx, event := range delivery.Events {
		occurredAt := parseParcelEventTime(event.Date)
		if occurredAt == nil {
			now := time.Now().UTC()
			occurredAt = &now
		}
		description := strings.TrimSpace(event.Event)
		if additional := strings.TrimSpace(event.Additional); additional != "" {
			if description != "" {
				description += " - "
			}
			description += additional
		}
		snapshot.Events = append(snapshot.Events, ShipmentTrackingEvent{
			EventKey:     fmt.Sprintf("parcel:%s:%d:%s", snapshot.TrackingNumber, idx, event.Date),
			Status:       status,
			StatusSource: models.ShipmentStatusSourceAPI,
			OccurredAt:   *occurredAt,
			Location:     strings.TrimSpace(event.Location),
			Description:  description,
			RawStatus:    fmt.Sprintf("%d", delivery.StatusCode),
		})
	}
	return snapshot
}

func parcelStatusCodeToShipmentStatus(code int) models.ShipmentStatus {
	switch code {
	case 0:
		return models.ShipmentStatusDelivered
	case 2:
		return models.ShipmentStatusInTransit
	case 3:
		return models.ShipmentStatusPending
	case 4:
		return models.ShipmentStatusOutForDelivery
	case 6, 7:
		return models.ShipmentStatusException
	case 8:
		return models.ShipmentStatusLabelCreated
	case 1, 5:
		return models.ShipmentStatusUnknown
	default:
		return models.ShipmentStatusUnknown
	}
}

func parcelExpectedTime(delivery ParcelAppDelivery) *time.Time {
	if delivery.TimestampExpected > 0 {
		t := time.Unix(delivery.TimestampExpected, 0).UTC()
		return &t
	}
	return parseParcelEventTime(delivery.DateExpected)
}

func parcelLatestEventTime(events []ParcelAppEvent) *time.Time {
	for _, event := range events {
		if t := parseParcelEventTime(event.Date); t != nil {
			return t
		}
	}
	return nil
}

func parseParcelEventTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"Jan 2, 2006 15:04",
		"Jan 2, 2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}
