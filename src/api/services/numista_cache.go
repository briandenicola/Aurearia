package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type NumistaClock interface {
	Now() time.Time
}

type realNumistaClock struct{}

func (realNumistaClock) Now() time.Time { return time.Now().UTC() }

func NewSystemNumistaClock() NumistaClock { return realNumistaClock{} }

type numistaSearchCacheEntry struct {
	value     []models.NumistaCandidate
	createdAt time.Time
	expiresAt time.Time
}

type numistaDetailCacheEntry struct {
	value     models.NumistaCandidate
	createdAt time.Time
	expiresAt time.Time
}

type numistaSearchCall struct {
	done     chan struct{}
	cancel   context.CancelFunc
	waiters  int
	value    []models.NumistaCandidate
	metadata *models.NumistaCacheMetadata
	err      error
}

type numistaDetailCall struct {
	done     chan struct{}
	cancel   context.CancelFunc
	waiters  int
	value    models.NumistaCandidate
	metadata *models.NumistaCacheMetadata
	err      error
}

type NumistaCache struct {
	mu             sync.Mutex
	clock          NumistaClock
	searchMax      int
	detailMax      int
	search         map[string]numistaSearchCacheEntry
	detail         map[string]numistaDetailCacheEntry
	searchInflight map[string]*numistaSearchCall
	detailInflight map[string]*numistaDetailCall
}

func NewNumistaCache(clock NumistaClock, searchMax, detailMax int) *NumistaCache {
	if clock == nil {
		clock = realNumistaClock{}
	}
	if searchMax <= 0 {
		searchMax = 500
	}
	if detailMax <= 0 {
		detailMax = 5000
	}
	return &NumistaCache{
		clock: clock, searchMax: searchMax, detailMax: detailMax,
		search:         make(map[string]numistaSearchCacheEntry),
		detail:         make(map[string]numistaDetailCacheEntry),
		searchInflight: make(map[string]*numistaSearchCall),
		detailInflight: make(map[string]*numistaDetailCall),
	}
}

func NumistaSearchCacheKey(query string, limit int) string {
	return hashNumistaIdentity("search|" + NormalizeNumistaText(query) + "|" + strconv.Itoa(limit))
}

func NumistaDetailCacheKey(id int) string {
	return hashNumistaIdentity("detail|" + strconv.Itoa(id))
}

