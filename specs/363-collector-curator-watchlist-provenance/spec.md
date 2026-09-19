# Feature Specification: Coin Copilot Collector Curator

**Feature Branch**: `beta` (existing worktree; no feature branch created)
**Created**: 2026-09-18
**Rewritten**: 2026-09-19
**Status**: Draft — user-approved SMALL F015 scope
**Input**: Extend the shipped Coin Copilot harness with lightweight private
collector context, read-only curator guidance, and restoration of the existing
dealer-result **Add to Wishlist** behavior.

## Scope

Feature 363 is a small, proportional extension of shipped Coin Copilot
capabilities. It adds:

1. one lightweight, private, owner-scoped collector profile used as optional
   context;
2. read-only curator guidance composed from the existing collection summary,
   portfolio review, and gap-analysis capabilities; and
3. the existing **Add to Wishlist** action on typed Coin Copilot cards for
   verified dealer listings that are currently available.

This feature does not create a new agent, orchestration, browsing, persistence,
or write platform. Profile storage follows existing owner-scoped application
patterns rather than introducing a new persistence subsystem. The AI remains
read-only. A wishlist coin can be created only after the authenticated owner
explicitly selects the UI action on an eligible dealer result. The action
reuses the canonical wishlist coin creation flow and its existing validation
and safeguards.

This scope applies Constitution Principle IV: the change restores one complete
user workflow and adds only the minimum private context and read-only guidance
needed to support it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Save an available dealer result to the wishlist (Priority: P1)

As a collector reviewing Coin Copilot search results, I want to add a verified,
currently available dealer listing to my wishlist so I can retain a coin I may
want to buy.

**Why this priority**: This restores the concrete acquisition-discovery
workflow requested for F015 without granting write authority to the AI.

**Independent Test**: Ask Coin Copilot for Aurelian coins under $500, select
**Add to Wishlist** on an eligible dealer result, resolve category or era when
prompted, and verify exactly one owner-scoped wishlist coin is created through
the existing coin creation flow.

**Acceptance Scenarios**:

1. **Given** a typed Coin Copilot result card represents a verified dealer
   listing that is currently available, **When** the owner reviews the card,
   **Then** the card shows an explicit **Add to Wishlist** button.
2. **Given** an eligible dealer result, **When** the owner selects **Add to
   Wishlist** and any required category or era confirmation succeeds, **Then**
   exactly one coin is created for that owner with wishlist status set.
3. **Given** Coin Copilot displays or recommends a dealer result, **When** the
   owner does not select **Add to Wishlist**, **Then** no coin, wishlist item,
   collection item, or draft is created or changed.
4. **Given** a dealer result is unverified, unavailable, sold, withdrawn,
   malformed, or lacks the minimum data required by the existing creation
   flow, **When** the card is displayed, **Then** **Add to Wishlist** is not
   offered.
5. **Given** a result is an auction result rather than a dealer listing,
   **When** its card is displayed, **Then** **Add to Wishlist** is not offered.
6. **Given** category or era needs owner confirmation, **When** the owner
   cancels that confirmation, **Then** no wishlist coin is created.
7. **Given** the same eligible result is submitted repeatedly or already maps
   to an existing wishlist coin under the existing duplicate rules, **When**
   creation is attempted, **Then** no duplicate wishlist coin is created and
   the owner receives clear status.
8. **Given** the wishlist coin is created but a dealer image cannot be safely
   attached, **When** the flow completes, **Then** the wishlist coin remains
   saved and the image failure is reported without creating another coin.

---

### User Story 2 - Maintain private collector context (Priority: P2)

As a collector, I want a lightweight private profile for my collecting
interests so Coin Copilot can tailor curator guidance to my goals and
constraints.

**Why this priority**: Small, explicit owner context improves the relevance of
existing guidance without introducing agent memory or global settings.

**Independent Test**: Save, reload, edit, and clear a profile for one owner;
verify it is used as optional curator context, remains invisible to other
owners and public users, and does not alter application-wide settings.

**Acceptance Scenarios**:

