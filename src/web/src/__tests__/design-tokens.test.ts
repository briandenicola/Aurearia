/**
 * Design Token Enforcement Tests (Constitution Principle V)
 *
 * These tests scan Vue component scoped styles for hardcoded values
 * that should use design tokens from variables.css. Catches violations
 * at test time — zero token cost.
 */
import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'fs'
import { join, relative } from 'path'

const SRC_DIR = join(__dirname, '..')
const STYLES_DIR = join(SRC_DIR, 'assets', 'styles')

function collectVueFiles(dir: string): string[] {
  const files: string[] = []
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) {
      if (!['node_modules', '__tests__', 'dist', '.git'].includes(entry)) {
        files.push(...collectVueFiles(full))
      }
    } else if (entry.endsWith('.vue')) {
      files.push(full)
    }
  }
  return files
}

function extractScopedStyles(content: string): string[] {
  const styles: string[] = []
  const regex = /<style[^>]*scoped[^>]*>([\s\S]*?)<\/style>/g
  let match
  while ((match = regex.exec(content)) !== null) {
    styles.push(match[1])
  }
  return styles
}

describe('Design Token Enforcement (Constitution Principle V)', () => {
  const vueFiles = collectVueFiles(SRC_DIR)

  it('should find Vue files to scan', () => {
    expect(vueFiles.length).toBeGreaterThan(0)
  })

  describe('no hardcoded border-radius in scoped styles', () => {
    // border-radius must use var(--radius-*), not raw px values
    const BORDER_RADIUS_RAW = /border-radius\s*:\s*(?!\s*(?:0[;\s]|var\(|50%|inherit|initial|unset))([^;]+)/g

    // Budget: known pre-existing violations as of 2026-06-06 after syncing origin/main.
    const VIOLATION_BUDGET = 264

    it('total border-radius violations stay within budget', () => {
      let totalViolations = 0
      const details: string[] = []

      for (const file of vueFiles) {
        const content = readFileSync(file, 'utf-8')
        const styles = extractScopedStyles(content)
        if (styles.length === 0) continue

        for (const style of styles) {
          let match
          while ((match = BORDER_RADIUS_RAW.exec(style)) !== null) {
            totalViolations++
            details.push(`${relative(SRC_DIR, file)}: ${match[0].trim()}`)
          }
        }
      }

      expect(
        totalViolations,
        `Border-radius violations (${totalViolations}) exceed budget (${VIOLATION_BUDGET}). ` +
        `Use var(--radius-sm/md/lg/full) instead:\n  ${details.join('\n  ')}`
      ).toBeLessThanOrEqual(VIOLATION_BUDGET)
    })
  })

  describe('no hardcoded hex colors in scoped styles', () => {
    const HEX_COLOR = /#(?:[0-9a-fA-F]{3,4}){1,2}(?![0-9a-fA-F])/g
    const ALLOWED_HEX = new Set(['#000', '#000000', '#fff', '#ffffff'])

    // Budget: known pre-existing violations as of 2026-06-06 after syncing origin/main.
    // This number must only decrease over time.
    const VIOLATION_BUDGET = 190

    it('total hex color violations stay within budget', () => {
      let totalViolations = 0
      const details: string[] = []

      for (const file of vueFiles) {
        const content = readFileSync(file, 'utf-8')
        const styles = extractScopedStyles(content)
        if (styles.length === 0) continue

        for (const style of styles) {
          const cleaned = style.replace(/\/\*[\s\S]*?\*\//g, '')
          const noVarFallback = cleaned.replace(/var\([^)]*\)/g, '')
          let match
          while ((match = HEX_COLOR.exec(noVarFallback)) !== null) {
            if (!ALLOWED_HEX.has(match[0].toLowerCase())) {
              totalViolations++
              details.push(`${relative(SRC_DIR, file)}: ${match[0]}`)
            }
          }
        }
      }

      expect(
        totalViolations,
        `Hex color violations (${totalViolations}) exceed budget (${VIOLATION_BUDGET}). ` +
        `Use CSS variables from variables.css instead:\n  ${details.join('\n  ')}`
      ).toBeLessThanOrEqual(VIOLATION_BUDGET)
    })
  })

  describe('no hardcoded font-size outside typography scale in scoped styles', () => {
    const ALLOWED_SIZES = new Set([
      '2rem', '1.5rem', '1.2rem', '1rem', '0.9rem', '0.85rem', '0.8rem', '0.75rem', '0.7rem',
    ])
    const FONT_SIZE = /font-size\s*:\s*(?!var\(|inherit|initial|unset|smaller|larger)([^;]+)/g

    // Budget: known pre-existing violations as of 2026-06-06 after syncing origin/main.
    // This number must only decrease over time. If a refactor reduces it, lower the cap.
    const VIOLATION_BUDGET = 126

    it('total font-size violations stay within budget', () => {
      let totalViolations = 0
      const details: string[] = []

      for (const file of vueFiles) {
        const content = readFileSync(file, 'utf-8')
        const styles = extractScopedStyles(content)
        if (styles.length === 0) continue

        for (const style of styles) {
          const cleaned = style.replace(/\/\*[\s\S]*?\*\//g, '')
          let match
          while ((match = FONT_SIZE.exec(cleaned)) !== null) {
            const value = match[1].trim()
            if (!ALLOWED_SIZES.has(value) && !value.startsWith('var(')) {
              totalViolations++
              details.push(`${relative(SRC_DIR, file)}: font-size: ${value}`)
            }
          }
        }
      }

      expect(
        totalViolations,
        `Font-size violations (${totalViolations}) exceed budget (${VIOLATION_BUDGET}). ` +
        `Fix violations or lower budget if you reduced them:\n  ${details.join('\n  ')}`
      ).toBeLessThanOrEqual(VIOLATION_BUDGET)
    })
  })

  describe('variables.css and main.css define required tokens', () => {
    it('variables.css defines core radius tokens', () => {
      const vars = readFileSync(join(STYLES_DIR, 'variables.css'), 'utf-8')
      expect(vars).toContain('--radius-sm')
      expect(vars).toContain('--radius-md')
      expect(vars).toContain('--radius-lg')
      expect(vars).toContain('--radius-full')
    })

    it('variables.css defines core color tokens', () => {
      const vars = readFileSync(join(STYLES_DIR, 'variables.css'), 'utf-8')
      expect(vars).toContain('--accent-gold')
      expect(vars).toContain('--bg-card')
      expect(vars).toContain('--border-subtle')
      expect(vars).toContain('--text-primary')
    })

    it('main.css defines global chip and button classes', () => {
      const main = readFileSync(join(STYLES_DIR, 'main.css'), 'utf-8')
      expect(main).toContain('.chip')
      expect(main).toContain('.btn')
      expect(main).toContain('.btn-primary')
    })
  })
})
