# Research: Quick Capture

## Decision: Create dedicated Quick Capture draft records

**Decision**: Implement new `QuickCaptureDraft`, `QuickCaptureDraftImage`, and `DraftLifecycleEvent` persistence rather than representing quick captures as incomplete `Coin` rows.

**Rationale**: Normal collection views and stats are driven by `coins` plus wishlist/sold filters. A partially saved quick capture stored as a `Coin` would risk appearing in `/coins`, `/stats`, health scoring, value snapshots, wishlist/sold totals, tags/sets, and public/gallery workflows before explicit promotion. Dedicated draft tables satisfy the spec requirement that drafts are separate, incomplete, resumable, and excluded until promotion.

**Alternatives considered**:

- Incomplete `coins` rows with an `isDraft` flag: rejected because every existing coin query and sibling workflow would need new exclusion rules.
- Reuse `CoinIntakeDraft`: rejected because it is AI-generated, expires after 24 hours, stores a serialized candidate payload, and is coupled to `/coins/intake/commit`.

## Decision: Keep AI enrichment deferred

**Decision**: Quick Capture v1 performs no automatic attribution, agent calls, or image-based enrichment.

**Rationale**: The feature spec and constitution require deterministic manual v1 behavior. Existing agentic add remains available through `/add`, but Quick Capture must be usable with sparse human-entered fields and photos without depending on Python agent availability.

**Alternatives considered**:

- Call the existing Python intake draft service after image upload: rejected for v1 scope and service-boundary risk.
- Add optional background enrichment: rejected because it complicates lifecycle semantics and acceptance tests before manual draft/promotion is stable.

## Decision: Reuse image validation/upload/display patterns

**Decision**: Extract/reuse existing image extension, magic-byte/content-type, size, safe-path, file-save, and authenticated media display patterns for draft images.

**Rationale**: Normal coin upload accepts `.jpg`, `.jpeg`, `.png`, `.gif`, and `.webp`, checks content type from bytes, enforces size limits, stores under `UPLOAD_DIR`, and serves through authenticated media routes. Draft images must behave the same from the user's perspective and must not introduce a less secure upload path.

**Alternatives considered**:

- Store draft photos as data URIs in the database: rejected due to database bloat and divergence from existing media handling.
- Save draft photos only in browser local state until promotion: rejected because drafts must be resumable after leaving the flow.

## Decision: Transactional, idempotent promotion claim

**Decision**: Promotion is a service-level transaction that first claims an active draft for promotion, validates normal coin readiness, creates one `Coin`, converts/copies draft image rows into `CoinImage` rows, records lifecycle/journal/value snapshot data, and marks the draft promoted with `promoted_coin_id`. If the draft is already promoted, the service returns the existing promoted coin ID instead of creating another coin.

**Rationale**: Double taps and retries are explicit edge cases. A claim step prevents concurrent requests from both creating coins, while returning the existing promoted coin gives idempotent user-facing behavior.

**Alternatives considered**:

- Handler-level duplicate button disabling only: rejected because network retries and concurrent requests still need server-side safety.
- Best-effort promotion without a transaction: rejected by constitution Principle I and the feature's transactional requirement.

## Decision: Mobile/PWA-first UX using existing navigation and tokens

**Decision**: Add a PWA-visible Quick Capture entry point and responsive desktop route using the existing sidebar/nav system, `lucide-vue-next`, design tokens, `btn`/`chip` classes, and camera/file upload fallbacks.

**Rationale**: Existing `/add` already has PWA camera-first patterns, but Quick Capture needs a separate sparse draft flow. Reusing the visual/system primitives keeps desktop functional and avoids a parallel UI language.

**Alternatives considered**:

- Fold Quick Capture into the current Add Coin page: rejected because the Add page currently mixes manual and AI-assist normal-coin creation semantics, while Quick Capture must persist separate resumable drafts.
- Mobile-only route: rejected because the spec requires an accessible desktop route/navigation path.

## Decision: Contract tests around sibling workflow counts

**Decision**: Treat collection counts, wishlist/sold totals, health eligible count, normal collection views, and existing add/edit image flows as regression contracts.

**Rationale**: The constitution requires proving touched shared workflow contracts. Drafts are separate, but promotion touches normal coin creation and image records, so tests must verify both non-effects before promotion and intentional effects after promotion.

**Alternatives considered**:

- Manual QA only: rejected because count/idempotency regressions are automatable.