func hashNumistaIdentity(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (c *NumistaCache) GetSearch(query string, limit int) ([]models.NumistaCandidate, *models.NumistaCacheMetadata, bool) {
	key := NumistaSearchCacheKey(query, limit)
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getSearchLocked(key, c.clock.Now())
}

func (c *NumistaCache) getSearchLocked(
	key string,
	now time.Time,
) ([]models.NumistaCandidate, *models.NumistaCacheMetadata, bool) {
	entry, ok := c.search[key]
	if !ok {
		return nil, nil, false
	}
	if !now.Before(entry.expiresAt) {
		delete(c.search, key)
		return nil, nil, false
	}
	return cloneCandidates(entry.value), cacheMetadata(now, entry.createdAt, entry.expiresAt, true), true
}

func (c *NumistaCache) SetSearch(query string, limit int, value []models.NumistaCandidate, ttl time.Duration) *models.NumistaCacheMetadata {
	key := NumistaSearchCacheKey(query, limit)
	now := c.clock.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setSearchLocked(key, value, ttl, now)
}

func (c *NumistaCache) setSearchLocked(
	key string,
	value []models.NumistaCandidate,
	ttl time.Duration,
	now time.Time,
) *models.NumistaCacheMetadata {
	entry := numistaSearchCacheEntry{value: cloneCandidates(value), createdAt: now, expiresAt: now.Add(ttl)}
	c.removeExpiredLocked(now)
	if _, exists := c.search[key]; !exists && len(c.search) >= c.searchMax {
		c.evictOldestSearchLocked()
	}
	c.search[key] = entry
	return cacheMetadata(now, entry.createdAt, entry.expiresAt, false)
}

func (c *NumistaCache) DoSearch(
	ctx context.Context,
	query string,
	limit int,
	ttl time.Duration,
	load func(context.Context) ([]models.NumistaCandidate, error),
) ([]models.NumistaCandidate, *models.NumistaCacheMetadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	key := NumistaSearchCacheKey(query, limit)
	c.mu.Lock()
	now := c.clock.Now()
	if cached, metadata, ok := c.getSearchLocked(key, now); ok {
		c.mu.Unlock()
		return cached, metadata, nil
	}
	if call, ok := c.searchInflight[key]; ok {
		call.waiters++
		c.mu.Unlock()
		return c.waitSearch(ctx, key, call)
	}
	loadCtx, cancel := context.WithCancel(context.Background())
	call := &numistaSearchCall{done: make(chan struct{}), cancel: cancel, waiters: 1}
	c.searchInflight[key] = call
	c.mu.Unlock()

	go c.runSearchLoad(key, call, ttl, loadCtx, load)
	return c.waitSearch(ctx, key, call)
}

func (c *NumistaCache) waitSearch(
	ctx context.Context,
	key string,
	call *numistaSearchCall,
) ([]models.NumistaCandidate, *models.NumistaCacheMetadata, error) {
	select {
	case <-ctx.Done():
		c.mu.Lock()
		if current, ok := c.searchInflight[key]; ok && current == call {
			call.waiters--
			if call.waiters == 0 {
				delete(c.searchInflight, key)
				call.cancel()
			}
		}
		c.mu.Unlock()
		return nil, nil, ctx.Err()
	case <-call.done:
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		return cloneCandidates(call.value), cloneCacheMetadata(call.metadata), call.err
	}
}

func (c *NumistaCache) runSearchLoad(
	key string,
	call *numistaSearchCall,
	ttl time.Duration,
	ctx context.Context,
	load func(context.Context) ([]models.NumistaCandidate, error),
) {
	defer call.cancel()
	value, err := load(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, ok := c.searchInflight[key]; !ok || current != call {
		call.value, call.err = cloneCandidates(value), err
		close(call.done)
		return
	}
	var metadata *models.NumistaCacheMetadata
	if err == nil {
		now := c.clock.Now()
		metadata = c.setSearchLocked(key, value, ttl, now)
	}
	call.value, call.metadata, call.err = cloneCandidates(value), cloneCacheMetadata(metadata), err
	delete(c.searchInflight, key)
	close(call.done)
}

func (c *NumistaCache) GetDetail(id int) (models.NumistaCandidate, *models.NumistaCacheMetadata, bool) {
	key := NumistaDetailCacheKey(id)
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getDetailLocked(key, c.clock.Now())
}

func (c *NumistaCache) getDetailLocked(
	key string,
	now time.Time,
) (models.NumistaCandidate, *models.NumistaCacheMetadata, bool) {
	entry, ok := c.detail[key]
	if !ok {
		return models.NumistaCandidate{}, nil, false
	}
	if !now.Before(entry.expiresAt) {
		delete(c.detail, key)
		return models.NumistaCandidate{}, nil, false
	}
	value := entry.value
	value.EnrichmentState = models.NumistaEnrichmentCached
	return value, cacheMetadata(now, entry.createdAt, entry.expiresAt, true), true
}

func (c *NumistaCache) SetDetail(id int, value models.NumistaCandidate, ttl time.Duration) {
	key := NumistaDetailCacheKey(id)
	now := c.clock.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setDetailLocked(key, value, ttl, now)
}

func (c *NumistaCache) setDetailLocked(
	key string,
	value models.NumistaCandidate,
	ttl time.Duration,
	now time.Time,
) {
	c.removeExpiredLocked(now)
	if _, exists := c.detail[key]; !exists && len(c.detail) >= c.detailMax {
		c.evictOldestDetailLocked()
	}
	c.detail[key] = numistaDetailCacheEntry{value: value, createdAt: now, expiresAt: now.Add(ttl)}
}

func (c *NumistaCache) DoDetail(
	ctx context.Context,
	id int,
	ttl time.Duration,
	load func(context.Context) (models.NumistaCandidate, error),
) (models.NumistaCandidate, *models.NumistaCacheMetadata, error) {
	if err := ctx.Err(); err != nil {
		return models.NumistaCandidate{}, nil, err
	}
	key := NumistaDetailCacheKey(id)
	c.mu.Lock()
	now := c.clock.Now()
	if cached, metadata, ok := c.getDetailLocked(key, now); ok {
		c.mu.Unlock()
		return cached, metadata, nil
	}
	if call, ok := c.detailInflight[key]; ok {
		call.waiters++
		c.mu.Unlock()
		return c.waitDetail(ctx, key, call)
	}
	loadCtx, cancel := context.WithCancel(context.Background())
	call := &numistaDetailCall{done: make(chan struct{}), cancel: cancel, waiters: 1}
	c.detailInflight[key] = call
	c.mu.Unlock()

	go c.runDetailLoad(key, call, ttl, loadCtx, load)
	return c.waitDetail(ctx, key, call)
}

func (c *NumistaCache) waitDetail(
	ctx context.Context,
	key string,
	call *numistaDetailCall,
) (models.NumistaCandidate, *models.NumistaCacheMetadata, error) {
	select {
	case <-ctx.Done():
		c.mu.Lock()
		if current, ok := c.detailInflight[key]; ok && current == call {
			call.waiters--
			if call.waiters == 0 {
				delete(c.detailInflight, key)
				call.cancel()
			}
		}
		c.mu.Unlock()
		return models.NumistaCandidate{}, nil, ctx.Err()
	case <-call.done:
		if err := ctx.Err(); err != nil {
			return models.NumistaCandidate{}, nil, err
		}
		return call.value, cloneCacheMetadata(call.metadata), call.err
	}
}

func (c *NumistaCache) runDetailLoad(
	key string,
	call *numistaDetailCall,
	ttl time.Duration,
	ctx context.Context,
	load func(context.Context) (models.NumistaCandidate, error),
) {
	defer call.cancel()
	value, err := load(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, ok := c.detailInflight[key]; !ok || current != call {
		call.value, call.err = value, err
		close(call.done)
		return
	}
	var metadata *models.NumistaCacheMetadata
	if err == nil {
		now := c.clock.Now()
		c.setDetailLocked(key, value, ttl, now)
		metadata = cacheMetadata(now, now, now.Add(ttl), false)
	}
	call.value, call.metadata, call.err = value, cloneCacheMetadata(metadata), err
	delete(c.detailInflight, key)
	close(call.done)
}

func (c *NumistaCache) removeExpiredLocked(now time.Time) {
	for key, entry := range c.search {
		if !now.Before(entry.expiresAt) {
			delete(c.search, key)
		}
	}
	for key, entry := range c.detail {
		if !now.Before(entry.expiresAt) {
			delete(c.detail, key)
		}
	}
}

func (c *NumistaCache) evictOldestSearchLocked() {
	var oldestKey string
	var oldest time.Time
	for key, entry := range c.search {
		if oldestKey == "" || entry.createdAt.Before(oldest) {
			oldestKey, oldest = key, entry.createdAt
		}
	}
	delete(c.search, oldestKey)
}

func (c *NumistaCache) evictOldestDetailLocked() {
	var oldestKey string
	var oldest time.Time
	for key, entry := range c.detail {
		if oldestKey == "" || entry.createdAt.Before(oldest) {
			oldestKey, oldest = key, entry.createdAt
		}
	}
	delete(c.detail, oldestKey)
}

func cacheMetadata(now, createdAt, expiresAt time.Time, hit bool) *models.NumistaCacheMetadata {
	age := now.Sub(createdAt).Seconds()
	if age < 0 {
		age = 0
	}
	return &models.NumistaCacheMetadata{
		Hit: hit, CreatedAt: createdAt.UTC(), ExpiresAt: expiresAt.UTC(), AgeSeconds: int64(age),
	}
}

func cloneCandidates(value []models.NumistaCandidate) []models.NumistaCandidate {
	if value == nil {
		return []models.NumistaCandidate{}
	}
	cloned := make([]models.NumistaCandidate, len(value))
	copy(cloned, value)
	for i := range cloned {
		cloned[i].Assessment.Reasons = append([]models.NumistaRelevanceReason(nil), value[i].Assessment.Reasons...)
	}
	return cloned
}

func cloneCacheMetadata(metadata *models.NumistaCacheMetadata) *models.NumistaCacheMetadata {
	if metadata == nil {
		return nil
	}
	cloned := *metadata
	return &cloned
}
