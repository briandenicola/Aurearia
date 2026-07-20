---
id: F028
title: "Track collection progress toward every Roman Emperor (West + East, to 476 AD)"
status: backlog
priority: P2
effort: L
value: 5
risk: 2
owner: unassigned
created: 2026-07-20
updated: 2026-07-20
---

# F028 — Track collection progress toward every Roman Emperor (West + East, to 476 AD)

## Summary

The app was originally built around tracking ancient Roman coins, but there is
no way today to see collection progress against a *historical* completion
goal — only against user-defined sets. This feature adds an opt-in tracker,
under Stats, that shows how a user is trending toward owning at least one
coin of every Western Roman Emperor (27 BC – 476 AD) and every Eastern Roman
Emperor up to the same 476 AD cutoff (Byzantine emperors after that point are
explicitly out of scope for now — a natural follow-on, not part of this
card). Progress is grouped by dynasty/era, and displayed using the app's
existing "Tray" museum-cabinet visual (`MuseumTray.vue`/`MuseumTrayWell.vue`,
already used at `/tray`) — owned emperors show the user's actual coin, unowned
emperors render as an empty well using the tray's existing placeholder state.
The feature is disabled by default and enabled per-user in Settings, matching
the existing `coinOfDayEnabled`-style opt-in pattern.

## Acceptance criteria

- [ ] A per-user setting (`User.EmperorTrackerEnabled`, default off) can be
      turned on/off from Settings → Account, next to the existing
      `coinOfDayEnabled` toggle.
- [ ] When enabled, a new Stats sub-page (e.g. `/stats/emperors`) becomes
      available, showing overall completion (X of Y emperors owned, as both
      a count and a percentage) plus a per-dynasty/per-era completion
      breakdown (e.g. "Julio-Claudian — 3 of 5 (60%)", "Severan — 1 of 8
      (13%)") — this is a first-class stats display, not just an implied
      byproduct of the tray view: the completion numbers must be visible
      even before scrolling to a given dynasty's wells.
- [ ] The default (and primary) goal is the **commonly accepted Augustuses**
      only — i.e. every `role: emperor` figure, exactly as already scoped
      above. Usurpers were, by definition, never accepted as legitimate
      emperors, so they (and empresses, and Caesar/other figures) are
      excluded from this goal by default — this keeps the core "collect all
      the emperors" goal realistic and unambiguous for every user out of
      the box.
- [ ] Users can optionally expand what they track via three independent
      Settings toggles (default off): **show usurpers**, **show empresses**,
      **show other figures** (Caesars who never acceded + Julius Caesar).
      Enabling one adds its own separate, independently-tracked section to
      `/stats/emperors` (own completion count/percentage, own dynasty/era
      grouping, own tray rendering) — it does **not** get merged into the
      core emperor completion number. This lets a user who wants a bigger
      or more textured challenge opt into tracking more, without changing
      what "100%" means for anyone who leaves the defaults alone.
- [ ] Each dynasty/era section renders its emperors using the existing tray
      visual (`MuseumTray`/`MuseumTrayWell`) — an owned emperor's well shows
      the user's real coin (image, click-through to coin detail, exactly like
      `/tray` today); an **unowned emperor's well is a visible placeholder**
      (the tray's existing "no image" state), labeled with the emperor's
      name, so a user can see at a glance exactly which emperors they're
      missing, not just a number.
- [ ] The page surfaces a "what to pursue next" list of missing emperors to
      help the user prioritize acquisitions — see the Suggestions section
      below for scope/phasing.
- [ ] Matching a coin to an emperor is structured, not fuzzy-text: when a
      coin's Category is Roman, the coin form offers an **optional**
      type-ahead picker over a curated list of Roman imperial figures
      (emperors, empresses, heirs/Caesars, usurpers), each tagged with a
      `role`. Only figures tagged `role: emperor` count toward this
      feature's completion stat — picking "Livia" (Augustus's wife, never
      an emperor herself) must never be counted as an Augustus coin. The
      picker never forces a choice: leaving it blank is always valid, and
      the existing free-text `Ruler` field is unaffected/unchanged.
