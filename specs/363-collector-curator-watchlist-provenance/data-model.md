# Data Model: Coin Copilot Collector Curator

## Relationship overview

```text
User 1 ── 0..1 CollectorProfile
  │
  ├── existing Coin records
  └── existing CoinCopilotRun/checkpoint/event records

validated Coin Copilot dealer evidence
  └── explicit browser click
        └── existing POST /api/coins
              └── existing owner-scoped wishlist Coin
```

Only `CollectorProfile` is a new persistent entity. Curator guidance and dealer
result mappings are transient contract values. No wishlist action, audit,
stage, revision, outcome, watchlist evaluation, or provenance table is added.

## 1. CollectorProfile

One optional private context row per owner.

| Field | Storage/type | Rules |
|---|---|---|
| `id` | uint primary key | Internal only. |
| `user_id` | uint, unique, indexed, not null | Authenticated owner; never accepted from a request body. |
| `budget_min` | nullable decimal/float | Finite, `0..100000000`; must not exceed `budget_max` when both exist. |
| `budget_max` | nullable decimal/float | Finite, `0..100000000`; must not be below `budget_min` when both exist. |
| `currency` | varchar(3), nullable | Empty/null when no budget currency is stated; otherwise uppercase three-letter code accepted by existing application conventions. No conversion. |
| `preferred_periods_json` | text JSON array | 0..20 unique trimmed strings, each 1..100 characters. |
| `preferred_categories_json` | text JSON array | 0..20 unique trimmed strings, each 1..100 characters. |
| `excluded_categories_json` | text JSON array | 0..20 unique trimmed strings, each 1..100 characters. |
| `preferred_dealers_json` | text JSON array | 0..20 unique trimmed strings, each 1..200 characters. |
| `collecting_goals_json` | text JSON array | 0..20 unique trimmed strings, each 1..500 characters. |
| `created_at` | UTC timestamp | Set on first save. |
| `updated_at` | UTC timestamp | Set on each successful complete replacement. |

Constraints:

- unique `user_id`;
- foreign key to the owning `User`;
- JSON is decoded and validated by the service before repository writes;
- normalized duplicate detection is case-insensitive after trimming and
  collapsing whitespace;
- invalid input rejects the full update;
- an absent row is represented by the neutral response and is not created by
  GET.

This model intentionally has no version history, child goal table, snapshot
table, feature flag columns, sharing state, action authority, or audit state.

## 2. CollectorProfileValue (API/internal value)

The public response and internal Coin Copilot context use the same bounded
shape:

```text
budgetMin: number | null
budgetMax: number | null
currency: string | null
preferredPeriods: string[]
preferredCategories: string[]
excludedCategories: string[]
preferredDealers: string[]
collectingGoals: string[]
updatedAt: timestamp | null
isDefault: boolean
```

Rules:

- absent storage returns null scalars, empty arrays, `updatedAt: null`, and
  `isDefault: true`;
- no fallback preference or goal is invented;
- owner ID and internal row ID are excluded from the client payload and Python
  context;
- profile text is treated as untrusted advisory data in prompts;
- values are copied once for a Coin Copilot request so one response does not
  mix profile revisions.

## 3. CuratorGuidance (transient response semantics)

Curator guidance is the existing Coin Copilot final response produced from:

1. `collection_summary`;
2. `portfolio_review`;
3. `gap_analysis`; and
4. optional `CollectorProfileValue`.

It is not a new table or independent API resource. The response must
distinguish:

- **observed collection facts**: derived only from the three existing
  capabilities;
- **strengths/themes**: interpretation of those facts;
- **gaps/areas to explore**: suggestions, not collection facts;
- **profile influences**: only preferences, exclusions, budgets, dealers, or
  goals explicitly present in the captured profile;
- **limitations**: missing, sparse, or contradictory collection/context data.

Viewing, replaying, cancelling, or dismissing guidance does not mutate
`CollectorProfile`, `Coin`, wishlist, draft, auction, or settings data.

## 4. EligibleDealerResult (existing typed contract, additive projection)

The internal result already uses `CopilotSpecialistEvidence` and the Pydantic
`DealerListing`. The public projection remains an existing Coin Copilot
specialist result with these fields:

