package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestPushoverService(t *testing.T, captured *url.Values) (*PushoverService, func()) {
	t.Helper()

	settingsSvc, _ := newTestSettingsService(t)
	if err := settingsSvc.SetSetting(SettingPushoverAppToken, "app-token"); err != nil {
		t.Fatalf("failed to set pushover token: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		*captured = r.PostForm
		w.WriteHeader(http.StatusOK)
	}))

	svc := NewPushoverService(settingsSvc, NewLogger(10))
	svc.httpClient = server.Client()
	svc.apiURL = server.URL

	return svc, server.Close
}

func TestPushoverServiceSendMessage_CoinOfDayUsesHTMLAndAbsoluteCoinLink(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	message := buildCoinOfDayPushoverMessage("Coin of the Day", 42, "<Rare & Coin>", "Summary with <script>bad</script>", "https://coins.example.com/")
	message.UserKey = "user-key"

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("html"); got != "1" {
		t.Fatalf("html form field = %q, want 1", got)
	}
	if got := captured.Get("url"); got != "https://coins.example.com/coin/42" {
		t.Fatalf("url form field = %q, want absolute coin URL", got)
	}

	body := captured.Get("message")
	if !strings.Contains(body, `<b>&lt;Rare &amp; Coin&gt;</b>`) {
		t.Fatalf("message did not escape and bold coin name: %q", body)
	}
	if !strings.Contains(body, `<a href="https://coins.example.com/coin/42">Open coin</a>`) {
		t.Fatalf("message did not include coin link: %q", body)
	}
	if strings.Contains(body, "<script>") {
		t.Fatalf("message contains unescaped script tag: %q", body)
	}
}

func TestPushoverServiceSendMessage_CoinOfDayOmitsBrokenRelativeCoinLinkWhenUnconfigured(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	message := buildCoinOfDayPushoverMessage("Coin of the Day", 42, "Rare Coin", "Summary", "")
	message.UserKey = "user-key"

	if err := svc.SendMessage(message); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if got := captured.Get("html"); got != "1" {
		t.Fatalf("html form field = %q, want 1", got)
	}
	if _, ok := captured["url"]; ok {
		t.Fatalf("url form field should be omitted when public app URL is unconfigured")
	}

	body := captured.Get("message")
	if !strings.Contains(body, `<b>Rare Coin</b>`) {
		t.Fatalf("message did not keep HTML formatting: %q", body)
	}
	if strings.Contains(body, `href="/coin/42"`) || strings.Contains(body, `/coin/42`) {
		t.Fatalf("message contains broken relative coin link: %q", body)
	}
}

func TestPushoverServiceSendNotification_RemainsPlain(t *testing.T) {
	var captured url.Values
	svc, cleanup := newTestPushoverService(t, &captured)
	defer cleanup()

	if err := svc.SendNotification("user-key", "Ancient Coins", "Pushover notifications are working!", ""); err != nil {
		t.Fatalf("SendNotification() error = %v", err)
	}

	if got := captured.Get("html"); got != "" {
		t.Fatalf("html form field = %q, want empty for plain notification", got)
	}
	if got := captured.Get("message"); got != "Pushover notifications are working!" {
		t.Fatalf("message form field = %q, want plain text", got)
	}
	if _, ok := captured["url"]; ok {
		t.Fatalf("url form field should be omitted when no reference URL is provided")
	}
}
