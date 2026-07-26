// Hand-tuned colors for the app's long-standing default categories/eras,
// keyed by normalized name. Anything not listed here (an admin-defined
// custom category or era) falls back to a deterministic hash into
// FALLBACK_PALETTE, so every value gets a distinct, stable color without
// each component maintaining its own hardcoded per-name switch.
const KNOWN_COLORS: Record<string, string> = {
  roman: 'var(--cat-roman)',
  greek: 'var(--cat-greek)',
  byzantine: 'var(--cat-byzantine)',
  modern: 'var(--cat-modern)',
  other: 'var(--cat-other)',
  ancient: 'var(--accent-gold)',
  medieval: 'var(--accent-bronze)',
}

const FALLBACK_PALETTE = [
  'var(--cat-extra-1)',
  'var(--cat-extra-2)',
  'var(--cat-extra-3)',
  'var(--cat-extra-4)',
  'var(--cat-extra-5)',
  'var(--cat-extra-6)',
]

// Matches combining diacritical marks (U+0300-U+036F) left behind after NFD
// normalization, e.g. splitting "é" into "e" + a combining acute accent.
const COMBINING_DIACRITICS = new RegExp(
  '[' + String.fromCharCode(0x0300) + '-' + String.fromCharCode(0x036f) + ']',
  'g',
)

function normalize(label: string): string {
  return label
    .toLowerCase()
    .trim()
    .normalize('NFD')
    .replace(COMBINING_DIACRITICS, '')
    .replace(/[^a-z0-9]/g, '')
}

function hashString(value: string): number {
  let hash = 0
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0
  }
  return hash
}

/** Returns a stable CSS color (a var() reference) for a category or era name. */
export function colorForLabel(label: string | undefined | null): string {
  const key = normalize(label ?? '')
  if (!key) return 'var(--text-muted)'
  const known = KNOWN_COLORS[key]
  if (known) return known
  const index = hashString(key) % FALLBACK_PALETTE.length
  return FALLBACK_PALETTE[index]!
}

/**
 * Returns a translucent version of colorForLabel's color, for chip/badge
 * backgrounds - equivalent to Tailwind's bg-x/NN opacity modifier, but
 * works for any color value (not just pre-registered utility classes).
 */
export function colorForLabelBackground(label: string | undefined | null, alphaPercent = 20): string {
  return `color-mix(in srgb, ${colorForLabel(label)} ${alphaPercent}%, transparent)`
}