| Field | Rules |
|---|---|
| `capability` | Must equal `market_search` for wishlist eligibility. |
| `kind` | Must equal `dealer_listing`; all other kinds are ineligible. |
| `title` | Existing bounded validated title. |
| `sourceUrl` | Existing validated HTTPS dealer URL. |
| `observedAt` | Existing validated timestamp. |
| `confidence` | Existing `high|medium|low`; displayed but not a substitute for verification. |
| `verificationState` | `verified` or `partial`; partial requires visible uncertainty and confirmation under amended FR-012/014. |
| `description` | Optional typed dealer description. |
| `dealerName` | Optional typed dealer/source label. |
| `listedPrice` | Optional non-negative number. |
| `currency` | Optional three-letter currency paired with the price where present. |
| `availability` | `available|sold|unknown|null`; available/unknown are eligible, unknown requires visible uncertainty and confirmation. |
| `ruler` | Optional bounded string. |
| `denomination` | Optional bounded string. |
| `era` | Optional bounded string. |
| `material` | Optional bounded string. |

These fields are projected from the already validated internal result. The
browser must not infer them from the display `facts` array.

## 5. DealerResultSuggestionAdapter (transient UI value)

An eligible dealer result is adapted to the existing `CoinSuggestion`:

| Eligible dealer field | `CoinSuggestion` field | Rule |
|---|---|---|
| `title` | `name` | Required. |
| `description` | `description` | Empty when absent. |
| no typed category | `category` | Empty; existing safe fallback/confirmation rules decide the destination value. |
| `era` | `era` | Existing resolver reconciles it with live options. |
| `ruler` | `ruler` | Empty when absent. |
| `material` | `material` | Existing mapper uses supported value or `Other`. |
| `denomination` | `denomination` | Empty when absent. |
| `listedPrice` + `currency` | `estPrice` | Formatted without conversion; empty if price is absent or unsafe. |
| no required image field | `imageUrl` | Empty unless the existing typed contract later supplies one; source-page scrape remains the first image path. |
| `sourceUrl` | `sourceUrl` | Passed to the existing bounded reference mapping. |
| `dealerName` | `sourceName` | Empty or a neutral dealer label when absent. |

The adapter does not add acquisition fields, ownership state, storage, sold
state, invoices, inventory values, draft state, auction values, or arbitrary
provider data.

## 6. Existing Wishlist Coin

`buildWishlistCoinPayload` remains authoritative for the new coin payload:

- `name`, `category`, `material`, `denomination`, `ruler`, `era`;
- bounded descriptive `notes`;
- `referenceUrl` and `referenceText`;
- supported structured catalog references when present;
- `isWishlist: true`;
- interpretable `currentValue`.

`CoinService.CreateCoin` remains authoritative for category/era validation,
owner assignment from the handler, persistence, structured references, and
value snapshots. Image attachment remains a separate best-effort operation
after successful creation.

Never populate:

- purchase date, purchase price, purchase location;
- invoice/SKU/inventory values;
- storage location or slot;
- sold fields;
- owned-collection state;
- quick-capture or deep-analysis workflow state;
- auction state.

## 7. Duplicate and state behavior

```text
Dealer card:
  ineligible -> no button
  eligible -> button idle
  click -> addingIdx guard
       -> category/era resolution
          cancel -> no write
          resolve -> canonical create transaction
             same owner + existing wishlist reference URL -> duplicate status, no insert
             otherwise -> one wishlist Coin
                -> optional image attachment
                   success -> saved with image
                   failure -> saved without image + warning
       -> addedSet marks the card after successful coin creation
```

The transaction-scoped reference URL lookup reuses
`CoinRepository.FindWishlistByReferenceURL`; it does not create a second source
identity, action record, or audit record.

## 8. Migration and deletion

Migration is additive:

1. add `collector_profiles`;
2. add unique owner index and owner foreign key;
3. do not backfill rows;
4. existing owners receive neutral defaults until they save.

Normal account deletion must remove the owner's profile with the rest of their
private data. No separate retention worker is needed. Removing Feature 363
code can leave the additive table inert; no destructive rollback is required.
