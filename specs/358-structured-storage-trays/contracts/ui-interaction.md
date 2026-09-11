# UI Interaction Contract: Physical Storage Trays

## Navigation and scope

- Collection navigation contains both:
  - **Tray** → existing Museum Tray `/tray`
  - **Storage Trays** → physical owner-scoped `/storage-trays`
- The routes and labels are distinct; no Museum Tray preference or workflow is repurposed.

## Settings

- Create form requires a type choice.
- Standard shows name only.
- Tray shows name, rows, columns, capacity preview, and bounds.
- List rows show type. Tray rows show `rows × columns` and `occupied / capacity`.
- Occupied tray: rename enabled; dimensions disabled/explained; delete conflict explained.
- Validation identifies fields. Conflict messages use server `code`, never string matching alone.

## Coin add/edit

- No tray selected: no slot grid.
- Standard selected: slot is cleared and hidden.
- Tray selected: load occupancy and show every coordinate.
- Available cell: selectable.
- Other coin's occupied cell: disabled and visibly identified as occupied.
- Edited coin's current cell: selectable and identified as current.
- Selecting another location clears stale local slot selection before occupancy loads.
- A stale occupancy view may lose at save; `slot_occupied` keeps form data, refreshes occupancy, and asks the user to choose another slot.

## Bulk assignment

- Standard and “No location” are selectable.
- Tray is either absent from selectable choices or disabled.
- The modal displays: “Tray assignments require choosing a slot on each coin.”
- Server rejection remains authoritative if a client submits a tray ID.

## Physical grid

- Exactly `rows × columns` grid cells are rendered in row-major order.
- CSS column count equals persisted `columns` at every breakpoint.
- Empty cells remain visible and named by one-based coordinate.
- Occupied cells never move based on coin diameter, sort, filter, viewport, or image availability.
- Narrow screens contain the entire fixed grid in a labeled horizontal scrolling region. No page-level overflow.

## Accessibility

- Tray container: named grid, e.g. `aria-label="Cabinet A, 3 rows by 3 columns, 2 of 9 occupied"`.
- Empty well: non-actionable gridcell with name `Empty, row 2, column 3`.
- Occupied well: gridcell containing a button/link named `<coin name>, row 2, column 3`.
- Activation: pointer/touch, Enter, and Space.
- Occupied action target: minimum 44×44 CSS pixels.
- Focus: visible token-based outline.
- State is not conveyed by color alone; coordinate and empty/occupied cues are textual or patterned.
- Missing image preserves the occupied action and accessible name.
- Reduced motion disables nonessential transitions.

## Museum Tray regression contract

The implementation must not change:

- `/tray` route or Collection → Tray link;
- responsive 3/4/6-column packing;
- 12-coins-per-drawer pagination;
- swipe gestures or click suppression;
- face toggle;
- size-scale preference;
- inclusion/filtering rules;
- caption/placeholder adapter fields.

