# ADR 0012: Vision-First Deep Identification (Image Hypothesis as Primary Claim Source)

Date: 2026-08-16
Status: Proposed

## Context

Feature 344 shipped the Deep Analysis pipeline and [ADR 0011](0011-deep-agentic-coin-identification.md)
recorded its persistence, eventing, and write-boundary decisions. Feature 345 /
[ADR 0010](0010-ocre-odbl-provider.md) added OCRE as an automated provider.

The pipeline's information architecture is inverted relative to its purpose.
Catalogue providers are treated as the source of truth; the coin images — the
only evidence guaranteed to be present on every run — are treated as decoration.
Three defects, all verified in shipped code:

1. **Providers search a placeholder.**
   `app/teams/deep_identification/providers/numista.py::_build_query` resolves
   `quick_evidence.numista_query` → `quick_evidence.label_text` → `notes[:200]` →
   the constant `_DEFAULT_QUERY = "unidentified ancient coin"`. `nomisma.py` and
   `ocre.py` share the shape. The vision result is never consulted.
2. **The vision result is computed, paid for, and discarded.**
   `graph.py::prepare_evidence_node` runs a full vision LLM call and writes
   `state["image_analysis"]`. A repository-wide search returns three hits: the
   declaration (`state.py:38`) and the two writes (`graph.py:71,86`). Zero reads.
3. **Narrative and draft fields are hard-gated on providers.**
   `synthesis.py::synthesize` replaces the narrative with
   `FALLBACK_NARRATIVE_NO_EVIDENCE` when no provider row has status
   `contributed`, does not accept the image analysis as a parameter, and builds
   `proposed_fields` exclusively from provider claims.

Consequently two landed Feature 344 requirements are unimplemented in substance:
FR-022 ("quick-lookup pass **plus image evidence**" for routing — `router_node`
passes only catalog/override/`max_providers`/`notes`) and FR-027
("contradictions... **including image evidence**" — the evaluator groups only
provider claims). The Feature 344 internal contract §5 already documents
`evidence_refs: [{"provider": "image"}]`, and Go's `buildDeepProposalDocumentJSON`
already tolerates and skips such refs. Nothing has ever emitted one.

The observed failure: a slabbed NGC **Maximinus I AR Denarius**, run with two
clear face photographs and no notes, produced `no_match` from all three
automatable providers, `not_automated` from NGC, a report reading "No provider
evidence could be gathered for this coin", and a draft proposal with **zero**
fields. A single one-shot prompt to the same model with the same two photographs
returned a correct, detailed attribution (Maximinus Thrax, `IMP MAXIMINVS PIVS
AVG`, Rome, AD 235-238, likely `PAX AVGVSTI`, RIC IV 12). The capability was
already being paid for on every run and thrown away.

## Decision

**The image hypothesis is the pipeline's primary identification. Providers are
fact-checkers.**

### 1. The vision node emits a typed hypothesis, not prose

The existing single vision call is bound to a structured schema and returns a
`CoinHypothesis`: per-field `{value, confidence}` entries keyed by the **existing
coin-field vocabulary** (`deepProposalCoinFieldAllowlist`), plus a short
observations string. Unsupported fields are omitted, never guessed. Failure,
timeout, or schema-validation failure degrades to a typed **empty** hypothesis;
the job never fails for this reason. **No second vision call is introduced** —
the call count is unchanged.

`state["image_analysis"]` is deleted. The hypothesis is consumed by all four
downstream nodes, and a test asserts each consumer receives it, so the
write-only-state-field defect class is closed by automation (Principle IX)
rather than by review discipline.

### 2. Provider queries are built from the hypothesis, deterministically

A single shared module builds query terms for every automatable provider with the
precedence `quick_evidence.numista_query` → `quick_evidence.label_text` →
**hypothesis-derived terms** → `notes`. Quick evidence keeps higher precedence
than the hypothesis. Query text remains **application-authored** — no LLM may
choose, rewrite, or extend it. The placeholder constant is deleted; when no tier
yields terms, the node returns a typed `no_match` /
`insufficient_query_evidence` with **zero** upstream calls, so "we had nothing to
search with" is no longer reported as "we searched and found nothing".

**Reverse legend and reverse type are deliberately excluded from query terms**
and instead rank candidates a provider has **already returned**. The same weak
signal has opposite risk profiles depending on where it is applied: in a query, a
misread reverse legend costs the entire result; in ranking, it costs only
ordering. This adds **zero** upstream calls and **zero** call budget — no second,
narrower probe is issued.

