# Research: Coin Shipment Tracking (Direct Carrier APIs)

## Decision: One `CarrierClient` interface, three independent adapters (USPS/UPS/FedEx)

**Rationale**: The requester explicitly chose direct carrier API integration over a
tracking aggregator (EasyPost/AfterShip), which was the original spike
recommendation on integration-effort grounds. To keep the rest of the system
(service, scheduler, notifications, frontend) carrier-agnostic despite three
different upstream APIs, `ShipmentService` and `ShipmentTrackingScheduler` depend
only on a small internal `CarrierClient` interface (`GetTracking(trackingNumber)
(ShipmentStatus, []ShipmentEvent, error)`), with `USPSClient`, `UPSClient`, and
`FedExClient` as separate concrete implementations, each owning its own
authentication flow and response-shape normalization. This mirrors the existing
`AuctionSource`/`numisbids_service.go`/`cng_auction_service.go` split, where
provider-specific scraping logic is isolated per source behind a shared status
model.

**Alternatives considered**:
- Tracking aggregator (EasyPost/AfterShip): one integration, unified webhook
  format, materially less work. Rejected per explicit requester decision — no
  third-party tracking intermediary.
- A single "generic HTTP carrier" client parameterized by carrier at runtime:
  rejected because USPS, UPS, and FedEx have meaningfully different auth models
  (USPS API-key/OAuth2 vs. UPS OAuth2 client-credentials vs. FedEx OAuth2
  client-credentials with different token/scopes) and response schemas; forcing
  them through one generic client would produce a leaky abstraction full of
  carrier-specific branches, working against Constitution Principle IV (Simple
  Complete Changes) more than three small, focused adapters would.

## Decision: Polling scheduler is the sole sync mechanism at MVP; no carrier webhooks

**Rationale**: A tracking aggregator would have given one normalized webhook
format for real-time updates. Direct carrier integration removes that: USPS has no
general-purpose tracking webhook product for typical developer accounts; UPS
supports tracking event webhooks but requires a separate subscription/approval
step beyond basic Tracking API access; FedEx's webhook/subscription tracking
options are similarly gated behind additional account setup. Building three
different webhook integrations (three signature schemes, three subscription
management flows, three retry/redelivery semantics) for MVP would roughly triple
Phase 3 effort for a real-time improvement that a scheduler-based poll (following
the existing `.squad/skills/adding-scheduled-job/SKILL.md` recipe already used by
`AvailabilityScheduler`/`ValuationScheduler`/`AuctionEndingScheduler`) delivers
"close enough" for a personal collection app. Per-carrier webhooks remain a
reasonable fast-follow once a given carrier's basic polling integration is proven.

**Alternatives considered**:
- Build UPS webhook support at MVP since UPS is the most webhook-capable of the
  three: rejected for v1 to keep all three carriers on one consistent, simpler
  sync mechanism (scheduler) rather than one carrier behaving differently from the
  other two; revisit as a fast-follow once UPS polling is stable.
- No automatic sync at all, manual-only tracking: rejected — automatic status is
  the feature's core value proposition per the spec's User Story 2.

## Decision: Per-carrier credentials as independent, optional `AppSetting` rows

**Rationale**: Following the existing `PushoverService`/`SettingsService` pattern
(admin-configured, DB-backed, not env vars), each carrier gets its own credential
settings (e.g. `USPSClientID`/`USPSClientSecret`, `UPSClientID`/`UPSClientSecret`,
`FedExClientID`/`FedExClientSecret`, plus an `{Carrier}TrackingEnabled` flag per
carrier). This is a single shared "the app tracks packages" capability, not a
per-user "connect your own carrier account" capability, matching the Pushover
precedent rather than the per-user encrypted-credential precedent used for
NumisBids/CNG login. Critically, since UPS and FedEx require developer-portal
approval for production access, each carrier must be independently optional: a
shipment on an unconfigured carrier must still support manual status entry and
never block scheduled checks for shipments on other, configured carriers (spec
FR-005, FR-010, FR-014).

**Alternatives considered**:
- Per-user carrier credentials (like NumisBids/CNG): rejected — collectors are not
  expected to have their own USPS/UPS/FedEx developer API access; this is
  infrastructure the app itself provides.
- A single combined "carrier credentials" JSON blob setting: rejected in favor of
  discrete per-carrier settings so one carrier's credential can be added, rotated,
  or disabled without touching the others, and so `{Carrier}TrackingEnabled` can be
  toggled independently while waiting on that carrier's approval.

## Decision: Shared `ShipmentStatus` enum; no invented data

**Rationale**: Each carrier's raw status vocabulary (USPS status descriptions, UPS
`statusType`/`statusCode`, FedEx `trackResults[].latestStatusDetail`) is mapped by
its respective `CarrierClient` into one shared enum (`pending`, `pre_transit`,
`in_transit`, `out_for_delivery`, `delivered`, `available_for_pickup`,
`return_to_sender`, `failure`, `cancelled`, `unknown`) before reaching
`ShipmentService`, the scheduler, notifications, or the frontend — mirroring how
`AuctionLotStatus` is a fixed internal enum regardless of source (NumisBids vs.
CNG). Per spec FR-012, no status, date, or event description may be shown that
wasn't actually returned by a carrier or entered by the collector; unmapped/unknown
carrier status strings fall through to `unknown` rather than being guessed at.

**Alternatives considered**:
- Store and display each carrier's raw status string directly: rejected — would
  leak carrier-specific vocabulary into notification logic and the frontend,
  requiring carrier-aware branching throughout instead of at the adapter boundary
  only.

## Decision: Shipment creation stays decoupled from `PurchaseCoin`/`ConvertToCoin`

**Rationale**: `CoinService.PurchaseCoin` and `AuctionLotService.ConvertToCoin` are
both existing, focused transactions (wishlist→purchased field flips; auction-lot→
coin creation). Adding shipment creation inside either would couple two concerns
that don't need to be atomic — a collector often doesn't have a tracking number at
the moment of purchase, and per Constitution Principle IV changes should be simple
and proportional. Instead, the frontend offers an optional "add tracking" step
immediately after either flow succeeds, using the same optional-field UX as
`PurchaseModal.vue`'s existing purchase-price/date/location fields, and the same
capability remains available any time later from the coin detail page.

**Alternatives considered**:
- Extend `PurchaseCoin`/`ConvertToCoin` to accept optional shipment fields in the
  same request/transaction: rejected — couples an optional, often-not-yet-known
  piece of data into a transaction that should stay minimal and reusable, and
  would require both existing call sites to grow shipment-aware parameters.

## Decision: Notify only on out-for-delivery, delivered, and exception/return transitions

**Rationale**: Matches spec FR-011 and User Story 3's edge case about avoiding
notification noise on routine intermediate transitions (e.g. `pre_transit` →
`in_transit`). Reuses the existing `NotificationService` "in-app row + best-effort
async Pushover push" pattern (see `NotifyAuctionPriceAlert`), fired once per
transition into a notify-worthy state rather than repeatedly on every scheduled
check that re-observes the same terminal status.

**Alternatives considered**:
- Notify on every status change: rejected as too noisy given scheduler polling
  frequency and the number of intermediate transit statuses carriers report.