- [ ] The imperial-figure picker supports filtering the list by `role`
      (Emperor / Empress / Caesar / Usurper / Other) — the curated dataset
      runs to ~165 entries (see Notes), so browsing or searching it without
      a way to narrow by role is impractical.
- [ ] Western and Eastern Roman emperors are both covered, capped at 476 AD
      by *reign start* (an emperor already reigning by 476 is included in
      full even if their reign continued past it, e.g. Zeno; no emperor
      whose reign began after 476, e.g. Anastasius I, is included in v1).
- [ ] Disabling the setting hides the Stats sub-page entirely; no background
      job or scheduled work runs for users who haven't opted in.

## Proposed approach

### Data: new static reference dataset, not the existing Coin Set / Target system

`models.CoinSetTarget` (used by "defined"/"goal" Coin Sets, see
`repository/set_repository.go`) looks like a natural fit at first glance — it
already models a named target with a completion percentage
(`GetSetCompletion`) — but doesn't actually fit:

1. `GetSetCompletion` only matches coins that are **manually added as members
   of that specific set** (`GetCoinsInSet`), not the user's whole collection.
   Requiring a user to manually add every Roman coin they own to a special
   "Emperors" set before this works at all defeats the point of an automatic
   tracker.
2. `CoinSetTarget.MatchRules` (a generic `*JSONObject`) exists in the model
   but is **not read anywhere** — `matchCoinToTarget` only matches on
   Year/MintMark/Denomination/Country(-via-Ruler-substring)/Material, tuned
   for modern (US mint-mark style) coin sets, not ruler identity.

Recommendation: a new, purpose-built static reference table (seeded, not
user-editable) — `models.RomanImperialFigure{ID, Name, Aliases []string-ish,
Role (emperor|empress|caesar|usurper|other), Region (west|east), Dynasty,
ReignStart, ReignEnd, SortOrder, RarityTier}` — scoped to *imperial figures*
broadly, not just emperors, since real coins commonly depict empresses,
heirs/Caesars, and usurpers who were never emperor themselves (see Matching
strategy below for why this matters). Per-user progress is computed on
request against the user's full active (non-wishlist, non-sold — confirm in
Open Questions) collection, the same way `AuctionLotService.Recommend`/
`MarketSignal` compute their results live rather than maintaining a
stored/cached table. Collections are small enough (hundreds, not millions of
coins) that this doesn't need a background job.

### Matching strategy: structured selection, not fuzzy text

`Coin.Ruler` is free text today (`binding:"max=200"`, no enum, no existing
canonical list anywhere in the codebase), and real collections have
"Augustus", "Octavian", "Divus Augustus", spelling variants, etc. — a naive
exact- or fuzzy-match against that field would under-count badly and is the
wrong foundation to build on.

Instead: add a new, optional `Coin.RomanImperialFigureID *uint` (nullable
FK into `RomanImperialFigure`), surfaced in the coin form as a type-ahead
picker **gated on `Category == Roman`**. This directly solves the fuzzy
alias-matching risk by replacing it with an explicit, unambiguous selection
made once at coin-entry time — no normalization/substring-matching code
needed at all for coins entered this way. Critically, the picker's list is
not limited to the ~90 emperors: it includes every commonly-depicted
imperial figure (empresses, heirs/Caesars, usurpers), each carrying a
`role`. Selecting a non-emperor figure (e.g. Livia, Faustina, a Caesar who
never acceded) is completely valid and simply doesn't count toward the
emperor-completion stat — the field is honestly "who's depicted on this
coin," not artificially forced into "which emperor does this map to." The
lookup/search endpoint behind the picker accepts an optional `role` filter
so the frontend can offer role tabs/chips (Emperor / Empress / Caesar /
Usurper / Other) instead of forcing users to search the full ~165-entry
list by name alone.

The picker is optional and additive: it never replaces or requires the
free-text `Ruler` field, and a coin can be left unmatched. Given the app's
current user base has at most a few hundred coins entered, not thousands,
there is **no bulk fuzzy-matching migration for existing coins** in scope —
existing coins simply show as unmatched until a user opens them and picks
the figure themselves (a one-time, per-coin, user-driven action, not a
background job or automated best-effort guess). This meaningfully shrinks
the feature's original biggest risk: there's no fuzzy-matcher to get wrong,
tune, or need an override affordance for.

