export function formatCurrency(value: number, currency?: string): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency || 'USD' }).format(value)
}
