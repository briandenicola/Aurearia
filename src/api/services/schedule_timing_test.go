package services

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Skipf("time zone %s unavailable: %v", name, err)
	}
	return loc
}

func TestDailyScheduleSlotAfter(t *testing.T) {
	utc := time.UTC
	at := func(d, h, m int) time.Time { return time.Date(2026, 9, d, h, m, 0, 0, utc) }

	tests := []struct {
		name     string
		schedule dailySchedule
		after    time.Time
		want     time.Time
	}{
		{"daily later today", dailySchedule{Hour: 3, Location: utc}, at(28, 1, 0), at(28, 3, 0)},
		{"daily already passed", dailySchedule{Hour: 3, Location: utc}, at(28, 3, 0), at(29, 3, 0)},
		{"hourly from start", dailySchedule{Hour: 6, Interval: time.Hour, Location: utc}, at(28, 9, 30), at(28, 10, 0)},
		{"hourly before start uses previous day's grid", dailySchedule{Hour: 6, Interval: time.Hour, Location: utc}, at(28, 2, 30), at(28, 3, 0)},
		{"non-dividing interval restarts at start", dailySchedule{Hour: 6, Interval: 7 * time.Hour, Location: utc}, at(29, 4, 0), at(29, 6, 0)},
		{"day-long interval runs at start", dailySchedule{Hour: 6, Interval: 36 * time.Hour, Location: utc}, at(28, 7, 0), at(29, 6, 0)},
		{"tiny interval is clamped", dailySchedule{Hour: 0, Interval: time.Second, Location: utc}, at(28, 0, 0), at(28, 0, 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.schedule.slotAfter(tt.after); !got.Equal(tt.want) {
				t.Fatalf("slotAfter(%v) = %v, want %v", tt.after, got, tt.want)
			}
		})
	}
}

func TestDailyScheduleSlotAfterKeepsWallClockAcrossDST(t *testing.T) {
	chicago := mustLoadLocation(t, "America/Chicago")
	s := dailySchedule{Hour: 3, Location: chicago}

	// DST ends on 2026-11-01: the day is 25 hours long.
	got := s.slotAfter(time.Date(2026, 10, 31, 3, 0, 0, 0, chicago))
	if want := time.Date(2026, 11, 1, 3, 0, 0, 0, chicago); !got.Equal(want) {
		t.Fatalf("fall back: got %v, want %v", got, want)
	}
	// DST starts on 2026-03-08: the day is 23 hours long.
	got = s.slotAfter(time.Date(2026, 3, 7, 3, 0, 0, 0, chicago))
	if want := time.Date(2026, 3, 8, 3, 0, 0, 0, chicago); !got.Equal(want) {
		t.Fatalf("spring forward: got %v, want %v", got, want)
	}
}

func TestDailyScheduleNext(t *testing.T) {
	utc := time.UTC
	hourly := dailySchedule{Hour: 0, Interval: time.Hour, Location: utc}
	now := time.Date(2026, 9, 28, 10, 15, 0, 0, utc)

	if got, want := hourly.next(now, nil, true), time.Date(2026, 9, 28, 11, 0, 0, 0, utc); !got.Equal(want) {
		t.Fatalf("no history: got %v, want %v", got, want)
	}

	recent := time.Date(2026, 9, 28, 10, 0, 20, 0, utc)
	if got, want := hourly.next(now, &recent, true), time.Date(2026, 9, 28, 11, 0, 0, 0, utc); !got.Equal(want) {
		t.Fatalf("history: got %v, want %v", got, want)
	}

	missed := time.Date(2026, 9, 28, 8, 0, 20, 0, utc)
	if got := hourly.next(now, &missed, true); !got.Equal(now) {
		t.Fatalf("startup catch-up: got %v, want now", got)
	}
	if got, want := hourly.next(now, &missed, false), time.Date(2026, 9, 28, 11, 0, 0, 0, utc); !got.Equal(want) {
		t.Fatalf("running skip: got %v, want %v", got, want)
	}

	everyTwoDays := dailySchedule{Hour: 2, Interval: 48 * time.Hour, Location: utc}
	ran := time.Date(2026, 9, 28, 2, 0, 30, 0, utc)
	if got, want := everyTwoDays.next(now, &ran, false), time.Date(2026, 9, 30, 2, 0, 0, 0, utc); !got.Equal(want) {
		t.Fatalf("two-day interval: got %v, want %v", got, want)
	}
}