1. **Given** an authenticated owner with no profile, **When** they open their
   collector profile, **Then** they receive neutral empty values and can use
   curator guidance without completing the profile.
2. **Given** an owner enters a budget range, preferred periods or categories,
   excluded categories, preferred dealers, or collecting goals, **When** they
   save valid values, **Then** those values are available as private context
   for that owner's later curator requests.
3. **Given** an owner edits or clears profile values, **When** the save
   completes, **Then** subsequent curator guidance uses the updated values and
   does not infer deleted preferences.
4. **Given** another owner, invited friend, public visitor, or administrator
   acting outside the profile owner's session, **When** they request the
   profile or profile-derived context, **Then** no private profile data is
   disclosed.
5. **Given** invalid or contradictory profile values, **When** the owner tries
   to save, **Then** the update is rejected with field-specific guidance and
   the previously saved profile remains unchanged.

---

### User Story 3 - Request read-only curator guidance (Priority: P3)

As a collector, I want Coin Copilot to summarize my collection and identify
strengths and gaps in light of my private collector context so I can decide
what to explore next.

**Why this priority**: It composes already shipped analysis capabilities and
adds user value without adding another autonomous workflow.

**Independent Test**: Run curator guidance against a controlled collection and
profile; verify the response uses the existing collection summary, portfolio
review, and gap analysis, explains relevant profile influence, and performs no
writes.

**Acceptance Scenarios**:

1. **Given** an owner has collection data and optional collector context,
   **When** they request curator guidance, **Then** Coin Copilot presents a
   collection summary, strengths, gaps, and possible areas to explore based on
   the existing analysis capabilities.
2. **Given** profile context affects the guidance, **When** the response is
   shown, **Then** it identifies which stated preferences or goals influenced
   the guidance.
3. **Given** profile context is empty, **When** guidance is requested, **Then**
   Coin Copilot uses neutral defaults and does not invent preferences.
4. **Given** collection data is sparse, missing, or contradictory, **When**
   guidance is generated, **Then** observed facts, suggestions, and
   limitations are distinguishable.
5. **Given** the owner views, reruns, cancels, or dismisses guidance, **When**
   the interaction ends, **Then** no collection, wishlist, draft, profile, or
   application-setting data is changed.

### Edge Cases

- The result changes from available to unavailable before the owner selects
  the action; the action must not create a wishlist coin once the result is
  known to be ineligible.
- A dealer result omits category or era, or uses a value outside the owner's
  configured options; the existing category/era confirmation behavior applies.
- A dealer card has valid coin data but no usable image; creation can succeed
  without an image.
- A source image is unsafe, unreachable, empty, or unsupported; it is not
  attached and does not cause a second coin to be created.
- A source price is missing or cannot be interpreted; the wishlist value
  remains empty rather than inventing a value.
- Collector context conflicts with observed collection data; the response
  reports the conflict rather than treating context as a collection fact.
- Coin Copilot or one of the existing analysis capabilities is unavailable;
  the owner receives a clear unavailable or fallback state and no mutation
  occurs.

## Requirements *(mandatory)*

### Functional Requirements

#### Private collector profile

- **FR-001**: The system MUST provide at most one lightweight collector profile
  for each authenticated owner.
- **FR-002**: The profile MUST support optional budget minimum and maximum,
  currency, preferred periods or categories, excluded categories, preferred
  dealers, and free-text collecting goals.
- **FR-003**: An absent or empty profile MUST produce neutral context and MUST
  NOT block use of Coin Copilot or curator guidance.
- **FR-004**: Profile values MUST be validated, bounded, and saved as one
  complete update so invalid input cannot cause a partial change.
- **FR-005**: Profile reads and writes MUST use the authenticated owner's
  identity. Profile values and profile-derived context MUST NOT be visible to
  another owner, invited friend, or public visitor, and administrator status
  alone MUST NOT grant access.
- **FR-006**: Collector profile values MUST remain separate from global admin
  settings and MUST be used only as advisory context. They MUST NOT authorize
  purchases, writes, or autonomous actions.

#### Read-only curator guidance

