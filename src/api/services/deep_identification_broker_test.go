package services

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDeepIdentificationBroker_ConcurrentPublishSubscribeUnsubscribe covers
// T103/T104's concurrency requirement at the broker layer: many goroutines
// concurrently subscribing, publishing, and unsubscribing for the same
// jobID must never panic, deadlock, or lose track of subscriber counts
// (contracts/sse-events.md §4's "max 3 concurrent streams" cap depends on
// SubscriberCount being race-safe under exactly this kind of contention).
func TestDeepIdentificationBroker_ConcurrentPublishSubscribeUnsubscribe(t *testing.T) {
	b := NewDeepIdentificationBroker()
	const jobID = uint(1)
	const goroutines = 50
	const publishesPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Half the goroutines repeatedly subscribe/read-one/unsubscribe.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ch, unsubscribe := b.Subscribe(jobID)
			defer unsubscribe()
			select {
			case <-ch:
			case <-time.After(500 * time.Millisecond):
			}
		}()
	}
	// The other half hammer Publish concurrently.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < publishesPerGoroutine; j++ {
				b.Publish(jobID)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("broker concurrent publish/subscribe/unsubscribe deadlocked")
	}

	if got := b.SubscriberCount(jobID); got != 0 {
		t.Fatalf("expected all subscribers to have unsubscribed, SubscriberCount=%d", got)
	}
}

// TestDeepIdentificationBroker_PublishOnlyWakesMatchingJob proves Publish
// never cross-wakes a subscriber registered for a different jobID, which
// the SSE handler and pipeline runner both depend on for correct per-job
// isolation when many jobs are streaming concurrently.
func TestDeepIdentificationBroker_PublishOnlyWakesMatchingJob(t *testing.T) {
	b := NewDeepIdentificationBroker()
	chA, unsubA := b.Subscribe(1)
	defer unsubA()
	chB, unsubB := b.Subscribe(2)
	defer unsubB()

	b.Publish(1)

	select {
	case <-chA:
	default:
		t.Fatal("expected job 1's subscriber to be woken")
	}
	select {
	case <-chB:
		t.Fatal("job 2's subscriber must not be woken by a Publish for job 1")
	default:
	}
}

// TestDeepIdentificationBroker_PublishCoalescesWithoutBlocking proves a
// buffered-capacity-1 subscriber channel never blocks Publish even when
// many wakes arrive before the subscriber reads any of them (T104's
// "gap-free sequencing under load" depends on Publish never blocking the
// producer side, since the producer is the same goroutine appending
// events in sequence).
func TestDeepIdentificationBroker_PublishCoalescesWithoutBlocking(t *testing.T) {
	b := NewDeepIdentificationBroker()
	_, unsubscribe := b.Subscribe(1)
	defer unsubscribe()

	var publishCount int64
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10_000; i++ {
			b.Publish(1)
			atomic.AddInt64(&publishCount, 1)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Publish blocked on a full subscriber channel after %d of 10000 calls", atomic.LoadInt64(&publishCount))
	}
}
