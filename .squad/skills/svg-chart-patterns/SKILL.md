# Skill: SVG Chart Patterns (Vue + Dark Theme)

## Context

Reusable SVG chart patterns used in `src/web/src/components/stats/`. All charts are inline SVG in Vue SFC `<template>` blocks, styled with CSS custom properties, no external chart library.

---

## Pattern 1: Line Chart with preserveAspectRatio="none"

**File:** `src/web/src/components/stats/StatsValueOverTime.vue`

### Key Design Choices

- `viewBox="0 0 1000 320"` with `preserveAspectRatio="none"` so the chart fills its container width.
- SVG stroke elements use `vector-effect: non-scaling-stroke` to maintain consistent 1px borders regardless of viewBox scaling.
- SVG `<text>` inside a `preserveAspectRatio="none"` SVG uses CSS `font-size` (e.g. `font-size: 0.6rem`). CSS rem units are **not** affected by viewBox distortion — the text position (x, y) scales, but the rendered character size stays readable.
- For a **circled endpoint callout**, use `<circle r="30">` + `<text>` at the last data point. The circle looks elliptical at extreme viewport widths but is acceptable. Alternative: use an absolutely positioned HTML div overlay (requires no SVG padding on the container).
- Sparse per-point labels: compute `every Nth point` based on history length; exclude the last point (handled by endpoint callout).

### Layout

```
.chart-main-layout { flex-row }
  .chart-area { flex: 1 }
    .line-chart-container { flex-row: y-axis + chart }
    .line-chart-footer { legend + dates }
  .chart-side-panel { width: 10.5rem }
    .panel-roi { big ROI% }
    .chart-summary-strip { 3 .summary-pill vertical stack }
```

### Smooth path algorithm

```typescript
function toSmoothPath(points: ChartPoint[]): string {
  return points.reduce((path, point, index) => {
    if (index === 0) return `M ${point.x} ${point.y}`
    const prev = points[index - 1]!
    const mid = (prev.x + point.x) / 2
    return `${path} Q ${prev.x} ${prev.y} ${mid} ${(prev.y + point.y) / 2} T ${point.x} ${point.y}`
  }, '')
}
```

---

## Pattern 2: Alluvial / Sankey Flow Chart

**File:** `src/web/src/components/stats/StatsCoinFlowChart.vue`

### Architecture

Self-contained component: fetches all active coins via `getCoins()` pagination on mount, computes cross-tabulated flows, renders custom SVG.

### Node Layout

```typescript
function buildNodes(counts: Map<string, number>, scale: number, colorFn): SankeyNode[] {
  // sorted by count descending
  // y position = cumulative (height + NODE_GAP)
  // height = Math.max(4, count * scale)
}

const scale = CHART_H / totalCoins  // universal scale: SVG units per coin
```

Using a **single global scale** (SVG units per coin) ensures that flow band heights + node heights are proportional and consistent across all columns.

### Flow Band Algorithm

```typescript
function buildFlows(sourceNodes, targetNodes, crossTabMap, scale): SankeyBand[] {
  // Per-node offset maps track where the next band starts within each node
  const srcOffsets = new Map(sourceNodes.map(n => [n.key, 0]))
  const tgtOffsets = new Map(targetNodes.map(n => [n.key, 0]))
  // Iterate source → target pairs and stack bands
}
```

### SVG Band Path (cubic bezier, closed)

```typescript
function flowPath(band: SankeyBand, x0: number, x1: number): string {
  const sx = x0 + NODE_W  // right edge of source
  const tx = x1           // left edge of target
  const mid = (sx + tx) / 2
  const t0 = band.sourceY, b0 = band.sourceY + band.height
  const t1 = band.targetY, b1 = band.targetY + band.height
  return `M ${sx} ${t0} C ${mid} ${t0} ${mid} ${t1} ${tx} ${t1} L ${tx} ${b1} C ${mid} ${b1} ${mid} ${b0} ${sx} ${b0} Z`
}
```

### Loading State

Initialize `isLoading = ref(true)` (not `false`) so the spinner is visible before `onMounted` fires — required for correct Vitest unit test behavior with `shallowMount`.

### Column positions (3-column 660px viewBox)

```
COL_X = [80, 320, 560]  // Category, Era, Material
NODE_W = 14
SVG_W = 660
SVG_H = 360
```

---

## Design Token Usage

| Use | Token |
|---|---|
| Category colors | `--cat-roman`, `--cat-greek`, `--cat-byzantine`, `--cat-modern` |
| Era colors | `--accent-gold` (ancient), `--accent-bronze` (medieval), `--cat-modern` (modern) |
| Material colors | `--mat-gold`, `--mat-silver`, `--mat-bronze` |
| Chart background | `--bg-input` |
| Grid lines | `--border-subtle` |
| Value line | `--accent-gold` |
| Invested line | `--text-secondary` (dashed) |
| Positive/negative | `--color-positive`, `--color-negative` |

---

## Test Patterns

- `shallowMount` + `flushPromises()` pattern from `@vue/test-utils` for components that fetch on mount.
- `mockGetCoins.mockReturnValue(new Promise(() => {}))` simulates loading state (never resolves).
- For pagination, use `.mockResolvedValueOnce()` chain.
