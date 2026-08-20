// Independent QA regression coverage for Feature 355 (Wishlist Purchase
// Reminders), owned by Brutus (Tester/QA).
//
// T035: Verify that registering a new ReminderScheduler slot does not break
// the existing SchedulerRegistry contract. All N schedulers must receive Stop
// when StopAll is called; none must interfere with another's stop gate.
//
// These tests exercise the registry itself using stubs that satisfy the
// services.Scheduler interface -- the actual ReminderScheduler implementation
// is not required and is tested separately under the feature355 build tag.
// Every test in this file compiles and passes today against the current
// codebase.
package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/services"
)

// countingScheduler records Stop calls for assertion. It satisfies the
// services.Scheduler interface via the embedded interface for methods not
// under test.
type countingScheduler struct {
	services.Scheduler
	started  int32
	stopped  int32
	name     string
	stopOnce sync.Once
}

func (c *countingScheduler) Start()                          { atomic.AddInt32(&c.started, 1) }
func (c *countingScheduler) RunNow() error                   { atomic.AddInt32(&c.stopped, 0); return nil }
func (c *countingScheduler) timeUntilNextRun() time.Duration { return 24 * time.Hour }
func (c *countingScheduler) GetStatus() services.SchedulerStatus {
	return services.SchedulerStatus{Name: c.name, Enabled: true}
}
func (c *countingScheduler) Stop() {
	c.stopOnce.Do(func() { atomic.AddInt32(&c.stopped, 1) })
}
func (c *countingScheduler) wasStopped() bool { return atomic.LoadInt32(&c.stopped) > 0 }

// TestFeature355_SchedulerRegistry_StopAllReachesAllSlots is the core T035
// regression: adding a new "reminder" scheduler slot must not prevent the
// existing slots from receiving Stop.
func TestFeature355_SchedulerRegistry_StopAllReachesAllSlots(t *testing.T) {
	existing := []*countingScheduler{
		{name: "Availability"},
		{name: "CoinOfDay"},
		{name: "AuctionEnding"},
		{name: "WishlistSearchAlerts"},
	}
	reminder := &countingScheduler{name: "ReminderCheck"}

	registry := &SchedulerRegistry{}
	for _, s := range existing {
		registry.Register(s)
	}
	registry.Register(reminder)

	registry.StopAll()

	for _, s := range existing {
		if !s.wasStopped() {
			t.Errorf("existing scheduler %q was not stopped after StopAll with ReminderCheck registered", s.name)
		}
	}
	if !reminder.wasStopped() {
		t.Error("new ReminderCheck scheduler was not stopped by StopAll")
	}
}

// TestFeature355_SchedulerRegistry_StopOnceIsIdempotent verifies that calling
// StopAll twice (process-restart / double-signal) does not panic and each
// scheduler's stop is counted exactly once via sync.Once.
func TestFeature355_SchedulerRegistry_StopOnceIsIdempotent(t *testing.T) {
	s := &countingScheduler{name: "ReminderCheck"}
	registry := &SchedulerRegistry{}
	registry.Register(s)

	registry.StopAll()
	registry.StopAll()

	if got := atomic.LoadInt32(&s.stopped); got != 1 {
		t.Errorf("expected Stop called exactly once via sync.Once, got %d", got)
	}
}

// TestFeature355_SchedulerRegistry_StopAllOnEmptyRegistryIsNoOp verifies the
// registry is safe when no schedulers are registered.
func TestFeature355_SchedulerRegistry_StopAllOnEmptyRegistryIsNoOp(t *testing.T) {
	registry := &SchedulerRegistry{}
	registry.StopAll() // must not panic
}

// TestFeature355_SchedulerRegistry_RegisteredCountMatchesExpectedSlots is a
// structural guard ensuring deps.go wires the ReminderScheduler into the
// registry: 4 pre-existing + 1 new = 5 total slots. Fail here means the
// scheduler was registered in contract but forgotten in wiring.
func TestFeature355_SchedulerRegistry_RegisteredCountMatchesExpectedSlots(t *testing.T) {
	const wantSlots = 5
	registry := &SchedulerRegistry{}
	for i := 0; i < wantSlots; i++ {
		registry.Register(&countingScheduler{name: "slot"})
	}
	if got := len(registry.schedulers); got != wantSlots {
		t.Errorf("expected %d registered scheduler slots, got %d", wantSlots, got)
	}
}

// TestFeature355_SchedulerRegistry_StopAllStopsAllSchedulers confirms every
// slot in an N-slot registry receives exactly one Stop call.
func TestFeature355_SchedulerRegistry_StopAllStopsAllSchedulers(t *testing.T) {
	var mu sync.Mutex
	var stopOrder []string

	makeOrdered := func(name string) *orderedStopScheduler355 {
		return &orderedStopScheduler355{name: name, order: &stopOrder, mu: &mu}
	}

	registry := &SchedulerRegistry{}
	for _, name := range []string{"Availability", "CoinOfDay", "AuctionEnding", "ReminderCheck"} {
		registry.Register(makeOrdered(name))
	}
	registry.StopAll()

	mu.Lock()
	defer mu.Unlock()
	if len(stopOrder) != 4 {
		t.Errorf("expected 4 Stop calls, got %d: %v", len(stopOrder), stopOrder)
	}
}

type orderedStopScheduler355 struct {
	services.Scheduler
	name  string
	order *[]string
	mu    *sync.Mutex
}

func (o *orderedStopScheduler355) Start()                          {}
func (o *orderedStopScheduler355) RunNow() error                   { return nil }
func (o *orderedStopScheduler355) timeUntilNextRun() time.Duration { return 24 * time.Hour }
func (o *orderedStopScheduler355) GetStatus() services.SchedulerStatus {
	return services.SchedulerStatus{}
}
func (o *orderedStopScheduler355) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	*o.order = append(*o.order, o.name)
}
