package handlers

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/gin-gonic/gin"
)

var (
	allowedPrefThemes = []string{"dark", "light", "british-museum", "louvre", "capitoline", "byzantine", "modern-greek"}
	allowedPrefViews  = []string{"swipe", "grid"}
	allowedPrefSorts  = []string{"updated_at_desc", "created_at_desc", "created_at_asc", "current_value_desc", "current_value_asc", "random_desc"}
	allowedPrefFelts  = []string{"red", "green", "navy"}
)

// userPreferencesRequest is the partial preference update accepted by
// PUT /user/profile. A nil field is left unchanged; an empty string clears it.
type userPreferencesRequest struct {
	Theme         *string `json:"theme"`
	Timezone      *string `json:"timezone"`
	DefaultView   *string `json:"defaultView"`
	DefaultSort   *string `json:"defaultSort"`
	TrayFeltColor *string `json:"trayFeltColor"`
}

func (r *userPreferencesRequest) apply(updates map[string]interface{}) error {
	enum := func(field, column string, value *string, allowed []string) error {
		if value == nil {
			return nil
		}
		v := strings.TrimSpace(*value)
		if v != "" && !slices.Contains(allowed, v) {
			return fmt.Errorf("Invalid %s preference", field)
		}
		updates[column] = v
		return nil
	}
	if err := enum("theme", "pref_theme", r.Theme, allowedPrefThemes); err != nil {
		return err
	}
	if err := enum("defaultView", "pref_default_view", r.DefaultView, allowedPrefViews); err != nil {
		return err
	}
	if err := enum("defaultSort", "pref_default_sort", r.DefaultSort, allowedPrefSorts); err != nil {
		return err
	}
	if err := enum("trayFeltColor", "pref_tray_felt_color", r.TrayFeltColor, allowedPrefFelts); err != nil {
		return err
	}
	if r.Timezone != nil {
		tz := strings.TrimSpace(*r.Timezone)
		if tz != "" {
			if len(tz) > 64 {
				return fmt.Errorf("Invalid timezone preference")
			}
			if _, err := time.LoadLocation(tz); err != nil {
				return fmt.Errorf("Invalid timezone preference")
			}
		}
		updates["pref_timezone"] = tz
	}
	return nil
}

func userPreferencesPayload(user *models.User) gin.H {
	return gin.H{
		"theme":         user.PrefTheme,
		"timezone":      user.PrefTimezone,
		"defaultView":   user.PrefDefaultView,
		"defaultSort":   user.PrefDefaultSort,
		"trayFeltColor": user.PrefTrayFeltColor,
	}
}