// TestDailyScheduleNextAppliesChangedStartTimeInZone is the #768 report: a run
// completed at 02:00 Chicago, then the start time was moved to 12:00 Chicago.
func TestDailyScheduleNextAppliesChangedStartTimeInZone(t *testing.T) {
	chicago := mustLoadLocation(t, "America/Chicago")
	s := dailySchedule{Hour: 12, Location: chicago}
	last := time.Date(2026, 9, 28, 7, 0, 30, 0, time.UTC) // 02:00 Chicago
	now := time.Date(2026, 9, 28, 14, 0, 0, 0, time.UTC)  // 09:00 Chicago

	want := time.Date(2026, 9, 28, 17, 0, 0, 0, time.UTC) // 12:00 Chicago
	if got := s.next(now, &last, false); !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseStartTime(t *testing.T) {
	tests := []struct {
		raw          string
		wantH, wantM int
	}{
		{"07:30", 7, 30},
		{" 23:59 ", 23, 59},
		{"", 4, 5},
		{"24:00", 4, 5},
		{"12:60", 4, 5},
		{"noon", 4, 5},
	}
	for _, tt := range tests {
		h, m := parseStartTime(tt.raw, 4, 5)
		if h != tt.wantH || m != tt.wantM {
			t.Errorf("parseStartTime(%q) = %d:%d, want %d:%d", tt.raw, h, m, tt.wantH, tt.wantM)
		}
	}
}

func newScheduleTimingSettings(t *testing.T) *SettingsService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewSettingsService(repository.NewSettingsRepository(db))
}

func TestScheduleLocationFallbackOrder(t *testing.T) {
	settings := newScheduleTimingSettings(t)
	logger := NewLogger(10)

	if got := scheduleLocation(settings, logger, SettingCoinOfDayTimezone); got != time.Local {
		t.Fatalf("empty settings: got %v, want server time", got)
	}

	if err := settings.SetSetting(SettingScheduleTimezone, "America/Chicago"); err != nil {
		t.Fatalf("set schedule zone: %v", err)
	}
	if got := scheduleLocation(settings, logger, SettingCoinOfDayTimezone); got.String() != "America/Chicago" {
		t.Fatalf("schedule zone: got %v", got)
	}

	if err := settings.SetSetting(SettingCoinOfDayTimezone, "Europe/London"); err != nil {
		t.Fatalf("set coin of day zone: %v", err)
	}
	if got := scheduleLocation(settings, logger, SettingCoinOfDayTimezone); got.String() != "Europe/London" {
		t.Fatalf("specific zone should win: got %v", got)
	}
	if got := scheduleLocation(settings, logger); got.String() != "America/Chicago" {
		t.Fatalf("other schedulers should ignore the coin of day zone: got %v", got)
	}
}

func TestScheduleLocationInvalidZoneUsesServerTime(t *testing.T) {
	settings := newScheduleTimingSettings(t)
	// Bypass SetSetting validation to mimic a legacy stored value.
	if err := settings.repo.Upsert(SettingScheduleTimezone, "Not/AZone"); err != nil {
		t.Fatalf("seed zone: %v", err)
	}
	if got := scheduleLocation(settings, NewLogger(10)); got != time.Local {
		t.Fatalf("invalid zone: got %v, want server time", got)
	}
}

func TestSetSettingRejectsInvalidScheduleTimezone(t *testing.T) {
	settings := newScheduleTimingSettings(t)
	if err := settings.SetSetting(SettingScheduleTimezone, "Not/AZone"); err == nil {
		t.Fatal("expected an invalid schedule time zone to be rejected")
	}
	if err := settings.SetSetting(SettingScheduleTimezone, ""); err != nil {
		t.Fatalf("empty zone should mean server time: %v", err)
	}
}

func TestScheduleLoopPicksUpChangedPlanWithinRecheck(t *testing.T) {
	stop := make(chan struct{})
	done := make(chan struct{})
	ran := make(chan struct{}, 4)
	var plan atomic.Int64
	plan.Store(time.Now().Add(time.Hour).UnixNano())

	go func() {
		defer close(done)
		scheduleLoop{
			category: "test",
			name:     "test job",
			logger:   NewLogger(50),
			stopCh:   stop,
			recheck:  20 * time.Millisecond,
			next: func(now time.Time, _ bool) time.Time {
				return time.Unix(0, plan.Load())
			},
			run: func() {
				plan.Store(time.Now().Add(time.Hour).UnixNano())
				ran <- struct{}{}
			},
		}.Run()
	}()

	select {
	case <-ran:
		t.Fatal("job ran before its planned time")
	case <-time.After(60 * time.Millisecond):
	}

	// Simulate an admin moving the start time to now.
	plan.Store(time.Now().UnixNano())
	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("changed schedule was not picked up within the recheck interval")
	}

	close(stop)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("loop did not stop")
	}
}

func TestScheduleLoopCatchUpOnlyOnFirstEvaluation(t *testing.T) {
	stop := make(chan struct{})
	done := make(chan struct{})
	var calls, catchUps atomic.Int32

	go func() {
		defer close(done)
		scheduleLoop{
			category: "test",
			name:     "test job",
			logger:   NewLogger(50),
			stopCh:   stop,
			recheck:  5 * time.Millisecond,
			next: func(now time.Time, catchUp bool) time.Time {
				calls.Add(1)
				if catchUp {
					catchUps.Add(1)
				}
				return now.Add(time.Hour)
			},
			run: func() {},
		}.Run()
	}()

	deadline := time.Now().Add(time.Second)
	for calls.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	close(stop)
	<-done

	if calls.Load() < 3 {
		t.Fatalf("expected repeated evaluations, got %d", calls.Load())
	}
	if catchUps.Load() != 1 {
		t.Fatalf("catchUp passed %d times, want exactly 1", catchUps.Load())
	}
}