- **FR-007**: Curator guidance MUST compose the shipped collection summary,
  portfolio review, and gap-analysis capabilities within the existing Coin
  Copilot harness.
- **FR-008**: Curator guidance MUST describe collection strengths, notable
  themes, gaps, and possible areas to explore while distinguishing collection
  facts from suggestions.
- **FR-009**: When profile values influence guidance, the response MUST explain
  the relevant preference, constraint, or goal. Missing context MUST remain
  unknown and MUST NOT be invented.
- **FR-010**: Curator requests and responses MUST be read-only. The AI MUST NOT
  create or alter collection coins, wishlist coins, quick-capture drafts,
  collector profiles, or application settings.
- **FR-011**: Curator guidance MUST NOT evaluate, score, or rank saved wishlist
  items or market listings and MUST NOT introduce a watchlist subsystem.

#### Dealer-only Add to Wishlist

- **FR-012**: **Add to Wishlist** MUST be available only on a typed Coin
  Copilot specialist card whose source is a dealer listing, whose evidence is
  verified, and whose listing is currently available.
- **FR-013**: Auction results and all unverified, partially verified, unknown,
  unavailable, sold, ended, or withdrawn results MUST NOT expose or invoke the
  restored wishlist action.
- **FR-014**: The AI MUST remain read-only. Only the authenticated owner's
  explicit selection of **Add to Wishlist** in the UI MAY initiate creation.
  Conversational text, tool output, card rendering, replay, or automatic agent
  behavior MUST NOT initiate a write.
- **FR-015**: The action MUST reuse the existing canonical wishlist coin
  creation path, including `useCoinSearchChat`,
  `buildWishlistCoinPayload`, `createCoin`, category/era confirmation, image
  attachment, and duplicate safeguards where applicable. It MUST NOT add a
  staged, revisioned, expiring, revocable, or parallel transaction flow.
- **FR-016**: Before creation, category and era values MUST be reconciled with
  the owner's current allowed values. When existing confirmation behavior
  requires owner input, cancellation MUST create nothing.
- **FR-017**: The created record MUST be an owner-scoped wishlist coin, not a
  collection coin and not a quick-capture draft.
- **FR-018**: Mapping from the dealer result MUST be limited to fields already
  accepted by the canonical wishlist creation flow: name, category, material,
  denomination, ruler, era, descriptive notes, source reference URL and text,
  supported catalog references, wishlist status, and an interpretable current
  value.
- **FR-019**: The action MUST NOT populate collection-only acquisition fields,
  including purchase date, actual purchase price, invoice or inventory data,
  storage location, sold state, or collection ownership state. It MUST NOT
  populate draft-only workflow state or overwrite manually maintained data on
  an existing coin.
- **FR-020**: Missing, unsupported, or unverified source values MUST remain
  empty or use the existing safe category/material fallback; the system MUST
  NOT invent listing facts or coerce data into unrelated fields.
- **FR-021**: Existing duplicate safeguards MUST prevent repeated clicks,
  retries, or an already-saved eligible result from creating more than one
  wishlist coin for the owner.
- **FR-022**: Existing safe image attachment behavior MAY attach one dealer
  image after wishlist creation. Image failure MUST NOT roll back a successful
  coin creation, populate another field, or trigger creation of a replacement
  coin.
- **FR-023**: The owner MUST receive clear feedback when creation succeeds,
  confirmation is cancelled, the result is ineligible, a duplicate is
  prevented, or optional image attachment fails.

### Key Entities

- **Collector Profile**: Private, optional context belonging to one owner. It
  contains bounded collecting preferences and goals and grants no write
  authority.
- **Curator Guidance**: A transient, read-only response composed from existing
  owner-scoped collection analysis and optional profile context.
- **Eligible Dealer Result**: An existing typed Coin Copilot specialist result
  that represents a verified, currently available dealer listing. It is the
  only result type eligible for the restored action.
- **Wishlist Coin**: The existing canonical coin record created with wishlist
  status. It remains distinct from an owned collection coin and from a
  quick-capture draft.

## Dependencies and Existing Boundaries

