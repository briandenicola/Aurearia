# Feature Specification: Coin Shipment Tracking via Direct Carrier APIs

**Feature Branch**: `340-coin-shipment-tracking`  
**Created**: 2026-08-03  
**Status**: Draft  
**Input**: GitHub issue #577: "F029: Track coin shipments via direct USPS/UPS/FedEx APIs"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Attach shipment details to a coin (Priority: P1)

As a collector, I want to add shipment details to a purchased coin so I can keep delivery tracking in the app.

**Why this priority**: Without coin-linked shipment records, the rest of the feature has no usable anchor.

**Independent Test**: Create a shipment from coin detail and from post-purchase/conversion flow; verify record persists and appears on the coin.

**Acceptance Scenarios**:

1. **Given** an owned coin, **When** the collector adds carrier + tracking number + notes, **Then** a shipment record is created for that coin under the authenticated user.
2. **Given** a wishlist coin that becomes purchased or a won auction lot converted into an owned coin, **When** the post-conversion flow completes, **Then** shipment data can be captured in the same workflow.
3. **Given** a shipment already exists, **When** the collector edits shipment details, **Then** changes are saved and reflected on coin detail.
4. **Given** carrier API credentials are not configured, **When** the collector creates a shipment, **Then** manual shipment tracking still works.

---

### User Story 2 - View shipment status and timeline on coin detail (Priority: P1)

As a collector, I want coin detail to show shipment status and scan history so I can see delivery progress without leaving the app.

**Why this priority**: Visibility is the core user-facing outcome.

**Independent Test**: Open a coin with shipment data and verify carrier, tracking link, normalized status, and chronological scan timeline render correctly.

**Acceptance Scenarios**:

1. **Given** a coin with a shipment, **When** coin detail loads, **Then** it shows carrier, tracking number, and a link to the carrier tracking page.
2. **Given** shipment events exist, **When** coin detail renders the timeline, **Then** events are ordered newest-to-oldest (or oldest-to-newest consistently, as specified) with timestamp and status text.
3. **Given** no scan events exist yet, **When** shipment details are shown, **Then** a clear empty timeline state is displayed.
4. **Given** manual override status is set, **When** coin detail displays shipment status, **Then** the manual status is clearly labeled as manual.

---

### User Story 3 - Auto-sync direct carrier statuses (Priority: P1)

As an admin/operator, I want USPS/UPS/FedEx integrations configured separately so shipment status updates come from each carrier directly.

**Why this priority**: Direct carrier integration is the primary architectural choice in #577.

**Independent Test**: Configure each carrier independently, run polling sync, and verify normalized status/events are persisted.

**Acceptance Scenarios**:

1. **Given** USPS/UPS/FedEx credentials are configured, **When** sync runs, **Then** the app fetches status/events from each carrier directly (no aggregator intermediary).
2. **Given** a carrier call succeeds, **When** response data is normalized, **Then** shipment current status and timeline are updated idempotently.
3. **Given** a carrier call fails or credentials are missing, **When** sync runs, **Then** the failure is recorded/observable and manual tracking remains available.
4. **Given** multiple carriers are configured, **When** polling runs, **Then** each carrier client authenticates and executes independently.

---

### User Story 4 - Notify on key shipment transitions (Priority: P1)

As a collector, I want alerts when shipment status reaches important milestones so I do not need to constantly check manually.

**Why this priority**: Transition notifications complete the tracking loop.

**Independent Test**: Simulate status transitions to out-for-delivery, delivered, and exception/return; verify in-app notification (and Pushover when enabled) is emitted once per transition.

**Acceptance Scenarios**:

1. **Given** a shipment status transitions to out-for-delivery, **When** the change is detected, **Then** an in-app notification is created for the shipment owner.
2. **Given** Pushover is enabled for the user, **When** a key transition occurs, **Then** a Pushover notification is sent.
3. **Given** a shipment status does not change, **When** polling runs repeatedly, **Then** duplicate transition notifications are not emitted.
4. **Given** a shipment transitions to delivered or exception/return, **When** the transition is persisted, **Then** notifications are emitted using the same user-scoped ownership checks.

### Edge Cases

- Tracking number is malformed for the selected carrier.
- Carrier API credentials are invalid or revoked mid-stream.
- Carrier returns partial event data (missing location or timestamp).
- Polling run overlaps with manual status edit.
- Carrier response includes duplicate/reordered events.
- Coin ownership changes while shipment exists.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow authenticated collectors to create, read, update, and delete shipment records scoped to their own coins.
- **FR-002**: Shipment record MUST include at minimum carrier, tracking number, optional notes, current status, and manual override fields.
- **FR-003**: System MUST allow shipment creation from coin detail and from post-purchase/post-auction-conversion flows.
- **FR-004**: Coin detail MUST display shipment carrier, tracking number, tracking link, current status, and chronological scan-event timeline.
- **FR-005**: System MUST integrate directly with USPS, UPS, and FedEx APIs through separate carrier clients and separate admin-configured credentials.
- **FR-006**: System MUST normalize carrier-specific responses to one internal shipment status + event schema.
- **FR-007**: Polling scheduler MUST sync shipment updates using existing scheduled-job architecture patterns in this repository.
- **FR-008**: System MUST detect key status transitions (out-for-delivery, delivered, exception/return) and emit in-app notifications.
- **FR-009**: If user Pushover notifications are enabled, system MUST also emit Pushover notifications for key shipment transitions.
- **FR-010**: System MUST keep manual status entry/override available for every carrier and for cases where credentials are absent or API sync fails.
- **FR-011**: Shipment and shipment-event queries MUST be user-scoped and MUST NOT expose data across users.
- **FR-012**: Admin carrier credentials MUST be stored in app settings and only manageable through authorized admin surfaces.
- **FR-013**: Shipment sync and transition processing MUST be idempotent and avoid duplicate notifications for unchanged state.
- **FR-014**: Failure paths (carrier API errors, auth failures, malformed responses) MUST surface explicit non-secret errors and preserve prior known shipment state.

### Key Entities

- **Shipment**: User-scoped record linked to a coin, containing carrier, tracking number, manual notes, current status, source-of-truth metadata, and timestamps.
- **ShipmentEvent**: Chronological scan/timeline event linked to a shipment, including normalized status, event timestamp, optional location/message, and raw reference metadata.
- **CarrierClient**: Internal integration contract with one implementation each for USPS, UPS, and FedEx.
- **CarrierSettings**: Admin-managed credentials/configuration entries for each carrier.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Collectors can create shipment records for owned coins from both coin detail and purchase/conversion workflows with zero cross-user access.
- **SC-002**: Coin detail shows shipment status + timeline for 100% of coins with shipment records.
- **SC-003**: USPS, UPS, and FedEx direct integrations each sync status successfully when credentials are configured.
- **SC-004**: Key transition notifications (out-for-delivery, delivered, exception/return) are emitted exactly once per transition per shipment.
- **SC-005**: Manual shipment tracking remains fully functional when any carrier integration is unavailable.

## Assumptions

- Carrier credentials are environment/admin-managed through existing app settings patterns.
- Initial implementation uses polling, not carrier webhooks.
- Shipment tracking is currently single-shipment-per-coin unless future requirements expand to multiple concurrent shipments.
- Carrier rate limits and credential quotas are handled within scheduler cadence and retry policy decisions during implementation planning.
