package services

import (
	"testing"
	"time"
)

// Feature 345 T009 — cache identity + TTL + negative-caching behavior.

func ocreCandidateFixture() []OCRECandidate {
	return []OCRECandidate{{
		TypeURI:       "https://numismatics.org/ocre/id/ric.2.hdn.39b",
		Label:         "RIC II Hadrian 39b",
		MatchedFields: []string{"ruler:hadrian"},
		Confidence:    0.86,
		Explanation:   "Matched ruler hadrian.",
	}}
}

func TestOCRECache_HitWithinTTL(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := NewOCRECacheForTest(func() time.Time { return now })
	params := NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 5)

	cache.Set(params, "gen1", OCRESearchOK, ocreCandidateFixture())
	status, candidates, ok := cache.Get(params, "gen1")
	if !ok || status != OCRESearchOK || len(candidates) != 1 {
		t.Fatalf("expected a cache hit, got ok=%v status=%q candidates=%+v", ok, status, candidates)
	}
}

func TestOCRECache_DistinctKeyPerBoundParameter(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := NewOCRECacheForTest(func() time.Time { return now })
	base := NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 5)
	cache.Set(base, "gen1", OCRESearchOK, ocreCandidateFixture())

	variants := []OCREQueryParams{
		NewOCREQueryParams("trajan", "denarius", "rome", "", nil, "", 5),
		NewOCREQueryParams("hadrian", "sestertius", "rome", "", nil, "", 5),
		NewOCREQueryParams("hadrian", "denarius", "alexandria", "", nil, "", 5),
		NewOCREQueryParams("hadrian", "denarius", "rome", "silver", nil, "", 5),
		NewOCREQueryParams("hadrian", "denarius", "rome", "", []string{"cos"}, "", 5),
		NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 10),
	}
	for i, v := range variants {
		if _, _, ok := cache.Get(v, "gen1"); ok {
			t.Fatalf("variant %d must not reuse the base cache entry", i)
		}
	}
}

func TestOCRECache_FlagGenerationInvalidatesReuse(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := NewOCRECacheForTest(func() time.Time { return now })
	params := NewOCREQueryParams("hadrian", "denarius", "rome", "", nil, "", 5)
	cache.Set(params, "gen1", OCRESearchOK, ocreCandidateFixture())

	if _, _, ok := cache.Get(params, "gen2"); ok {
		t.Fatal("a flagGeneration change (toggle) must never reuse a stale entry")
	}
	if _, _, ok := cache.Get(params, "gen1"); !ok {
		t.Fatal("the original generation should still hit")
	}
}

func TestOCRECache_NegativeResultCached(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := NewOCRECacheForTest(func() time.Time { return now })
	params := NewOCREQueryParams("obscurus", "", "", "", nil, "", 5)
	cache.Set(params, "gen1", OCRESearchNoMatch, nil)

	status, _, ok := cache.Get(params, "gen1")
	if !ok || status != OCRESearchNoMatch {
		t.Fatalf("expected a cached no_match, got ok=%v status=%q", ok, status)
	}
}

func TestOCRECache_TransientFailuresNeverCached(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := NewOCRECacheForTest(func() time.Time { return now })
	params := NewOCREQueryParams("hadrian", "", "", "", nil, "", 5)
	cache.Set(params, "gen1", OCRESearchUnavailable, nil)

	if _, _, ok := cache.Get(params, "gen1"); ok {
		t.Fatal("an unavailable/transient outcome must never be cached")
	}
}

func TestOCRECache_ExpiryEvictsEntry(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := NewOCRECacheForTest(func() time.Time { return now })
	params := NewOCREQueryParams("hadrian", "", "", "", nil, "", 5)
	cache.Set(params, "gen1", OCRESearchOK, ocreCandidateFixture())

	now = now.Add(ocreCacheTTL + time.Second)
	if _, _, ok := cache.Get(params, "gen1"); ok {
		t.Fatal("expected the entry to have expired past its TTL")
	}
}

func TestOCRECache_LegendTokenOrderIrrelevant(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := NewOCRECacheForTest(func() time.Time { return now })
	a := NewOCREQueryParams("hadrian", "", "", "", []string{"cos", "iii"}, "", 5)
	b := NewOCREQueryParams("hadrian", "", "", "", []string{"iii", "cos"}, "", 5)
	cache.Set(a, "gen1", OCRESearchOK, ocreCandidateFixture())
	if _, _, ok := cache.Get(b, "gen1"); !ok {
		t.Fatal("legend token ordering must not affect the cache key")
	}
}
