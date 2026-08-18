export function formatCurrency(value: number, currency?: string): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency || 'USD' }).format(value)
}

/**
 * Turns a coin-field vocabulary key (e.g. `dateRange`, `obverseInscription`,
 * `coin_type`) into a human-readable label (`Date Range`, `Obverse
 * Inscription`, `Coin Type`). Shared by every Deep Analysis surface that
 * renders proposed/hypothesis field names so the formatting stays identical
 * across the proposal editor and the report panel.
 */
export function formatFieldName(name: string): string {
  return name
    .replace(/_/g, ' ')
    .replace(/([A-Z])/g, ' $1')
    .trim()
    .replace(/^./, (c) => c.toUpperCase())
}