- Coin Copilot and its shipped typed specialist-result contract remain the
  agentic harness and source of dealer result cards.
- Existing collection summary, portfolio review, and gap analysis remain the
  only analysis capabilities composed for curator guidance.
- Existing coin creation owns validation, persistence, owner scope, field
  limits, and wishlist status.
- Existing category and era options and confirmation behavior remain
  authoritative.
- Existing image proxy, scrape, validation, and upload behavior remains
  authoritative and best-effort.
- Existing duplicate safeguards remain authoritative; this feature does not
  introduce a second identity or transaction model.
- Collection records, wishlist records, and quick-capture drafts retain their
  current field and lifecycle boundaries.

## Non-Goals

- Any change to the auction subsystem; auction results are not eligible for
  this action.
- Watchlist evaluation, candidate scoring, market-result ranking, or a
  watchlist subsystem.
- Provenance-risk scoring, suspicious-listing detection, duplicate-image
  analysis, forensic review, fraud detection, or authenticity claims.
- Wishlist-action stage, revision, revoke, expiry, or confirm endpoints.
- Wishlist action or audit tables, durable action workflows, or a parallel
  transaction platform.
- A new agent orchestration platform, AI provider, browsing mechanism, search
  provider, or persistence subsystem.
- Autonomous, conversational, scheduled, or bulk wishlist creation.
- AI-initiated writes, arbitrary write tools, purchasing, bidding, offers, or
  payments.
- Replacing collection summary, portfolio review, gap analysis, canonical coin
  creation, category/era confirmation, image attachment, or duplicate
  safeguards.
- Converting a dealer result into an owned collection coin or quick-capture
  draft.
- Changing wishlist availability tracking or other existing wishlist
  lifecycle behavior.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In owner-isolation tests, 100% of profile reads, profile writes,
  profile-derived context, and created wishlist coins remain scoped to the
  authenticated owner, with zero disclosure to another owner or public user.
- **SC-002**: In profile acceptance tests, 100% of valid bounded profiles can
  be saved, reloaded, edited, and cleared, while 100% of invalid fixtures are
  rejected without partial updates.
- **SC-003**: In controlled curator tests, 100% of responses use only the
  existing collection summary, portfolio review, gap-analysis output, and
  optional profile context; all identified limitations are visible and zero
  data writes occur.
- **SC-004**: Across result-card eligibility tests, 100% of verified,
  currently available dealer results expose **Add to Wishlist**, while 0% of
  auction, unverified, partially verified, unknown, or unavailable results
  expose the action.
- **SC-005**: At least 95% of test users can add an eligible dealer result to
  the wishlist on their first attempt in under 60 seconds, excluding time
  spent browsing the external dealer page.
- **SC-006**: Mapping tests place 100% of supported dealer values into their
  corresponding wishlist fields and place 0 values into collection-only or
  draft-only fields.
- **SC-007**: Cancellation, repeated-click, retry, and existing-duplicate tests
  create at most one owner-scoped wishlist coin per eligible result.
- **SC-008**: In all tested flows where no explicit **Add to Wishlist** click
  occurs, Coin Copilot creates zero wishlist, collection, or draft records.
- **SC-009**: For every eligible fixture with a safe reachable image, the
  existing attachment flow attaches at most one image; for every image-failure
  fixture, the saved wishlist coin remains usable and no additional coin is
  created.
- **SC-010**: Regression checks confirm no behavioral change to collection
  creation, quick-capture drafts, existing wishlist management, or ineligible
  Coin Copilot result cards.

## Assumptions

- The target user is an authenticated collection owner. Invited friends and
  public visitors do not manage collector context or create wishlist coins.
- “Verified” and “currently available” are supplied by the existing typed Coin
  Copilot dealer-result contract; Feature 363 does not create another
  verification or availability mechanism.
- Collector profile context is advisory and is supplied to the existing
  harness only for the requesting owner and request.
- Currency conversion is not introduced. A price that the existing wishlist
  payload cannot safely interpret remains empty.
- Optional image attachment follows the existing source-safety and upload
  rules and is not required for a valid wishlist coin.