For **Numista and Nomisma** this is new: `providers/numista.py` requests
`limit=5` and then takes `candidates[0]` unconditionally, discarding four
candidates unranked. A deterministic, application-owned ranker replaces that
blind pick; ranking is never delegated to an LLM.

For **OCRE this is not new, and ADR 0010 is not amended.** The mechanism already
exists and is ADR 0010 governed: `providers/ocre.py::_legend_tokens` already
passes tokens documented as *"scoring-only signals (never SPARQL)"*, and
`src/api/services/ocre_scoring.go` already applies a per-match legend bonus,
capped, added to a weighted base score, clamped to `[0,1]`, over a stable sort.
The only change here is widening the **source** of those tokens — today
`_legend_tokens` reads only `quick_evidence.label_text`, so it contributes
nothing in exactly the evidence-free scenario this feature exists to fix. The
scoring weights, bonus-per-match, bonus cap, clamping, and stable sort remain
untouched: that math is ADR 0010's deterministic contract, and this ADR widens an
input to it rather than amending it.

### 3. The router becomes deterministic

`route()` becomes a pure function of `(catalog, provider_override, bounds,
quick_evidence, hypothesis)`. `ROUTER_PROMPT` and its LLM call are deleted; the
`RouterDecision` shape and the `router_selected` frame are unchanged, as is
`provider_override` winning outright over the closed Go-supplied catalog.

