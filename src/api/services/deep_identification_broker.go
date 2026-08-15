package services

import "sync"

// DeepIdentificationBroker is a minimal in-process pub/sub "wake" registry
// used to fan out live progress to connected SSE readers (T095,
// contracts/sse-events.md). It deliberately carries no event payloads: a
// Publish(jobID) call only wakes every subscriber currently waiting on
// that job's live tail, and each subscriber then re-reads the
// authoritative state from DeepIdentificationRepository
// (ListEventsSince/GetJob). This keeps the broker unable to ever desync
// from storage, drop a payload, or race with the durable event log -
// repository.SettleTerminal's own transactional, conditional UPDATE
// remains the single source of truth for "exactly one terminal event per
// job" (FR-019); the broker only decides *when* a subscriber should look
// again, and mirrors the existing in-memory cancel-registry map pattern
// (DeepIdentificationService.cancels, registerCancel/unregisterCancel).
type DeepIdentificationBroker struct {
	mu   sync.Mutex
	subs map[uint]map[chan struct{}]struct{}
}

// NewDeepIdentificationBroker constructs an empty broker. One instance is
// owned by DeepIdentificationService and shared with the pipeline runner
// (so provider/progress events wake subscribers as they are persisted) and
// with the SSE handler (so it can Subscribe/SubscriberCount).
func NewDeepIdentificationBroker() *DeepIdentificationBroker {
	return &DeepIdentificationBroker{subs: make(map[uint]map[chan struct{}]struct{})}
}

// Subscribe registers a new listener for jobID's live tail. It returns a
// buffered (capacity 1) wake channel - a caller should treat any receive
// on it (or timeout/ping tick) as "check ListEventsSince again" - plus an
// unsubscribe func the caller MUST invoke (typically via defer) once it
// stops reading, so SubscriberCount and memory stay accurate.
func (b *DeepIdentificationBroker) Subscribe(jobID uint) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	b.mu.Lock()
	if b.subs[jobID] == nil {
		b.subs[jobID] = make(map[chan struct{}]struct{})
	}
	b.subs[jobID][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if set, ok := b.subs[jobID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(b.subs, jobID)
			}
		}
	}
	return ch, unsubscribe
}

// Publish wakes every current subscriber for jobID. It never blocks: if a
// subscriber's channel already has a pending wake, the send is dropped
// (default case) since the coalesced wake still results in the same "read
// everything new since lastSeq" behavior on the subscriber side.
func (b *DeepIdentificationBroker) Publish(jobID uint) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[jobID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// SubscriberCount reports the current number of live subscribers for
// jobID. The SSE handler uses this to enforce the "max 3 concurrent
// streams per job per owner" cap (contracts/sse-events.md §4) before
// calling Subscribe.
func (b *DeepIdentificationBroker) SubscriberCount(jobID uint) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs[jobID])
}
