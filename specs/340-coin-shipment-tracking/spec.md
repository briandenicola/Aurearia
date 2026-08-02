# Feature Specification: Coin Shipment Tracking (Direct Carrier APIs)

**Feature Branch**: `claude/coin-shipment-tracking-n24fhx`
**Created**: 2026-08-02
**Status**: Draft
**Input**: GitHub issue #577: "F029: Track coin shipments via direct USPS/UPS/FedEx APIs"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add tracking to a purchased coin (Priority: P1)

As a collector, I want to attach a carrier and tracking number to a coin so I can
see where my shipment is without leaving the app.

**Why this priority**: This is the core, independently valuable slice — a collector
can already get value from centralized tracking-number storage and a carrier
deep-link even before any live carrier API integration exists.

**Independent Test**: On an owned (non-wishlist) coin, add a shipment with carrier
and tracking number; verify it appears on the coin detail page with a working link
to the carrier's public tracking page, and can be edited or removed.

**Acceptance Scenarios**:

1. **Given** an owned coin with no shipment, **When** the collector adds a carrier
   and tracking number from the coin detail page, **Then** a shipment record is
   created and shown with a link to that carrier's public tracking page.
2. **Given** a coin that was just purchased from the wishlist, or an auction lot
   that was just converted to a coin, **When** the purchase/conversion completes,
   **Then** the collector is offered an optional, skippable "add tracking" step
   using the same optional-field convention as `PurchaseModal.vue`.
3. **Given** an existing shipment, **When** the collector edits the tracking number,
   carrier, or notes, **Then** the changes are saved and reflected immediately.
4. **Given** an existing shipment, **When** the collector deletes it, **Then** it no
   longer appears on the coin and no further status checks occur for it.

---

### User Story 2 - Automatic status updates from USPS, UPS, and FedEx (Priority: P1)

As a collector, I want shipment status to update automatically from the carrier so I
don't have to manually check USPS.com, UPS.com, or FedEx.com.

**Why this priority**: Automatic status is the feature's core value over just
storing a tracking number; it requires direct integration with each carrier's own
API since no aggregator is used.

**Independent Test**: With carrier credentials configured for at least one carrier,
create a shipment with a real/sandbox tracking number for that carrier; verify the
shipment status and event timeline update to match the carrier's reported status
within one polling cycle.

**Acceptance Scenarios**:

1. **Given** a shipment with a carrier whose API credentials are configured,
   **When** the scheduled tracking check runs, **Then** the system calls that
   carrier's tracking API directly, updates the shipment's status, and appends any
   new scan events to the timeline.
2. **Given** a shipment with a carrier whose API credentials are not yet configured
   (e.g. still awaiting production approval), **When** the scheduled tracking check
   runs, **Then** that shipment is skipped for automatic updates and the collector
   can still set/update its status manually.
3. **Given** a carrier API call fails, times out, or is rate-limited, **When** the
   scheduled check processes that shipment, **Then** the failure is logged in the
   run record without blocking checks for other shipments/carriers, and the
   shipment's last-known status is preserved.
4. **Given** a shipment reaches a terminal status (delivered, cancelled, return to
   sender), **When** future scheduled checks run, **Then** that shipment is no
   longer polled.

---

### User Story 3 - Get notified on meaningful status changes (Priority: P2)

As a collector, I want to be notified when a shipment is out for delivery, has been
delivered, or has an exception, so I don't have to keep checking the app.

**Why this priority**: Notifications turn passive tracking data into an actionable
signal, reusing the existing in-app + Pushover notification pattern.

**Independent Test**: Force a shipment's status to transition to
`out_for_delivery`, `delivered`, and `return_to_sender` in turn; verify an in-app
notification is created for each transition, and a Pushover push is sent if the
collector has Pushover enabled.

**Acceptance Scenarios**:

1. **Given** a shipment status changes to `out_for_delivery`, **When** the change is
   detected (via scheduled check), **Then** the collector receives an in-app
   notification and, if enabled, a Pushover push.
2. **Given** a shipment status changes to `delivered`, **When** the change is
   detected, **Then** the collector is notified once (not repeatedly on subsequent
   checks of the same terminal status).
3. **Given** a shipment status changes to `return_to_sender` or a carrier-reported
   exception, **When** the change is detected, **Then** the collector is notified
   with the exception reason if the carrier provides one.
4. **Given** a status change that is not one of the notify-worthy transitions
   (e.g. `pre_transit` → `in_transit`), **When** it is detected, **Then** the
   timeline updates but no notification is sent, to avoid noise.

---

### User Story 4 - Admin visibility into carrier sync health (Priority: P3)

