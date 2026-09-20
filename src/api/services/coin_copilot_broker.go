package services

import "sync"

type CoinCopilotBroker struct {
	mu   sync.Mutex
	subs map[string]map[chan struct{}]struct{}
	max  int
}

func NewCoinCopilotBroker(maxSubscribers int) *CoinCopilotBroker {
	if maxSubscribers < 1 {
		maxSubscribers = 3
	}
	return &CoinCopilotBroker{subs: make(map[string]map[chan struct{}]struct{}), max: maxSubscribers}
}

func (b *CoinCopilotBroker) Subscribe(runID string) (<-chan struct{}, func(), bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.subs[runID]) >= b.max {
		return nil, nil, false
	}
	ch := make(chan struct{}, 1)
	if b.subs[runID] == nil {
		b.subs[runID] = make(map[chan struct{}]struct{})
	}
	b.subs[runID][ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subs[runID], ch)
		if len(b.subs[runID]) == 0 {
			delete(b.subs, runID)
		}
	}, true
}

func (b *CoinCopilotBroker) Publish(runID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[runID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
