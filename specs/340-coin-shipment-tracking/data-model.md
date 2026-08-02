# Data Model: Coin Shipment Tracking (Direct Carrier APIs)

## Overview

The feature adds Go-owned persistence for shipment tracking without modifying
existing `Coin`/`AuctionLot` schemas. `Shipment` and `ShipmentEvent` are new
per-coin entities; `ShipmentTrackingRun` is a scheduler run-log entity following
the existing `AvailabilityRun`/`AuctionEndingRun` pattern. Per-carrier API
credentials are not a new model — they are existing `AppSetting` rows, admin-only.

## Entity: Shipment

A carrier shipment attached to a coin.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `coin_id` | uint | yes | FK to `Coin`; indexed. No unique constraint — a coin may have multiple `Shipment` rows over time (e.g. return/reship), even though v1 UI surfaces the most recent prominently |
| `auction_lot_id` | uint? | no | Optional FK to `AuctionLot`, set only when the shipment originated from a converted auction lot, for provenance |
| `carrier` | string (enum) | yes | `usps`, `ups`, `fedex`, `other` |
| `tracking_number` | string | yes | Indexed; format not validated against carrier-specific patterns since carrier is explicit, not inferred |
| `tracking_url` | string | yes | Derived carrier deep link, always computable from carrier + tracking number regardless of API configuration |
| `status` | string (enum) | yes | Shared `ShipmentStatus`: `pending`, `pre_transit`, `in_transit`, `out_for_delivery`, `delivered`, `available_for_pickup`, `return_to_sender`, `failure`, `cancelled`, `unknown`; default `pending` |
| `status_source` | string | yes | `carrier_api` or `manual` — mirrors `AuctionLotStatusSource` |
| `estimated_delivery` | time? | no | From carrier API when available |
| `shipped_at` | time? | no | |
| `delivered_at` | time? | no | Set when status transitions to `delivered` |
| `last_checked_at` | time? | no | Updated by scheduler; used to select stale shipments for the next poll cycle |
| `notes` | text | no | Collector-facing free text |
| `user_id` | uint | yes | Owner; indexed; every query scoped with existing ownership pattern |
| `created_at` / `updated_at` | time | yes | |

### Validation

- Must be authenticated and owner-scoped (owner derived from the parent `Coin`).
- `carrier` must be one of the supported enum values.
- `tracking_number` required and non-empty; no cross-carrier format inference —
  the collector always specifies carrier explicitly (spec Edge Cases).
- `tracking_url` is server-computed from `carrier` + `tracking_number`, not
  client-supplied, so it can't be spoofed to an arbitrary destination.
- Deleting a `Shipment` cascades to its `ShipmentEvent` rows.

## Entity: ShipmentEvent

A single tracking scan/status event for a shipment, as reported by a carrier API.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `shipment_id` | uint | yes | FK to `Shipment`; indexed |
| `status` | string (enum) | yes | Shared `ShipmentStatus` at the time of this event |
| `description` | string | no | Carrier-provided scan description, verbatim where possible |
| `location` | string | no | Carrier-provided location text, when present |
| `occurred_at` | time | yes | Carrier-reported event time |
| `created_at` | time | yes | Row-insert time, for ordering when `occurred_at` ties |

### Validation

- Never fabricated: every row is either sourced from a carrier API response or
  represents a collector's own manual status change (in which case `description`
  is set to reflect "manually updated" rather than left implying carrier
  provenance).
- Ordered by `occurred_at` ascending for timeline display; duplicate events (same
  shipment, same `occurred_at` + `description`) from repeated polls are not
  re-inserted.

## Entity: ShipmentTrackingRun

Scheduler run-log record, following the existing `{Feature}Run` pattern
(`AvailabilityRun`, `AuctionEndingRun`) from
`.squad/skills/adding-scheduled-job/SKILL.md`.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `trigger_type` | string | yes | `scheduled` or `manual` |
| `trigger_user_id` | uint? | no | Set only for manual admin-triggered runs |
| `status` | string | yes | `running`, `success`, `error` |
| `shipments_checked` | int | yes | Count of shipments polled this run |
| `status_changes_applied` | int | yes | Count of shipments whose status actually changed |
| `usps_failures` / `ups_failures` / `fedex_failures` | int | yes | Per-carrier failure counters so a run record attributes failures to a specific carrier (spec User Story 4, FR-016) |
| `duration_ms` | int64 | no | |
| `started_at` | time | yes | |
| `completed_at` | time? | no | |
| `error_message` | text | no | Sanitized, non-secret |
| `created_at` | time | yes | |

## Config (not a model): per-carrier credentials

Stored as existing `AppSetting` rows via `SettingsService`, following the
`PushoverService` precedent — admin-configured, DB-backed, not env vars:

- `USPSClientID`, `USPSClientSecret`, `USPSTrackingEnabled`
- `UPSClientID`, `UPSClientSecret`, `UPSTrackingEnabled`
- `FedExClientID`, `FedExClientSecret`, `FedExTrackingEnabled`
- `ShipmentTrackingCheckEnabled`, `ShipmentTrackingCheckInterval`,
  `ShipmentTrackingCheckStartTime` (scheduler cadence, per the standard
  `{Feature}CheckEnabled`/`Interval`/`StartTime` naming convention)

Each carrier's `{Carrier}TrackingEnabled` flag lets the scheduler skip a carrier
entirely (e.g. while UPS/FedEx production approval is pending) without any code
change — shipments on that carrier remain manually manageable.

## Relationships

- `Coin` 1 → many `Shipment` (v1 UI surfaces the most recent; data model allows
  more for future split-shipment/reship support)
- `AuctionLot` 0/1 → many `Shipment` (optional provenance link)
- `Shipment` 1 → many `ShipmentEvent`
- `User` 1 → many `Shipment` (via ownership)
- `ShipmentTrackingRun` has no FK to `Shipment` — it is a batch summary, not a
  per-shipment audit trail (per-shipment history lives in `ShipmentEvent` and
  `Shipment.last_checked_at`)

## State Transitions

```text
Shipment.status: pending → pre_transit → in_transit → out_for_delivery → delivered
Shipment.status: (any non-terminal) → return_to_sender | failure | cancelled
Shipment.status: (any) → unknown (carrier reports an unmapped status, or manual reset)

Shipment.status_source: manual → carrier_api (once a carrier successfully reports for the first time)
Shipment.status_source: carrier_api → manual (collector overrides)

ShipmentTrackingRun.status: running → success
ShipmentTrackingRun.status: running → error
```

Terminal statuses (`delivered`, `cancelled`, `return_to_sender`) exclude a
`Shipment` from future scheduler polling (spec User Story 2, Acceptance Scenario
4) — a collector can still manually reopen tracking (e.g. mis-marked as
delivered) by editing status, which resets `status_source` to `manual`.

## AutoMigrate

Add `&models.Shipment{}`, `&models.ShipmentEvent{}`, and
`&models.ShipmentTrackingRun{}` to the existing `AutoMigrate(...)` call in
`src/api/database/database.go`. No backfill required — this is a net-new, opt-in
table set with no relationship to existing data that needs migrating.
