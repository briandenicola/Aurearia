package services

import (
	"strconv"
	"testing"
	"time"
)

func TestNomismaCache_MissThenHit(t *testing.T) {
	now := time.Now()
	cache := NewNomismaCacheForTest(func() time.Time { return now })

	if _, _, ok := cache.Get("Roma"); ok {
		t.Fatal("expected a miss before Set")
	}

	candidates := []NomismaCandidate{{URI: "http://nomisma.org/id/roma", Label: "Roma", Score: 100, Match: true}}
	cache.Set("Roma", NomismaSearchOK, candidates)

	status, got, ok := cache.Get("Roma")
	if !ok {
		t.Fatal("expected a hit after Set")
	}
	if status != NomismaSearchOK || len(got) != 1 || got[0].URI != candidates[0].URI {
		t.Fatalf("unexpected cached value: %v %+v", status, got)
	}
}

func TestNomismaCache_KeyIsNormalized(t *testing.T) {
	now := time.Now()
	cache := NewNomismaCacheForTest(func() time.Time { return now })
	cache.Set("  Roma   Mint  ", NomismaSearchOK, []NomismaCandidate{{URI: "u", Label: "l"}})

	if _, _, ok := cache.Get("roma mint"); !ok {
		t.Fatal("expected a hit for a differently-cased/whitespace-collapsed query")
	}
}

func TestNomismaCache_Expiry(t *testing.T) {
	now := time.Now()
	cache := NewNomismaCacheForTest(func() time.Time { return now })
	cache.Set("Roma", NomismaSearchOK, []NomismaCandidate{{URI: "u", Label: "l"}})

	now = now.Add(nomismaCacheTTL + time.Second)
	if _, _, ok := cache.Get("Roma"); ok {
		t.Fatal("expected the entry to have expired")
	}
}

func TestNomismaCache_NegativeEntryIsCached(t *testing.T) {
	now := time.Now()
	cache := NewNomismaCacheForTest(func() time.Time { return now })
	cache.Set("zzzzgibberish", NomismaSearchNoMatch, []NomismaCandidate{})

	status, got, ok := cache.Get("zzzzgibberish")
	if !ok {
		t.Fatal("expected a zero-result search to be cached as a negative entry")
	}
	if status != NomismaSearchNoMatch || len(got) != 0 {
		t.Fatalf("unexpected cached negative entry: %v %+v", status, got)
	}
}

func TestNomismaCache_NeverCachesUnavailable(t *testing.T) {
	now := time.Now()
	cache := NewNomismaCacheForTest(func() time.Time { return now })
	cache.Set("Roma", NomismaSearchUnavailable, nil)

	if _, _, ok := cache.Get("Roma"); ok {
		t.Fatal("expected an unavailable outcome to never be cached")
	}
}

func TestNomismaCache_EvictsOldestWhenBoundReached(t *testing.T) {
	now := time.Now()
	cache := NewNomismaCacheForTest(func() time.Time { return now })

	for i := 0; i < nomismaCacheMaxEntry; i++ {
		now = now.Add(time.Millisecond)
		cache.Set("query-"+strconv.Itoa(i), NomismaSearchOK, []NomismaCandidate{{URI: "u", Label: "l"}})
	}
	if len(cache.entries) != nomismaCacheMaxEntry {
		t.Fatalf("expected cache size bounded at %d, got %d", nomismaCacheMaxEntry, len(cache.entries))
	}

	// One more insert should evict the single oldest entry, keeping the
	// total at the bound rather than growing unbounded.
	now = now.Add(time.Millisecond)
	cache.Set("one-more-query", NomismaSearchOK, []NomismaCandidate{{URI: "u", Label: "l"}})
	if len(cache.entries) != nomismaCacheMaxEntry {
		t.Fatalf("expected cache size to remain bounded at %d after eviction, got %d", nomismaCacheMaxEntry, len(cache.entries))
	}
}
