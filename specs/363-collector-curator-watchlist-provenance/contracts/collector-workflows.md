# Contract: Collector Profile and Read-Only Workflows

## 1. Boundary and compatibility

Vue calls authenticated Go `/api/*` routes only. Go derives owner identity and
resolves every referenced profile, goal, coin, run, result and evidence record.
Python receives only an internal, bounded execution request from Go and returns
strict Pydantic values; it has no public route, database access, durable memory,
write tool, approval field or generic credential.

All request decoders reject unknown fields. Go uses explicit DTOs and
`DisallowUnknownFields`; Python models use `extra="forbid"`; TypeScript uses
discriminated unions. Strings are UTF-8, trimmed and bounded. JSON request
bodies are capped at 1 MiB; workflow-specific payload limits below are lower.
Foreign and unknown identifiers return indistinguishable `404` responses.

## 2. Closed vocabularies

```text
confidence: low | medium | high
goalPriority: low | medium | high
goalState: active | inactive
recommendationKind: strength | theme | gap | acquisition_idea
profileEffect: included | excluded | deprioritized | neutral
inputKind: wishlist_coin | collecting_goal | market_result
marketKind: dealer_listing | auction_lot
evaluationStatus: match | conflict | unknown | not_applicable
riskKind: missing_provenance | unverified_claim | conflicting_claim |
          duplicate_image | broken_source_link | documentation_gap
riskTier: review_low | review_medium | review_high
dependencyState: baseline | feature_362 | attribution_evidence_unavailable
```

Unknown enum values fail closed. No “other” escape value is accepted at a
service boundary.

## 3. Collector profile public contract

Base path: `/api/collector/profile`.

### `GET /api/collector/profile`

`200`:

```json
{
  "budgetMin": null,
  "budgetMax": null,
  "currency": "USD",
  "favoritePeriods": [],
  "dislikedCategories": [],
  "preferredDealers": [],
  "goals": [],
  "version": 0,
  "updatedAt": null,
  "isDefault": true
}
```

Missing storage returns this neutral projection. Currency is the existing
owner display currency when supported, otherwise `USD`; it does not imply a
preference.

### `PUT /api/collector/profile`

```json
{
  "expectedVersion": 2,
  "budgetMin": 100,
  "budgetMax": 500,
  "currency": "USD",
  "favoritePeriods": ["Flavian"],
  "dislikedCategories": ["Modern replicas"],
  "preferredDealers": ["Example Dealer"],
  "goals": [
    {
      "id": "b8488f81-a94b-4cc6-93d1-8caebac57e25",
      "title": "Add a Flavian denarius",
      "description": "Prioritize documented examples.",
      "priority": "high",
      "state": "active",
      "version": 1
    }
  ]
}
```

Bounds are normative from `data-model.md`: amounts `0..100000000`; uppercase
supported ISO-style three-letter currency; list counts `20/20/50`; goal count
50; title 1..200; description ≤1000. Duplicate normalized entries, non-finite
numbers, reversed budgets, unknown goal/version/state or stale
`expectedVersion` reject the entire transaction. `409 profile_version_conflict`
returns only current version and a refresh instruction, not profile content.

`200` returns the complete saved profile. Clearing nullable/scalar/list fields
is explicit; omission is rejected because this is a full atomic replacement.

## 4. Public read-only workflow requests

The existing Coin Copilot durable start/stream/replay/cancel routes remain
canonical. A new run request selects one closed workflow in the existing typed
intent/capability contract:

```json
{
  "workflow": "collector_curator",
  "profileVersion": 2,
  "inputs": []
}
```

or:

```json
{
  "workflow": "watchlist_evaluation",
  "profileVersion": 2,
  "inputs": [
    {"kind": "wishlist_coin", "coinId": 42},
    {"kind": "collecting_goal", "goalId": "b8488f81-a94b-4cc6-93d1-8caebac57e25"},
    {
      "kind": "market_result",
      "marketKind": "auction_lot",
      "runId": "ccr_123",
      "executionId": "cce_123",
      "toolCallId": "call_2",
      "canonicalSourceId": "https://approved.example/auction/1",
      "resultDigest": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    }
  ]
}
```

Bounds: at most 20 inputs; positive numeric coin ids; bounded existing run ids;
64-character lowercase SHA-256 digest; URL ≤2048. A posted URL is identity
only—Go must resolve it inside retained owner-bound evidence and never fetch it
because the client supplied it. Direct URL/saved-search inputs reject with
`400 deferred_input_surface`.

## 5. Internal Go → Python request

```json
{
  "schema_version": 1,
  "workflow": "watchlist_evaluation",
  "profile_snapshot": {
    "profile_version": 2,
    "profile_digest": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    "captured_at": "2026-09-18T18:00:00Z",
    "budget_min": 100,
    "budget_max": 500,
    "currency": "USD",
    "favorite_periods": ["Flavian"],
    "disliked_categories": [],
    "preferred_dealers": [],
    "goals": []
  },
  "collection_facts": [],
  "inputs": [],
  "feature_362": {"state": "unavailable", "projection": null},
  "limits": {
    "max_items": 20,
    "max_evidence_per_item": 10,
    "max_limitations": 10,
    "max_text_chars": 1000
  }
}
```