### Suggestions: which missing emperor(s) to pursue next

Phase this the same way F023 (bid recommendation) was phased into a
historical-data-only V1 and a market-data-assisted V2 — don't build the
expensive version before the cheap one proves useful.

**V1 — static, no agent/network call.** The curated `RomanImperialFigure`
dataset (see Data section) already carries a `RarityTier` field
(`common | scarce | rare | very_rare`, hand-curated alongside the
name/dynasty/alias/role data — e.g. an Augustus or Constantine I denarius is
routinely available and inexpensive; a Romulus Augustulus or a legitimate
Otho coin is a genuine numismatic rarity most collectors never own). The
suggestions list is simply the user's missing emperors sorted with the most
*attainable* ones first (common/scarce before rare/very_rare), optionally
tie-broken by dynasty proximity to eras the user already collects in. This
requires no new backend service, no agent call, and no live pricing — it's a
sort over already-loaded data, computable entirely in
`AuctionLotService`-adjacent Go code or even client-side.

**V2 — market-data assisted (stretch, mirrors F023's `bid_market_signal.py`
pattern).** Once V1 ships and the rarity-tier heuristic has been used for a
while, consider reusing the same Python-agent market-search approach F023 V2
built: for a specific missing emperor the user picks, search live auction
results for coins of that ruler and surface a rough price range / recent
availability, the same way `bid_market_signal.py` does today for a specific
tracked lot. This is meaningfully more effort (new agent request shape keyed
by emperor name/dynasty instead of a tracked lot, a new Go proxy method, new
UI) and depends on the Ruler-matching problem already being solved well
enough that "search for coins of Emperor X" produces relevant results — do
not start V2 before V1's rarity-tier sort has been validated against real
usage.

Do not build a live web-search-backed suggestion engine as the *first*
version of this feature — that repeats the exact overreach F023 V1
deliberately avoided (a recommendation that's only as good as data/matching
no one has stress-tested yet).

### Dynasty/era scope

The actual curated dataset is `specs/_backlog/F028-imperial-figures.md` — a
first pass exists (reviewed 2026-07-20) covering:

**Western Roman (27 BC – 476 AD)**, roughly: Julio-Claudian → Flavian →
Nerva–Antonine → Severan → Crisis of the Third Century (Maximinus Thrax
through Carinus/Numerian) → Tetrarchic/Diocletianic → Constantinian →
Valentinianic → Theodosian (West) → the fragmented last Western emperors
(Petronius Maximus through Romulus Augustulus, 455–476).

**Eastern Roman**: Theodosian (East: Arcadius, Theodosius II, Marcian) →
Leonid (Leo I, Leo II, Zeno, the usurper Basiliscus). The 476 cutoff rule
was resolved during curation: an emperor is in scope if their reign **began**
on or before 476 AD, even if it continued after. **Zeno is explicitly
included** in full (474–491) on that basis, since his first reign began in
474 — his restored reign is not truncated. Anastasius I (r. 491–518) is
excluded since his reign began after the cutoff.

~96 entries carry `role: emperor` (the ones that drive the completion
stat); the full dataset with usurpers, Caesars, empresses, and Julius Caesar
(included as a non-emperor `role: other` precursor entry, per explicit
request) runs to ~165 rows. See the curated file for the complete list,
per-entry rationale, and remaining open follow-ups (co-emperor counting
still needs final sign-off; rarity tiers are an unsourced first guess).

### UI

- Coin form: a new optional type-ahead field, "Imperial figure," shown only
  when Category is Roman, sitting alongside the existing free-text `Ruler`
  input (not replacing it). Backed by the curated `RomanImperialFigure`
  list; supports leaving it unset. Includes a `role` filter (tabs or chips —
  Emperor / Empress / Caesar / Usurper / Other) so the ~165-entry list is
  actually browsable, not just searchable by typing a name. The backing
  lookup/search endpoint (see Matching strategy) accepts a `role` query
  param so this is a server-side filter, not a client-side scan of the
  full list.
