# Data Model: Vision-First Deep Identification

**Feature**: 351 | **Spec**: [spec.md](./spec.md) | **Contract**: [contracts/vision-hypothesis.md](./contracts/vision-hypothesis.md)

**No database migration.** Every change is additive JSON inside columns that
already exist, or is confined to in-memory Python pipeline state. The four
Feature 344 tables (`deep_identification_jobs`, `_events`, `_provider_runs`,
`_artifacts`) are untouched in shape.

## 1. New logical entity — Coin Hypothesis (image-derived)

The typed output of the single vision call. Lives in Python graph state for the
duration of one request and is persisted **only** as an additive key inside the
existing report JSON.

| Attribute | Type | Notes |
|---|---|---|
| `<coin field>` | `{ value: string, confidence: float 0..1 }` | keys restricted to the coin-field vocabulary (§3); unsupported fields omitted entirely |
| `observations` | bounded string | short prose for the narrative writer only; never a proposed value |
| `legible` | bool | `false` for the typed-empty degrade case |

Invariants:

- No citation. Never a `ProviderClaim`. Never in the provider catalog, coverage,
  or attribution lists.
- Absent/failed/unparseable ⇒ typed empty hypothesis, never a job failure.
- Never written to logs or `progress` payloads.

## 2. New logical entity — Claim Source

Generalizes "provider" for evaluation and proposal-evidence purposes.

| Source kind | Identity | Citation | Appears in coverage/attribution |
|---|---|---|---|
| Provider claim | `ProviderName` (`numista`/`nomisma`/`ngc`/`ocre`/`rpc`) | required, host-allowlist validated | yes |
| Image claim | the literal `image` | none | **no** |

`EvidenceRef.provider` is already a bounded free string, so an image ref needs no
model widening. `ProviderCoverageEntry.provider` and
`ProviderAttribution.provider` remain the `ProviderName` literal union.

## 3. Coin-field vocabulary (unchanged, reused)

Hypothesis keys are normalized into the existing
`deepProposalCoinFieldAllowlist` vocabulary
(`src/api/services/deep_identification_proposal.go`) so image-derived fields
reach the draft through the **existing** write allowlist:

`denomination`, `ruler`, `era`, `dateRange`, `mint`, `material`, `weightGrams`,
`diameterMm`, `obverseInscription`, `reverseInscription`, `obverseDescription`,
`reverseDescription`, `notes`, `coin_type`.

Keys outside this list are dropped during normalization. **No new writable field
is introduced by this feature.**

The intake/draft path continues to use the narrower
`deepProposalDraftFieldAllowlist` (`workingTitle`, `era`, `dateRange`, `notes`)
and continues to synthesize `workingTitle` from `ruler` + `denomination` — which
is precisely why an empty `proposed_fields` map produced an empty draft on the
Maximinus run, and why populating it from the hypothesis fixes the draft too.

## 4. Persistence reuse (no new columns)

| Data | Where it lives today | Change |
|---|---|---|
| Synthesized report | `DeepIdentificationJob` report JSON (written verbatim from the `synthesis` frame) | gains optional `image_hypothesis` key |
| Proposal document | job proposal JSON (`deepProposalDocument`) | may contain fields whose `evidence` array is empty (image-only support) |
| Provider run rows | `DeepIdentificationProviderRun` | unchanged; gains the `insufficient_query_evidence` error kind value in the existing bounded string column |
| Events | `DeepIdentificationEvent` | unchanged vocabulary and payload shapes |
| Artifacts | `DeepIdentificationArtifact` | unchanged; hint images remain ephemeral |

## 5. Status / error-kind vocabulary delta

| Vocabulary | Change |
|---|---|
| `ProviderStatus` | **unchanged** (`contributed`, `no_match`, `failed`, `timed_out`, `not_automated`, `unavailable`, `skipped`) |
| `ProviderErrorKind` | **+ `insufficient_query_evidence`** — paired with `status: "no_match"` and `call_count: 0` |
| Quick-lookup outcome | **new, typed**: at minimum `ok`, `no_data`, `unavailable` — replacing the current "log a Warn and return nil", which conflates the last two |
| Job status | **unchanged** |

## 6. Proposal document semantics

`deepProposalFieldEntry` shape is unchanged. Two clarifications:

1. A field supported only by the image hypothesis has `Proposed`, `Confidence`,
   and an **empty** `Evidence` array. The Go builder MUST emit it, not drop it.
2. `OwnerEdited` / `OwnerValue` / `Accepted` semantics are unchanged; whether
   image-only fields default to unaccepted is spec OQ-2.

## 7. Backward compatibility matrix

| Artifact written | Read by old code | Read by new code |
|---|---|---|
| Report **without** `image_hypothesis` (pre-351) | ✅ unchanged | ✅ hypothesis surface renders empty |
| Report **with** `image_hypothesis` (post-351) | ✅ key ignored — Go unmarshals only `narrative` + `proposed_fields` | ✅ |
| Proposal with provider-cited fields only | ✅ | ✅ |
| Proposal with image-only fields (empty evidence) | ✅ applies normally — evidence is display metadata, not a write input | ✅ marked as image-derived |
| Provider run with `insufficient_query_evidence` | ✅ bounded string column | ✅ |

Rollback: reverting the code leaves every persisted artifact readable. No
backfill, no down-migration, no re-run of terminal jobs.

## 8. Targeted regression coverage (Definition of Done §21.6)

| Path | Test |
|---|---|
| The Maximinus run (the exact failing path) | end-to-end fixture asserting the spec US2 before/after table |
| Write-only state field defect class | assertion that router, query building, evaluator, and synthesis each receive the hypothesis |
| Placeholder queries | assertion that no provider is ever called with a placeholder string, across all three automatable nodes |
| Fallback narrative boundary | provider-empty + hypothesis-present run must NOT emit the fallback |
| Image-vs-provider contradiction | unresolved disagreement with both refs; field withheld from the proposal |
| Corroboration | confidence upgrade bounded, both refs present, citation attached |
| Router determinism | two identical runs produce byte-identical selection output |
| Wishlist apply | confirm-gated write lands as a wishlist entry; no write before confirmation |
| Quick-lookup failure | typed `unavailable` outcome distinct from `no_data`; no user content in payload/logs |
| Backward compatibility | pre-351 report + post-351 image-only proposal both load, render, and apply |