Owner/user ids, credentials, raw provider payloads, model messages, hidden
prompts, image bytes/paths and public write URLs are excluded. Goal/provider
text is marked untrusted data and cannot alter tools, bounds or safety rules.

## 6. Recommendation result

```json
{
  "schema_version": 1,
  "workflow": "collector_curator",
  "profile_version": 2,
  "profile_digest": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "outcome": "complete",
  "recommendations": [
    {
      "id": "rec_1",
      "kind": "gap",
      "observed_facts": [
        {"fact_id": "collection_gap:flavian_denarius", "summary": "No matching owned coin was found."}
      ],
      "recommendation": "Recommendation: review a documented Flavian denarius.",
      "profile_effects": [
        {"effect": "included", "field": "favorite_periods", "goal_id": null}
      ],
      "confidence": "medium",
      "why_this_matters": "It addresses a supported collection gap.",
      "citations": [{"kind": "internal_fact", "id": "collection_gap:flavian_denarius", "url": null}],
      "limitations": ["Collection metadata may be incomplete."]
    }
  ],
  "warnings": [],
  "truncation": {"truncated": false, "omitted_items": 0}
}
```

`outcome` is `complete|partial|no_match|unavailable`. At most 20 items, 10
facts/citations/limitations each; summary/recommendation text ≤1000 characters.
An empty profile has version 0 and produces no invented preference effects.

## 7. Watchlist evaluation result

```json
{
  "schema_version": 1,
  "workflow": "watchlist_evaluation",
  "profile_version": 2,
  "outcome": "partial",
  "evaluations": [
    {
      "id": "eval_1",
      "input": {"kind": "market_result", "market_kind": "auction_lot", "source_identity_digest": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
      "checks": {
        "goal_fit": "match",
        "budget_fit": "unknown",
        "preference_fit": "match",
        "dealer_fit": "unknown",
        "owned_duplicate": "conflict",
        "wishlist_duplicate": "not_applicable",
        "collection_coverage": "conflict",
        "source_quality": "match",
        "listing_state": "unknown",
        "price_comparability": "unknown"
      },
      "evidence": [],
      "confidence": "low",
      "why_this_matters": "A possible owned duplicate requires review.",
      "limitations": ["The listing currency cannot be compared.", "Availability evidence is stale."]
    }
  ],
  "warnings": [],
  "truncation": {"truncated": false, "omitted_items": 0}
}
```

Unknown/missing/stale/contradictory values remain `unknown` and are never
coerced. Existing availability and auction references include their observed
timestamps and authoritative resource links.

## 8. Risk result

```json
{
  "schema_version": 1,
  "workflow": "provenance_risk_review",
  "outcome": "partial",
  "findings": [
    {
      "id": "risk_1",
      "kind": "documentation_gap",
      "tier": "review_low",
      "needs_review": true,
      "display_label": "Needs review: documentation gap",
      "observed_facts": [],
      "source_claims": [],
      "conflicts": [],
      "recommendation": "Request or record the missing ownership documentation.",
      "evidence": [],
      "confidence": "low",
      "why_this_matters": "Documentation can help future review.",
      "limitations": ["Absence in the current record does not prove the documentation does not exist."],
      "dependency_state": "baseline"
    }
  ],
  "gates": ["attribution_evidence_unavailable"],
  "warnings": []
}
```

All findings require evidence or an explicit missing-field observation,
`needs_review=true`, a safe display label, why-it-matters and at least one
limitation. Low confidence remains visible in `review_low`. Output is rejected
if it contains a person/dealer accusation, fraud conclusion, authenticity
determination, unsupported duplicate-image claim, unsafe citation, or
attribution inference while Feature 362 is gated.

## 9. URL/citation rules

- HTTPS only; hostname required; no user-info.
- Reject localhost, loopback, private, link-local and metadata destinations.
- Source host must match the registered provider boundary.
- Revalidate every redirect at the existing outbound helper.
- URLs ≤2048; citation URL must be present in the validated evidence set.
- Preserve accepted display URL; canonicalize only for equality/duplicates.
- No raw HTML, response body, provider JSON, base64 image or query credentials.

## 10. Cancellation, replay and mutation absence

Go cancellation is checked before each capability, after every awaited
operation, before persistence of result frames and before any stage request.
Late frames are discarded after cancellation wins. Existing completed
checkpoint values replay without provider/model reruns.

The internal tool allowlist contains no create/save/stage/confirm/apply/bid/buy
operation. Recommendation/evaluation/risk schemas contain no confirmation
boolean, write token or public mutation URL. Viewing, dismissing, replaying or
cancelling these values cannot mutate profile, coin, wishlist, auction,
availability, Deep Analysis proposal or settings data.

## 11. Cross-language fixtures

Go, Python and Vue must consume canonical valid/adversarial fixtures covering:

- every closed enum and unknown enum/field;
- min/max counts, lengths, amounts and non-finite JSON;
- profile v0 and stale profile/goal versions;
- all three input variants plus foreign/cross-user ids;
- complete/partial/no-match/unavailable outcomes;
- missing/stale/conflicting/incomparable evidence;
- low-confidence visible risk and banned accusatory wording;
- duplicate-image finding with/without defensible comparison evidence;
- safe, unsafe, redirecting and unregistered URLs;
- instruction/token-shaped provider and goal text;
- cancellation, truncated result and replay digest mismatch;
- Feature 362 available/unavailable projections.