- Settings toggle: `SettingsAccountSection.vue`, same visual pattern as the
  existing `coinOfDayEnabled` checkbox — plus three more checkboxes, shown
  only once the main toggle is on: "Also track usurpers," "Also track
  empresses," "Also track other figures (Caesars & precursors)," each
  independently off by default.
- New page under Stats (existing sibling pages: `/stats/mint-map`,
  `/stats/timeline`, `/stats/health`, `/stats/value-trends`,
  `/stats/investment-breakdown`, `/stats/distribution` — add
  `/stats/emperors` alongside them), gated on
  `auth.user.emperorTrackerEnabled`. The page always shows the primary
  Emperor section first; any of the three optional categories the user has
  enabled render as additional sections below it, each with its own
  dynasty/era grouping and its own completion count/percentage — never
  merged into the primary emperor stat.
- Reuses `MuseumTray.vue`/`MuseumTrayWell.vue` as-is where possible: an
  unmatched emperor is just a `TrayCoin`-shaped entry with no `images`, which
  already renders `MuseumTrayWell`'s built-in placeholder (a dim coin icon)
  with no code changes needed there. Needs a synthetic, non-colliding `id`
  scheme for placeholder wells (e.g. negative IDs) and `interactive: false`
  (or a different click behavior — see Open Questions) so clicking an empty
  well doesn't try to navigate to a nonexistent coin detail page.

## Constitution alignment

- Principle I (Clear Layered Architecture) — new repository/service/handler
  trio for the emperor reference data + per-user progress computation;
  reuses existing tray components at the presentation layer rather than
  forking them.
- Principle III (Strict Types and Explicit Contracts) — the
  `RomanImperialFigure` dataset and match-result shape need explicit Go
  structs / TS types, not ad hoc maps.
- Principle IV (Simple Complete Changes) — do not extend
  `CoinSetTarget`/`matchCoinToTarget` for this; that system's assumptions
  (set-scoped membership, US-coin-oriented match fields) don't fit and
  bending it would make both features harder to reason about.
- Principle VI (Consistent User Experience) — reuses the existing tray visual
  language exactly rather than inventing a new "collection progress" UI
  pattern.

## Open questions

- [x] Who curates the canonical `RomanImperialFigure` dataset, and where
      does it live? A first pass (~165 figures, ~96 of them `role: emperor`)
      has been curated and checked in at
      `specs/_backlog/F028-imperial-figures.md`. Still open: task #30 needs
      to decide how it's represented in Go (hardcoded seed function, matching
      the existing `seedMintLocations` pattern in `database/database.go`, is
      the presumed default absent a reason to do otherwise).
- [ ] Do sold coins count toward "collected" (you owned it at some point) or
      only currently-owned, non-wishlist coins? Leaning toward
      currently-owned only, matching how the rest of the app treats
      "collection" vs "sold" vs "wishlist," but worth confirming.
- [ ] Should an empty (unmatched) emperor well be non-interactive, or should
      clicking it deep-link to "Add Coin" with that emperor pre-selected in
      the new "Imperial figure" picker? Nice-to-have, not required for v1.
- [x] Exact 476 AD cutoff handling for Eastern emperors whose reigns straddle
      that date (notably Zeno) — **resolved**: in scope if the reign began
      on or before 476, even if it continued after. Zeno is included in
      full (474–491); Anastasius I (began 491) is not.
- [ ] Should co-emperors (e.g. Lucius Verus, Geta — both `role: emperor`,
      both in the default goal) count as separate tracked entries, or be
      folded into the primary emperor's entry? The curated dataset's first
      pass leans toward separate entries (they minted coinage under their
      own name/portrait), but this is not yet a final sign-off — still
      affects the default goal's total count and user expectations if
      reversed. (Note: this question is now scoped to co-emperors only —
      usurpers are a separate, opt-in, always-independently-tracked
      category per the new visibility toggles, so there's no
      folding/merging ambiguity for them.)
- [ ] Who curates the per-emperor `rarityTier` used to sort V1 suggestions,
      and against what standard (auction frequency? price? both?) — same
      content-ownership question as the core dataset, called out separately
      here since it's easy to under-scope as "just add a column."
- [ ] Is a V2 (agent-assisted, live market search) suggestion engine even
      wanted, or is the V1 static rarity sort sufficient long-term? Don't
      pre-approve V2 scope now — revisit after V1 ships and is used.
