package main

import (
	"testing"
	"time"
)

// TestTimezoneEmbed_IANAZonesAvailable checks that representative non-UTC IANA
// zones resolve correctly in the production binary's embedded timezone data.
// The test intentionally does NOT import time/tzdata itself; it relies on the
// blank import in main.go. If that import is ever removed, this test will fail
// on any host without /usr/share/zoneinfo (e.g., the Alpine production image),
// catching the regression before it reaches production.
func TestTimezoneEmbed_IANAZonesAvailable(t *testing.T) {
	zones := []string{
		"America/Chicago",
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Europe/Paris",
		"Asia/Tokyo",
		"Australia/Sydney",
		"Pacific/Auckland",
	}
	for _, tz := range zones {
		if _, err := time.LoadLocation(tz); err != nil {
			t.Errorf("time.LoadLocation(%q) failed: %v -- time/tzdata may have been removed from main.go", tz, err)
		}
	}
}

// TestTimezoneEmbed_InvalidZoneRejected verifies that strict rejection of
// genuinely invalid timezone strings is preserved after embedding tzdata.
// Embedding the IANA database must not weaken input validation.
func TestTimezoneEmbed_InvalidZoneRejected(t *testing.T) {
	invalid := []string{
		"Not/A/Timezone",
		"America/Fakecity",
		"Bogus",
	}
	for _, tz := range invalid {
		if _, err := time.LoadLocation(tz); err == nil {
			t.Errorf("time.LoadLocation(%q) unexpectedly succeeded; invalid zones must still be rejected", tz)
		}
	}
}
