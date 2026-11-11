package handlers

import (
	"errors"
	"net/url"
	"strings"

	"github.com/briandenicola/ancient-coins-api/services"
)

const (
	maxPaperlessURLLength  = 500
	maxPaperlessTagsLength = 500
)

var errInvalidPaperlessURL = errors.New("Invalid paperless-ngx URL: must be an absolute http(s) URL")

// normalizePaperlessURL validates the user's paperless-ngx base URL and strips
// trailing slashes so the frontend can append paths directly. An empty value
// clears the setting.
func normalizePaperlessURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if len(value) > maxPaperlessURLLength {
		return "", errInvalidPaperlessURL
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errInvalidPaperlessURL
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errInvalidPaperlessURL
	}
	return strings.TrimRight(value, "/"), nil
}

// normalizePaperlessTags turns "coins, brian ,, shopping" into
// "coins, brian, shopping": trimmed, empty names dropped. An empty value
// clears the setting.
func normalizePaperlessTags(raw string) (string, error) {
	names := services.SplitPaperlessTags(raw)
	value := strings.Join(names, ", ")
	if len(value) > maxPaperlessTagsLength {
		return "", errors.New("paperless-ngx tags must be 500 characters or fewer")
	}
	return value, nil
}