- [ ] Should the three optional categories (usurpers, empresses, other) each
      also get their own V1 "what to pursue next" suggestions list once
      enabled, or is that list a core-emperor-only feature for v1? Leaning
      toward core-emperor-only for v1 to keep scope bounded, with the
      optional categories being pure display/tracking (no suggestions) until
      there's a reason to add it.

## Notes

Prior art investigated during planning:
- `/tray` (`TrayViewPage.vue`, `components/tray/MuseumTray.vue`,
  `MuseumTrayWell.vue`, `utils/trayLayout.ts`) — the existing museum-cabinet
  display this feature is asked to reuse. Confirmed `MuseumTrayWell.vue`
  already has a graceful no-image placeholder state, which is exactly what
  an "emperor not yet collected" well needs with no component changes.
- `models.CoinSetTarget` / `CoinSetCompletion` / `matchCoinToTarget`
  (`repository/set_repository.go:436-702`) — the existing "defined set"
  completion-tracking system. Investigated as a possible reuse target;
  documented above why it doesn't fit as-is (set-scoped membership, unused
  `MatchRules` field, US-coin-oriented matching).
- `User.CoinOfDayEnabled` (`models/user.go`) and its toggle in
  `SettingsAccountSection.vue` — the exact per-user opt-in pattern to mirror
  for `EmperorTrackerEnabled`.
- `AuctionLotService.Recommend`/`MarketSignal` (F023) — precedent for
  computing a derived result live per-request rather than maintaining a
  stored/cached table, applicable here since per-user emperor-match
  computation is cheap at realistic collection sizes.

Tracking issue: [#501](https://github.com/briandenicola/Aurearia/issues/501)
(request + research; implementation not yet started).

Curated dataset: `specs/_backlog/F028-imperial-figures.md` (first pass,
~165 imperial figures, ~96 `role: emperor`).

## History

- 2026-07-20: created (status: backlog) — feature request to track
  Western + Eastern Roman Emperor collection completeness (to 476 AD),
  grouped by dynasty/era, displayed via the existing tray UI, opt-in per
  user. Byzantine (post-476) emperors explicitly deferred to a future card.
- 2026-07-20: refined — made completion stats (overall + per-dynasty
  percentage) an explicit, first-class acceptance criterion rather than an
  implied byproduct of the tray view, and added a "what to pursue next"
  suggestions requirement, phased V1 (static rarity-tier sort, no agent
  call) / V2 (agent-assisted live market search, mirroring F023's
  `bid_market_signal.py` pattern — explicitly not to be started before V1
  ships and is validated).
- 2026-07-20: refined — replaced the free-text-fuzzy-match matching
  strategy with a structured, optional coin-form picker over a widened
  `RomanImperialFigure` dataset (emperors, empresses, heirs/Caesars,
  usurpers, each tagged with a `role`), so non-emperor figures like Livia
  are never miscounted toward an emperor's completion and no
  alias/substring-matching code is needed. No bulk migration of existing
  coins is in scope — the app's user base is small enough that users will
  pick the figure themselves when they next edit a coin.
- 2026-07-20: linked GitHub tracking issue #501.
- 2026-07-20: committed the first curated `RomanImperialFigure` dataset pass
  (`F028-imperial-figures.md`, ~165 figures, ~96 `role: emperor`). Resolved
  the 476-cutoff open question (in scope if reign began on or before 476 —
  Zeno included in full through 491; Anastasius I excluded) and added
  Julius Caesar as a non-emperor `role: other` precursor entry, both per
  explicit request. Added a new requirement: the imperial-figure picker
  (and its backing search endpoint) must support filtering by `role`, since
  the curated list is too large to browse by name search alone.
- 2026-07-20: refined — confirmed the default/primary goal is explicitly
  the "commonly accepted Augustuses" (`role: emperor` only; usurpers were
  by definition never accepted as legitimate emperors). Added three
  independent, default-off Settings toggles (show usurpers / show
  empresses / show other figures) that each add their own separately
  tracked section to `/stats/emperors` when enabled, so users can opt into
  a bigger or more textured goal without changing what "100%" means for
  everyone else.
