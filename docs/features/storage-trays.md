# Storage Trays

## Configure locations

In **Settings → Data Management → Storage Locations**, choose **Standard
Location** for shelves, safes, and boxes, or **Coin Tray** for fixed physical
geometry. Trays require 1–20 rows and 1–20 columns (up to 400 slots). Occupied
trays can be renamed, but must be emptied before resizing or deletion.

## Assign an exact slot

Select a Coin Tray while adding or editing a coin. The picker lists every
one-based row and column. Occupied slots are disabled; the edited coin's own
slot remains selectable. A concurrent conflict refreshes as `slot_occupied`
without partially moving the coin. Selecting no location or a Standard
Location clears the slot. Bulk operations intentionally disable trays because
each coin needs an individual slot.

## View physical trays

Open **Collection → Storage Trays**. Every configured tray is shown, including
empty trays. Wells are materialized in row-major order from `1` through
`rows × columns`; they never pack, filter, paginate, or change column count.
On narrow screens, scroll inside the tray. Empty wells are read-only grid
cells. Occupied wells have 44×44 minimum targets and support pointer, Enter,
and Space activation. Missing images use the standard authenticated fallback.

If loading fails, use **Retry**. The separate **Tray** navigation item remains
the responsive Museum Tray and does not represent physical placement.

## Upgrade compatibility

The migration is additive and repeatable. Existing locations become Standard
Locations, existing IDs and coin assignments are retained, and
`storageSlot` remains null. A partial unique SQLite index prevents two coins
from owning the same non-null `(storageLocationId, storageSlot)` pair.
