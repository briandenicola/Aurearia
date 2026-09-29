package services

import (
	"fmt"
	"strings"
	"time"
)

// schedulerRecheckInterval bounds each scheduler sleep so a changed start
// time, interval or time zone takes effect within a minute, not after the
// run that was already planned.
const schedulerRecheckInterval = time.Minute

// minScheduleStep guards the slot walk against a zero or tiny interval.
const minScheduleStep = time.Minute

// dailySchedule describes "start at HH:MM in a zone, then repeat every
// Interval". Sub-day intervals restart from HH:MM each day; intervals of a day
// or longer run at HH:MM every whole number of days.
type dailySchedule struct {
	Hour     int
	Minute   int
	Interval time.Duration // 0 = once a day
	Location *time.Location
}

func (d dailySchedule) location() *time.Location {
	if d.Location == nil {
		return time.Local
	}
	return d.Location
}

func (d dailySchedule) wholeDays() int {
	if d.Interval < 24*time.Hour {
		return 1
	}
	return int(d.Interval / (24 * time.Hour))
}

// slotAfter returns the first scheduled slot strictly after t. Days are built
// from the calendar date in the zone, so HH:MM keeps its wall-clock time
// across DST changes.
func (d dailySchedule) slotAfter(t time.Time) time.Time {
	loc := d.location()
	local := t.In(loc)
	y, mo, day := local.Date()
	step := d.Interval
	if step >= 24*time.Hour {
		step = 0
	} else if step > 0 && step < minScheduleStep {
		step = minScheduleStep
	}
	for offset := -1; ; offset++ {
		start := time.Date(y, mo, day+offset, d.Hour, d.Minute, 0, 0, loc)
		if step == 0 {
			if start.After(t) {
				return start
			}
			continue
		}
		end := time.Date(y, mo, day+offset+1, d.Hour, d.Minute, 0, 0, loc)
		for slot := start; slot.Before(end); slot = slot.Add(step) {
			if slot.After(t) {
				return slot
			}
		}
	}
}

// next returns when the job should run next.
//
// With no completed run it is the first slot after now. Otherwise it is the
// first slot after the last completed run (plus any extra whole days for
// multi-day intervals). If that slot has already passed, catchUp (used once at
// startup) runs immediately to cover downtime; otherwise the missed slot is
// skipped, so moving the start time earlier never triggers a surprise run.
func (d dailySchedule) next(now time.Time, lastCompleted *time.Time, catchUp bool) time.Time {
	if lastCompleted == nil || lastCompleted.IsZero() {
		return d.slotAfter(now)
	}
	ref := *lastCompleted
	if days := d.wholeDays(); days > 1 {
		ref = ref.In(d.location()).AddDate(0, 0, days-1)
	}
	slot := d.slotAfter(ref)
	if slot.After(now) {
		return slot
	}
	if catchUp {
		return now
	}
	return d.slotAfter(now)
}

// parseStartTime reads an "HH:MM" setting, falling back to the default when
// it is empty or out of range.
func parseStartTime(raw string, defaultHour, defaultMinute int) (int, int) {
	var h, m int
	if _, err := fmt.Sscanf(strings.TrimSpace(raw), "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return defaultHour, defaultMinute
	}
	return h, m
}

// scheduleLocation resolves the first non-empty zone setting among keys and
// then the app-wide ScheduleTimezone, falling back to server time.
func scheduleLocation(settingsSvc *SettingsService, logger *Logger, keys ...string) *time.Location {
	for _, key := range append(keys, SettingScheduleTimezone) {
		name := strings.TrimSpace(settingsSvc.GetSetting(key))
		if name == "" {
			continue
		}
		loc, err := time.LoadLocation(name)
		if err != nil {
			if logger != nil {
				logger.Warn("scheduler", "Unknown time zone %q in %s, using server time", name, key)
			}
			return time.Local
		}
		return loc
	}
	return time.Local
}

// scheduleLoop runs a job at the times returned by next, re-checking at least
// every recheck interval so setting changes apply without a restart.
type scheduleLoop struct {
	category     string
	name         string
	logger       *Logger
	stopCh       <-chan struct{}
	initialDelay time.Duration
	recheck      time.Duration
	// next receives catchUp=true only on the first evaluation after startup.
	next func(now time.Time, catchUp bool) time.Time
	run  func()
}

func (l scheduleLoop) Run() {
	if l.initialDelay > 0 {
		select {
		case <-time.After(l.initialDelay):
		case <-l.stopCh:
			return
		}
	}
	recheck := l.recheck
	if recheck <= 0 {
		recheck = schedulerRecheckInterval
	}

	var announced time.Time
	catchUp := true
	for {
		now := time.Now()
		next := l.next(now, catchUp)
		catchUp = false
		if !next.Equal(announced) {
			l.logger.Info(l.category, "Next %s at %s (in %s)", l.name, next.Format(time.RFC3339), next.Sub(now).Round(time.Second))
			announced = next
		}

		wait := next.Sub(now)
		due := wait <= recheck
		if !due {
			wait = recheck
		}
		if wait < 0 {
			wait = 0
		}

		select {
		case <-time.After(wait):
		case <-l.stopCh:
			l.logger.Info(l.category, "%s scheduler stopped", l.name)
			return
		}

		if due {
			l.run()
		}
	}
}