As the app admin, I want to see when tracking checks last ran, how many shipments
were checked, and which carrier calls failed, so I can diagnose credential or
API issues without digging through logs.

**Why this priority**: With three independent carrier integrations instead of one
aggregator, sync failures are more likely to be carrier-specific (expired token,
carrier outage, revoked API key); an admin-visible run history is needed to
diagnose which carrier is failing without server log access.

**Independent Test**: Trigger a manual tracking-check run from the admin surface;
verify a run record is created showing shipments checked, per-carrier error counts,
and duration, matching the existing scheduler run-log pattern used elsewhere in the
app.

**Acceptance Scenarios**:

1. **Given** the admin triggers a manual tracking check, **When** it completes,
   **Then** a run record shows trigger type, shipments checked, status changes
   applied, per-carrier failure counts, and duration.
2. **Given** a carrier's credentials are invalid or expired, **When** the scheduled
   check runs, **Then** the run record clearly attributes the failure to that
   carrier rather than surfacing a generic error.
3. **Given** run history exists, **When** the admin views it, **Then** the most
   recent runs are listed with enough detail to decide whether a carrier
   integration needs attention.

### Edge Cases

- A carrier's production API access is still pending approval (common for UPS and
  FedEx, which require developer-portal review) — the system must operate in
  manual-only mode for that carrier without blocking the rest of the feature.
- A tracking number format is ambiguous or could plausibly belong to more than one
  carrier — the collector must explicitly select the carrier; the system must not
  guess.
- A carrier API returns a "not found" or "invalid tracking number" response —
  surfaced to the collector as a clear, non-alarming state, not treated as a
  transport error.
- A carrier's OAuth/API token expires or is revoked mid-operation — automatic
  refresh where the carrier's flow supports it; otherwise the run record flags it
  for admin action and manual status entry remains available.
- A coin has multiple shipments over time (e.g. an item was returned and reshipped)
  — the data model supports multiple `Shipment` rows per coin even if MVP UI shows
  the most recent prominently.
- A coin or shipment is deleted while a tracking check is in flight for it — the
  check must not fail the whole run or resurrect a deleted record.
- Carrier API rate limits are hit during a scheduled run — remaining shipments for
  that carrier are deferred to the next cycle rather than the whole run failing.
- Shipment/tracking data must remain scoped to the owning collector like all other
  coin data.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow an authenticated collector to create, view, edit,
  and delete a shipment (carrier, tracking number, notes) attached to a coin they
  own.
- **FR-002**: System MUST support carrier values USPS, UPS, and FedEx at minimum,
  with an "Other/Unknown" option that supports manual tracking only.
- **FR-003**: System MUST generate a working public tracking-page deep link from
  carrier + tracking number, independent of whether that carrier's API is
  configured.
- **FR-004**: System MUST integrate directly with each carrier's own tracking API
  (USPS, UPS, FedEx) — no third-party tracking-aggregator service — behind a common
  internal interface so the rest of the system is carrier-agnostic.
- **FR-005**: Each carrier's API credentials (OAuth client ID/secret or API key)
  MUST be configured independently by an admin, MUST be storable per carrier, and
  MUST support the carrier being left unconfigured without breaking the rest of the
  feature.
- **FR-006**: System MUST periodically check for status updates on shipments that
  have a configured carrier and have not reached a terminal status
  (delivered/cancelled/return_to_sender), via a scheduled background job.
- **FR-007**: System MUST record each scheduled or manually triggered tracking-check
  run, including trigger type, shipments checked, status changes applied, and
  per-carrier failure counts.
- **FR-008**: System MUST normalize each carrier's carrier-specific status
  vocabulary into one shared `ShipmentStatus` set (e.g. pending, pre_transit,
  in_transit, out_for_delivery, delivered, available_for_pickup,
  return_to_sender, failure, cancelled, unknown) so the UI and notification logic
  do not branch on carrier.
- **FR-009**: System MUST preserve a chronological event history (scan
  description, location, timestamp) per shipment as returned by the carrier, where
  the carrier's API provides it.
- **FR-010**: System MUST allow manual status entry/override regardless of whether
  a carrier's API is configured, so tracking remains usable during carrier API
  approval delays or outages.
- **FR-011**: System MUST send an in-app notification, and a Pushover push if the
  collector has Pushover enabled, when a shipment transitions to
  out-for-delivery, delivered, or an exception/return-to-sender state — and MUST
  NOT notify on every intermediate status change.
- **FR-012**: System MUST NOT invent or guess shipment status, delivery dates, or
  event details not actually returned by the carrier's API or entered by the
  collector.
