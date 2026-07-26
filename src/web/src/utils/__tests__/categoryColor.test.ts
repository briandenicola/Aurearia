import { describe, expect, it } from 'vitest'
import { colorForLabel, colorForLabelBackground } from '../categoryColor'

describe('colorForLabel', () => {
  it('returns the hand-tuned color for known default categories/eras', () => {
    expect(colorForLabel('Roman')).toBe('var(--cat-roman)')
    expect(colorForLabel('roman')).toBe('var(--cat-roman)')
    expect(colorForLabel('Byzantine')).toBe('var(--cat-byzantine)')
    expect(colorForLabel('ancient')).toBe('var(--accent-gold)')
    expect(colorForLabel('medieval')).toBe('var(--accent-bronze)')
  })

  it('returns a stable, deterministic color for a custom value not in the known list', () => {
    const first = colorForLabel('Celtic')
    const second = colorForLabel('Celtic')
    expect(first).toBe(second)
    expect(first).toMatch(/^var\(--cat-extra-\d\)$/)
  })

  it('gives different custom values distinct colors when possible', () => {
    const celtic = colorForLabel('Celtic')
    const sassanian = colorForLabel('Sassanian')
    expect(celtic).not.toBe(sassanian)
  })

  it('is insensitive to case, whitespace, and punctuation', () => {
    expect(colorForLabel('  ROMAN  ')).toBe(colorForLabel('roman'))
    expect(colorForLabel('Byzantine!')).toBe(colorForLabel('byzantine'))
  })

  it('falls back to a muted color for blank input', () => {
    expect(colorForLabel('')).toBe('var(--text-muted)')
    expect(colorForLabel(undefined)).toBe('var(--text-muted)')
    expect(colorForLabel(null)).toBe('var(--text-muted)')
  })
})

describe('colorForLabelBackground', () => {
  it('wraps colorForLabel in a color-mix with the given alpha', () => {
    expect(colorForLabelBackground('Roman')).toBe('color-mix(in srgb, var(--cat-roman) 20%, transparent)')
    expect(colorForLabelBackground('Roman', 30)).toBe('color-mix(in srgb, var(--cat-roman) 30%, transparent)')
  })
})
