# Data Model: Refine Coin Details Page for PWA and Desktop (#219)

## Entity: CoinDetailOverviewViewModel

Presentation model for the streamlined `/coin/:id` overview page.

### Fields

- `coinId` (number, required)
- `title` (string, required)
- `subtitle` (string, optional; ruler/secondary descriptor)
- `obverseImage` (object, optional)
- `reverseImage` (object, optional)
- `metadataRows` (array of `CoinMetadataRow`, required)
- `sectionLinks` (array of `CoinDetailSectionLink`, required)

### Validation Rules

- At least one media slot (`obverseImage` or `reverseImage`) must render a deterministic state (image or placeholder).
- `metadataRows` order must be stable for consistent scanability.
- `sectionLinks` must include all moved sections in v1 (`journal`, `notes`, `actions`, `analysis`).

## Value Object: CoinMetadataRow

Single row in the overview table/list-row metadata surface.

### Fields

- `key` (string enum, required; e.g., `denomination`, `mint`, `material`, `weight`, `diameter`, `grade`, `rarity`, `purchaseDate`)
- `label` (string, required)
- `value` (string, required; may be fallback text when source is empty)
- `emphasis` (enum: `default|accent`, optional)

### Validation Rules

- `label` must be human-readable and consistent with existing terminology.
- `value` must not be raw `null`/`undefined` in rendered output.

## Entity: CoinDetailSectionLink

Settings-style overview link row to a dedicated detail section page.

### Fields

- `id` (enum: `journal|notes|actions|analysis`, required)
- `title` (string, required)
- `routeName` (string, required)
- `routeParams` (object, required; includes `id`)
- `trailingIcon` (enum: `chevron-right`, required)

### Validation Rules

- Link must resolve to an authenticated route.
- `routeParams.id` must match overview `coinId`.

## Entity: CoinDetailSectionPageContext

Shared context for dedicated coin detail section pages.

### Fields

- `coinId` (number, required)
- `coinName` (string, required)
- `section` (enum: `journal|notes|actions|analysis`, required)
- `coinSummary` (object, optional; minimal identity/media payload for header continuity)

### Validation Rules

- Invalid `coinId` follows existing detail error/not-found handling.
- Section pages must preserve existing section-level capabilities and permissions.

## State Transitions

### Coin Detail Navigation

1. `overview_loaded` → `section_loaded` (user taps/clicks a settings-style section row)
2. `section_loaded` → `overview_loaded` (user navigates back)

### Transition Constraints

- Transition does not mutate coin data by itself.
- Section page actions retain existing side effects (journal writes, action operations, AI analysis updates) and refresh coin context as needed.