- **FR-013**: System MUST scope all shipment data (records, event history, run
  visibility) to the owning collector; shipment CRUD endpoints MUST enforce
  coin ownership.
- **FR-014**: System MUST handle carrier API failures (auth failure, rate limit,
  timeout, not-found) per shipment without one carrier's failure blocking checks
  for other carriers or other shipments in the same run.
- **FR-015**: System MUST allow the collector to optionally add a shipment
  immediately after a wishlist-to-purchase transition or an auction-lot-to-coin
  conversion, without requiring it (all shipment fields remain optional at that
  point, matching existing `PurchaseModal.vue` UX).
- **FR-016**: System MUST provide an admin-facing way to view tracking-check run
  history and manually trigger a run, following the existing scheduler run-log
  pattern used elsewhere in the app.

### Key Entities

- **Shipment**: A carrier shipment attached to a coin — carrier, tracking number,
  tracking URL, current status, status source (carrier-API vs. manual), estimated
  delivery, shipped/delivered timestamps, last-checked timestamp, notes, owning
  user, and the coin it belongs to.
- **ShipmentEvent**: A single tracking scan/status event for a shipment — status,
  description, location, and when it occurred, ordered chronologically.
- **CarrierCredential** (config, not user data): Per-carrier admin-configured API
  credentials (USPS, UPS, FedEx each independently), including whether that
  carrier is currently enabled for automatic checks.
- **ShipmentTrackingRun**: A record of one scheduled or manual tracking-check
  cycle — trigger type, shipments checked, status changes applied, per-carrier
  failure counts, duration, timestamps.

## Constitution-Aligned Constraints

- **Clear layered ownership**: A single `CarrierClient` interface with three
  concrete USPS/UPS/FedEx implementations lives in the service layer; handlers stay
  thin CRUD wrappers; shipment creation is not coupled into the `PurchaseCoin` /
  `ConvertToCoin` transactions.
- **Security and privacy by default**: Carrier API credentials are admin-owned
  settings, never exposed to non-admin users or in API responses; shipment data is
  authenticated and owner-scoped like all other coin data.
- **Consistent UX**: Shipment UI follows the existing coin-detail section-page
  pattern (alongside Journal/Notes/Valuation) and the existing optional-field
  convention from `PurchaseModal.vue`.
- **Simple, complete, proportional scope**: v1 covers manual entry, direct
  USPS/UPS/FedEx polling, status timeline, and notifications on meaningful
  transitions. Carrier-native webhooks, multi-shipment-per-coin UI, and
  return/reship workflows are deferred past v1 unless direct-API research during
  planning shows they're low-risk to include.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector can add a shipment to a coin and see a working carrier
  tracking link in under 30 seconds, with zero required fields beyond carrier and
  tracking number.
- **SC-002**: For any carrier with configured credentials, a shipment's status
  reflects the carrier's actual reported status within one scheduled polling cycle.
- **SC-003**: 100% of notify-worthy status transitions (out-for-delivery,
  delivered, exception/return) generate exactly one in-app notification each,
  with a Pushover push when enabled — no duplicate notifications for repeated
  observations of the same status.
- **SC-004**: 0 shipment status values, dates, or event descriptions are shown that
  were not actually returned by a carrier API or entered by the collector.
- **SC-005**: A carrier whose credentials are not yet configured never blocks
  manual tracking usage for shipments on that carrier, and never blocks scheduled
  checks for shipments on other, configured carriers.
- **SC-006**: Owner-scoping tests verify collectors cannot view, edit, or delete
  shipments on coins they do not own.
- **SC-007**: Admin run history shows, for each run, per-carrier success/failure
  counts sufficient to identify which single carrier integration (if any) needs
  attention, without server log access.

## Assumptions

- The app owner (Brian) will register developer accounts directly with USPS, UPS,
  and FedEx; UPS and FedEx production API access requires their own
  developer-portal approval, which is outside this feature's engineering timeline
  and should be started as early as possible.
- Direct carrier APIs are polled on a scheduled interval (following the existing
  scheduler pattern); carrier-native webhooks are out of scope for v1 given the
  differing setup/verification requirements per carrier, and may be added later per
  carrier if it proves low-risk.
- A shared `AppSetting`-based admin credential store (one entry set per carrier) is
  sufficient; this is not a per-user "connect your own carrier account" feature.
- One shipment per coin is sufficient for the v1 UI even though the data model
  supports multiple; split shipments/returns/reships are a fast-follow if needed.
- This feature is additive and does not require changes to existing
  `PurchaseCoin`/`ConvertToCoin` transactional behavior — shipment attachment is
  always a separate, optional step.