Rationale: only three providers are automatable and `bounds.max_providers` is
≥ 3 in practice, so the decision space is nearly always "all three". The shipped
prompt already biases toward inclusion and **every** failure path already falls
back to selecting all automatable providers — the model pays latency and tokens
to usually reproduce a constant. It was also the only nondeterministic step in an
otherwise deterministic selection/merge/ranking design (`merge.sort_claims`,
`evaluator.detect_disagreements`, and ADR 0010's deterministic OCRE scoring).
Making it deterministic is what makes "routing uses image evidence" testable
rather than hopeful, and removing the call offsets the cost of structured vision
output.

### 4. Image evidence is a first-class claim source — but not a provider

The evaluator flattens hypothesis fields into claims alongside provider claims
and applies the existing deterministic normalization, so a provider that
contradicts what is legible on the coin produces a `DisagreementEntry` with
`resolution: "unresolved"` referencing both sources. Detection stays LLM-free.

`image` is a **claim source**, not a provider: it never appears in
`provider_catalog`, `provider_override`, `DeepSynthesis.coverage`,
`DeepSynthesis.attributions`, or the `ProviderName` union. `EvidenceRef.provider`
is already a bounded free string, so this needs no breaking model change.

### 5. Synthesis narrates from vision AND providers

The narrative describes what the images support, what each provider confirmed,
refined, or contradicted, and what remains open. `proposed_fields` are built from
the hypothesis **and** provider claims: an image-only field is proposed at its
hypothesis confidence with `evidence_refs: [{"provider": "image"}]`; a
corroborating provider claim raises confidence by a deterministic, bounded,
documented rule and attaches its citation; a contradicted field is withheld from
the proposal and surfaced as a disagreement.
`FALLBACK_NARRATIVE_NO_EVIDENCE` becomes reachable **only** when both the
hypothesis and provider evidence are empty.

### 6. Three narrow Go changes; the rest of the Go layer is untouched

- The quick-lookup pass inside Deep Analysis gets a budget proportionate to the
  work it does. `deep_identification_pipeline_runner.go:112` currently allows
  `15*time.Second` for a full vision LLM round trip that is allowed **five
  minutes** on the standalone path (`agent_proxy.go:36`). The literal becomes a
  named, admin-tunable setting bounded by the proxy ceiling, and the interaction
  with the pipeline's remaining-budget/safety-margin computation
  (`runner.go:116-123`) is **verified empirically rather than assumed**. This
  defect is independently sufficient to reproduce the observed failure and is
  shippable ahead of the redesign.
- A Deep Analysis result may be saved, on explicit owner confirmation, as a
  **wishlist item** as well as a collection item. The mechanism is a
  `models.Coin` with `IsWishlist = true`, created through the existing
  `CoinService` create path, populated from the **unwidened**
  `deepProposalCoinFieldAllowlist`. It is a third destination on the existing
  closed `switch target` in `DeepIdentificationProposalService.Apply`, gated on
  the intake job source exactly as the `draft` destination is. **No new write
  surface, no new field allowlist, no database migration, no change to
  `QuickCaptureDraft`.**
  `isWishlist` is a **destination intent carried on the apply request, never a
  proposed field**: it is not in the allowlist, never appears in
  `proposed_fields`, and is not proposable or influenceable by model output. It
  is derived only from a normalized target supplied by the owner at confirm
  time, mirroring the existing `QuickCapturePromotionTarget` precedent
  (`services/quick_capture_service.go:530`).
- Quick-lookup **failure** becomes a typed, observable job outcome instead of
  `deep_identification_pipeline_runner.go::extractQuickEvidence` logging a `Warn`
  and returning `nil`, which is indistinguishable downstream from "no quick
  evidence existed".

Job lifecycle, persistence, event sequencing/replay, cancellation, retry,
retention, and authorization are audited as production-quality and are
explicitly **out of scope**.

### 7. Additive only; the Python agent stays stateless

`DeepSynthesis` gains an optional `image_hypothesis` key; `ProviderErrorKind`
gains one member. No database migration, no public SSE vocabulary change, no new
credential, no new egress, no new dependency. The Python agent continues to hold
no database handle and no cross-request state (Constitution Principle II).

## Relationship to ADR 0011

ADR 0011 remains **in force**. It is **amended, not superseded**, in exactly two
respects:

1. Its Decision sentence "The router records its selected provider set and
   rationale" is retained as an *outcome*, but the router ceases to be an LLM
   step (§3 above).
2. Its evidence model — implicitly provider-only — is extended: the image
   hypothesis is a first-class claim source for evaluation and proposal evidence,
   while remaining excluded from provider coverage and attribution (§4 above).

Everything else in ADR 0011 (the four tables, Go-owned persistence and events,
sequence-numbered replay, hint-image ephemerality, confirm-gated writes,
stateless Python, RPC unavailable, default-off) is unchanged and binding.

ADR 0010 (OCRE / ODbL) is unaffected: OCRE's transport, licensing, attribution,
deterministic scoring, and default-off posture are untouched; only the *query
terms* it receives change.

## Alternatives Considered

- **Patch the fallback narrative to mention the image analysis.** Rejected — it
  would surface a paragraph that is still never used for routing, fact-checking,
  or draft fields, and would leave the empty-draft failure intact. A cosmetic fix
  to a structural defect (Principle IV forbids the hopeful patch).
- **Keep free-prose vision output and have the LLM extract fields later.**
  Rejected — adds a second LLM call, reintroduces nondeterminism, and lets a
  model author provider query text, violating the "query text is never freely
  chosen by an LLM" property.
- **Keep the LLM router and feed it the hypothesis.** Rejected — see §3. Three
  providers, an inclusion-biased prompt, and an all-providers failure fallback
  make the call nearly always a constant, at the cost of the only remaining
  nondeterminism in the selection path.
- **Make `image` a full provider** (catalog entry, coverage row, attribution).
  Rejected — it has no citation, no license, no upstream, and no call budget;
  widening `ProviderName` would break the Go mirror, the coverage UI, and the
  citation-allowlist invariant for no benefit.
- **Let the image hypothesis override contradicting providers (or vice versa).**
  Rejected — 344 FR-024/FR-027 forbid silent precedence resolution in either
  direction; the honest outcome is a visible unresolved disagreement.
- **A second, dedicated "deep vision" call in addition to the existing one.**
  Rejected — doubles the most expensive step to obtain something the existing
  call can return with a schema binding.
- **Carry the wishlist destination as a flag on `QuickCaptureDraft`** (rejected
  in favour of §6's direct wishlist `Coin`). Rejected on two independent grounds.
  First, it requires a **database migration on a shipped table**, breaking this
  feature's no-migration guarantee and widening blast radius into Quick Capture,
  a workflow this change otherwise does not touch. Second,
  `deepProposalDraftFieldAllowlist` is four fields (`workingTitle`, `era`,
  `dateRange`, `notes`) against the coin allowlist's fourteen, so routing a
  wishlist result through a draft would discard precisely the ruler,
  denomination, mint, and legend data this feature exists to produce — the
  wishlist item would arrive stripped of its identification. The counter-argument
  (the direct-coin path skips intake's draft review step) is answered by the fact
  that the Deep Analysis proposal editor **is** that review step, and the confirm
  gate is unchanged.
- **Feed reverse legend/type into provider query terms**, or issue a second
  narrower probe when the first returns `no_match`. Rejected — reverse legends
  are the least reliably legible text on a worn ancient coin, so a misread one
  poisons the query and converts a good match into `no_match`; and a second probe
  spends call budget (344 FR-013) for a signal that is more useful as a
  tie-breaker. The same signal is applied to already-returned candidates instead,
  at zero additional call cost.
- **Change ADR 0010's OCRE scoring weights to account for hypothesis-derived
  legend tokens.** Rejected — the deterministic scoring math is ADR 0010's
  contract. Widening the token *source* achieves the goal without touching
  weights, bonus caps, clamping, or sort stability, and therefore requires no
  amendment to ADR 0010.
- **Default image-only proposed fields to unaccepted (source-driven opt-in).**
  Rejected — on the very case this feature exists to serve, *every* field is
  image-only, so the owner would have to hand-tick a dozen checkboxes to accept a
  result the system was already confident about, and the feature would appear to
  have done nothing. Acceptance is driven by confidence, with provenance
  communicated separately and visibly.
- **Rewrite the Go job/event/SSE layer while we are here.** Rejected — it was
  audited as production-quality; the defect is entirely in the Python pipeline
  plus three narrow Go seams (Principle IV: proportional).

## Consequences

### Positive

- A coin that no catalogue can match still yields a real identification, a real
  narrative, and a savable draft — the Maximinus run stops producing an empty
  result.
- Providers finally receive meaningful queries, so their hit rate should rise for
  reasons unrelated to the providers themselves.
- Provider-vs-image contradictions become visible, closing 344 FR-027 in
  substance.
- Routing becomes reproducible and explainable in the rationale the owner already
  sees, closing 344 FR-022 in substance.
- **Fewer LLM calls per job** (router removed) with no additional vision call.
- Additive, migration-free, and reversible.

### Negative and trade-offs

- **Structured-output reliability varies by model.** A self-hosted Ollama model
  may fail schema validation more often than Claude; mitigated by the typed empty
  degrade (behavior equal to today) but it is the plan's highest-uncertainty item.
- **A confidently wrong hypothesis can misdirect all three providers.** Mitigated
  by quick evidence retaining higher precedence and by contradictions surfacing
  rather than resolving — but the failure mode is real and new.
- **Provider-selection nuance moves from a prompt into application code.** With
  three providers this is cheaper; past roughly six genuinely
  context-dependent providers an LLM router could be reintroduced behind the same
  `RouterDecision` interface.
- **Confidence semantics become richer** (image-only vs. corroborated), which the
  UI must communicate clearly or owners may over-trust image-only fields.
- Image-only proposed fields carry no citation, so the report contains asserted
  facts with no external source. This is honest and marked, but it is a change in
  the character of the output.

## Security and Privacy

- No new upstream integration, credential, egress, or dependency.
- The hypothesis, legend text, and owner notes are excluded from application logs
  and `progress` event payloads; existing Python and Go sanitizers are unchanged.
- Hint images remain ephemeral and never enter the vision-prompt face slots
  (344 FR-004/FR-030).
- No automatic writes: the confirm gate, owner-edit distinction, and coin-field
  allowlists are unchanged, and the added wishlist destination reuses existing
  Go-owned write services.
- The Python agent remains stateless with no database handle.

## Rollout

**Straight cutover — no transitional A/B flag.** The existing
`SettingDeepIdentificationEnabled` remains the kill switch. Running the old and
new synthesis paths side by side would mean *maintaining the provider-gated path*
and doubling the synthesis test matrix in order to preserve a fallback to output
the owner has already rejected as useless. There is nothing to A/B against.

Staged validation comes from the phased merge plan instead: work reaches a
`beta` branch in independently shippable groups, with the vision rewrite
(structured output → hypothesis → query terms → router → evaluator → synthesis →
regression gate) merged as **one indivisible unit**, because any partial merge
leaves the pipeline provably half-rewritten. See `tasks.md` §Beta merge plan.

## Rollback

Revert the code. No down-migration exists because no schema changed. Reports
persisted with `image_hypothesis` remain readable (the older Go reader unmarshals
only `narrative` and `proposed_fields`), and proposals containing
`{"provider": "image"}` evidence refs remain applicable (the older Go builder
already skips them). Terminal jobs are not re-run or backfilled. The existing
`DeepIdentificationEnabled` setting remains the kill switch for the feature as a
whole.

## Related

- [Feature 351 specification](../../specs/351-vision-first-deep-identification/spec.md)
- [Feature 351 plan](../../specs/351-vision-first-deep-identification/plan.md)
- [Feature 351 contract: vision hypothesis](../../specs/351-vision-first-deep-identification/contracts/vision-hypothesis.md)
- [Feature 351 data model](../../specs/351-vision-first-deep-identification/data-model.md)
- [ADR 0011: Persisted Deep Agentic Coin Identification](0011-deep-agentic-coin-identification.md) — amended by this ADR
- [ADR 0010: OCRE ODbL Provider](0010-ocre-odbl-provider.md) — unaffected
- [Feature 344 specification](../../specs/344-deep-agentic-coin-identification/spec.md) — FR-022, FR-024, FR-025, FR-027, FR-028, FR-029 amended
- [Feature 344 internal contract](../../specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md)
