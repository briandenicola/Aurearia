---
id: F028
title: "Track collection progress toward every Roman Emperor (West + East, to 476 AD)"
status: backlog
priority: P2
effort: L
value: 4
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
      available, showing overall completion (X of Y emperors owned) plus a
      per-dynasty/per-era breakdown (e.g. "Julio-Claudian — 3 of 5",
      "Severan — 1 of 8").
- [ ] Each dynasty/era section renders its emperors using the existing tray
      visual (`MuseumTray`/`MuseumTrayWell`) — an owned emperor's well shows
      the user's real coin (image, click-through to coin detail, exactly like
      `/tray` today); an unowned emperor's well shows the tray's existing
      "no image" placeholder state, labeled with the emperor's name.
- [ ] Matching a coin to an emperor is based on the coin's free-text `Ruler`
      field against a canonical emperor reference dataset (see Notes) —
      the matching approach and its known failure modes are documented
      before this is promoted (see Open Questions).
- [ ] Western and Eastern Roman emperors are both covered, both capped at
      476 AD; nothing after that date is included in v1.
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
user-editable) — e.g. `models.RomanEmperor{ID, Name, Aliases
[]string-ish, Region (west|east), Dynasty, ReignStart, ReignEnd, SortOrder}`
— with per-user progress computed on request against the user's full active
(non-wishlist, non-sold — confirm in Open Questions) collection, the same way
`AuctionLotService.Recommend`/`MarketSignal` compute their results live
rather than maintaining a stored/cached table. Collections are small enough
(hundreds, not millions of coins) that this doesn't need a background job.

### Matching strategy (the real risk in this feature)

`Coin.Ruler` is free text (`binding:"max=200"`, no enum, no existing
canonical list anywhere in the codebase). Real collections will have
"Augustus", "Octavian", "Divus Augustus", "Constantine I", "Constantine the
Great", spelling variants, etc. A naive exact-match will under-count badly.
Proposed v1: normalize (trim/lowercase) both sides and match against each
emperor's canonical name **and** a curated alias list, allowing substring
matches. This will still miss things. Whether v1 needs a manual
match/override affordance (e.g. "this coin is actually Nerva, not matched
automatically") or whether that's a v2 addition once real usage shows how bad
the miss rate is, is an open question below — don't over-build this before
seeing it fail on real collections.

### Dynasty/era scope (illustrative, not final — the actual list is a
content-curation deliverable of this card, not something to hard-code from
this summary alone)

**Western Roman (27 BC – 476 AD)**, roughly: Julio-Claudian → Flavian →
Nerva–Antonine → Severan → Crisis of the Third Century (Maximinus Thrax
through Carinus/Numerian) → Tetrarchic/Diocletianic → Constantinian →
Valentinianic → Theodosian (West) → the fragmented last Western emperors
(Petronius Maximus through Romulus Augustulus, 455–476).

**Eastern Roman, capped at 476**: Theodosian (East: Arcadius, Theodosius II,
Marcian) → Leonid (Leo I, Leo II, Zeno's first reign, the usurper
Basiliscus). Zeno's restored reign continues past 476 and is out of scope —
document the exact cutoff rule (by emperor, not just by date) since several
reigns straddle 476.

Expect on the order of ~90 individual rulers once usurpers and joint
emperors are included — the card doesn't fix an exact count; that's decided
during data curation.

### UI

- Settings toggle: `SettingsAccountSection.vue`, same visual pattern as the
  existing `coinOfDayEnabled` checkbox.
- New page under Stats (existing sibling pages: `/stats/mint-map`,
  `/stats/timeline`, `/stats/health`, `/stats/value-trends`,
  `/stats/investment-breakdown`, `/stats/distribution` — add
  `/stats/emperors` alongside them), gated on
  `auth.user.emperorTrackerEnabled`.
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
- Principle III (Strict Types and Explicit Contracts) — canonical emperor
  dataset and match-result shape need explicit Go structs / TS types, not ad
  hoc maps.
- Principle IV (Simple Complete Changes) — do not extend
  `CoinSetTarget`/`matchCoinToTarget` for this; that system's assumptions
  (set-scoped membership, US-coin-oriented match fields) don't fit and
  bending it would make both features harder to reason about.
- Principle VI (Consistent User Experience) — reuses the existing tray visual
  language exactly rather than inventing a new "collection progress" UI
  pattern.

## Open questions

- [ ] Who curates the canonical emperor + alias dataset, and where does it
      live (seed data checked into the repo, e.g. `database/seed/`, versus a
      migration)? This is real historical research/content work, not just
      code.
- [ ] Does the match need a manual override/confirm step in v1, or is
      auto-match-only acceptable to ship first and iterate once real
      collections are tested against it? (Leaning toward v1 auto-match-only
      per Principle IV, with this flagged as the most likely v2 addition.)
- [ ] Do sold coins count toward "collected" (you owned it at some point) or
      only currently-owned, non-wishlist coins? Leaning toward
      currently-owned only, matching how the rest of the app treats
      "collection" vs "sold" vs "wishlist," but worth confirming.
- [ ] Should an empty (unmatched) emperor well be non-interactive, or should
      clicking it deep-link to "Add Coin" pre-filled with that emperor's name
      in the Ruler field? Nice-to-have, not required for v1.
- [ ] Exact 476 AD cutoff handling for Eastern emperors whose reigns straddle
      that date (notably Zeno) — include or exclude the straddling reign?
- [ ] Should co-emperors / usurpers (e.g. Lucius Verus, Basiliscus) count as
      separate tracked entries, or be folded into the primary emperor's
      entry? Affects both the total count and user expectations.

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

## History

- 2026-07-20: created (status: backlog) — feature request to track
  Western + Eastern Roman Emperor collection completeness (to 476 AD),
  grouped by dynasty/era, displayed via the existing tray UI, opt-in per
  user. Byzantine (post-476) emperors explicitly deferred to a future card.
