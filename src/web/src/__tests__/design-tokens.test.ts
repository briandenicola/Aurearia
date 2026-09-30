/**
 * Design Token Enforcement Tests (Constitution Principle VI)
 *
 * These tests scan Vue component scoped styles for hardcoded values
 * that should use design tokens from variables.css. Catches violations
 * at test time — zero token cost.
 */
import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'fs'
import { join, relative, sep } from 'path'

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

describe('Design Token Enforcement (Constitution Principle VI)', () => {
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

  describe('font families use the shared typography tokens', () => {
    const FONT_FAMILY = /font-family\s*:\s*([^;]+)/g

    it('does not use raw font-family declarations in application styles', () => {
      const violations: string[] = []
      const styleFiles = [
        join(STYLES_DIR, 'main.css'),
        ...vueFiles,
      ]

      for (const file of styleFiles) {
        const content = readFileSync(file, 'utf-8')
        let match
        while ((match = FONT_FAMILY.exec(content)) !== null) {
          const value = match[1].trim()
          if (value !== 'inherit' && !value.startsWith('var(--font-family-')) {
            violations.push(`${relative(SRC_DIR, file)}: font-family: ${value}`)
          }
        }
      }

      expect(
        violations,
        `Use --font-family-sans or --font-family-display:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('does not bypass font tokens with arbitrary Tailwind families', () => {
      const violations = vueFiles
        .filter(file => /font-\[[^\]]+\]/.test(readFileSync(file, 'utf-8')))
        .map(file => relative(SRC_DIR, file))

      expect(
        violations,
        `Use font-sans or font-display instead:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('bundles Inter and Cinzel instead of loading remote Google Fonts', () => {
      const mainCss = readFileSync(join(STYLES_DIR, 'main.css'), 'utf-8')
      const mainTs = readFileSync(join(SRC_DIR, 'main.ts'), 'utf-8')

      expect(mainCss).not.toContain('fonts.googleapis.com')
      expect(mainTs).toContain("@fontsource/inter/400.css")
      expect(mainTs).toContain("@fontsource/inter/600.css")
      expect(mainTs).toContain("@fontsource/cinzel/500.css")
      expect(mainTs).toContain("@fontsource/cinzel/600.css")
    })
  })

  describe('admin schedule typography', () => {
    const scheduleDir = join(SRC_DIR, 'components', 'admin', 'schedules')
    const scheduleFiles = [
      join(SRC_DIR, 'components', 'admin', 'AdminSchedulesSection.vue'),
      ...collectVueFiles(scheduleDir),
    ]

    it('uses the standard h3 display hierarchy', () => {
      const violations: string[] = []

      for (const file of scheduleFiles) {
        const content = readFileSync(file, 'utf-8')
        const headings = content.match(/<h3\b[^>]*>/g) ?? []
        for (const heading of headings) {
          if (
            !heading.includes('text-lg') ||
            !heading.includes('font-medium') ||
            !heading.includes('text-heading')
          ) {
            violations.push(`${relative(SRC_DIR, file)}: ${heading}`)
          }
        }
      }

      expect(
        violations,
        `Schedule h3 elements must use text-lg font-medium text-heading:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })
  })

  describe('shared UI primitives replace hand-rolled controls', () => {
    const toggleComponent = join(SRC_DIR, 'components', 'ui', 'BaseToggle.vue')
    const statusBadgeComponent = join(SRC_DIR, 'components', 'ui', 'BaseStatusBadge.vue')

    it('routes every switch through BaseToggle', () => {
      // Any visually hidden checkbox is a hand-rolled switch, however the
      // class list is spelled or ordered.
      const hiddenCheckbox = /<input\b[^>]*>/g
      const violations = vueFiles
        .filter((file) => file !== toggleComponent)
        .filter((file) => {
          const content = readFileSync(file, 'utf-8')
          return [...content.matchAll(hiddenCheckbox)].some(
            (tag) => tag[0].includes('sr-only') && /type="checkbox"/.test(tag[0])
          )
        })
        .map((file) => relative(SRC_DIR, file))

      expect(
        violations,
        `Hand-rolled toggles must use BaseToggle:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('routes every run-status pill through BaseStatusBadge', () => {
      // A pill whose colour is chosen from a status expression must use the
      // primitive, regardless of how its utility classes are ordered.
      const violations = vueFiles
        .filter((file) => file !== statusBadgeComponent)
        .filter((file) => {
          const content = readFileSync(file, 'utf-8')
          return [...content.matchAll(/<span\b[^>]*>/g)].some(
            (tag) =>
              tag[0].includes('rounded-full') &&
              /\bstatus\b/i.test(tag[0]) &&
              /rgba\(|#[0-9a-fA-F]{3,8}\b/.test(tag[0])
          )
        })
        .map((file) => relative(SRC_DIR, file))

      expect(
        violations,
        `Status pills must use BaseStatusBadge:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('keeps hardcoded colour literals in templates within budget', () => {
      // Budget: remaining pre-existing hardcoded template colour literals
      // after the #784 consolidation. Counts rgba(), rgb() and hex literals so
      // the ratchet cannot be evaded by switching notation.
      //
      // The budget is per file, not a single net total. A net total lets a
      // literal removed in one file pay for a literal added in another, which
      // is exactly the drift this is meant to stop. A file may only go down,
      // and a file not listed here may have none at all.
      const COLOR_BUDGET = new Map([
        ['components/auction/AuctionLotDetailModal.vue', 16],
        ['components/stats/CollectionHealthScorecard.vue', 12],
        ['components/admin/AdminAISection.vue', 6],
        ['components/collection/NeedsAttentionQueue.vue', 6],
        ['pages/NotificationsPage.vue', 3],
        ['components/AuctionLotCard.vue', 2],
        ['components/admin/AdminHealthSection.vue', 2],
        ['components/stats/CollectionHealthTrendIndicator.vue', 2],
        ['components/wishlist-alerts/AlertCriteriaSummary.vue', 2],
        ['pages/CoinLookupPage.vue', 2],
        ['pages/WishlistPage.vue', 2],
        ['components/HelpSection.vue', 1],
        ['components/admin/schedules/AdminValuationSchedule.vue', 1],
        ['pages/FollowerCoinDetailPage.vue', 1],
        ['pages/NotesPage.vue', 1],
        ['pages/PublicShowcasePage.vue', 1],
        ['pages/SetDetailPage.vue', 1],
      ])
      const violations: string[] = []

      for (const file of vueFiles) {
        const template = /<template>([\s\S]*)<\/template>/.exec(readFileSync(file, 'utf-8'))?.[1]
        if (!template) continue
        const name = relative(SRC_DIR, file).split(sep).join('/')
        const count = (template.match(/rgba?\(|#[0-9a-fA-F]{3,8}\b/g) ?? []).length
        const allowed = COLOR_BUDGET.get(name) ?? 0
        if (count > allowed) {
          violations.push(`${name}: ${count} literals, budget ${allowed} — use a design token`)
        }
      }

      expect(
        violations,
        `Hardcoded template colour literals exceed their per-file budget:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('uses no arbitrary absolute text sizes in templates', () => {
      // Every font size must be a step on the scale in main.css. `em` values
      // are allowed: they size rendered markdown relative to its container,
      // which is a ratio rather than a scale step.
      const violations: string[] = []

      for (const file of vueFiles) {
        const template = /<template>([\s\S]*)<\/template>/.exec(readFileSync(file, 'utf-8'))?.[1]
        if (!template) continue
        for (const match of template.matchAll(/text-\[[0-9.]+(rem|px|pt)\]/g)) {
          violations.push(`${relative(SRC_DIR, file)}: ${match[0]}`)
        }
      }

      expect(
        violations,
        `Arbitrary font sizes bypass the typography scale. Use a text-* step from main.css:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('uses theme colour utilities rather than raw var() text classes', () => {
      // `text-[var(--x)]` and a `text-x` utility render identically, but only
      // the utility is auditable and only it fails when the token disappears.
      const violations: string[] = []

      for (const file of vueFiles) {
        const template = /<template>([\s\S]*)<\/template>/.exec(readFileSync(file, 'utf-8'))?.[1]
        if (!template) continue
        for (const match of template.matchAll(/text-\[var\(--[a-z0-9-]+\)\]/g)) {
          violations.push(`${relative(SRC_DIR, file)}: ${match[0]}`)
        }
      }

      expect(
        violations,
        `Use the theme colour utility instead of a raw var() text class:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('styles every uppercase label with the one documented recipe', () => {
      // An uppercase element carrying a small size token is a label. All of
      // them use text-label at 600 weight with 0.08em tracking; only the
      // colour and the alignment vary by role. Elements without a small size
      // token are out of scope: they are badges, data cells or inputs that
      // happen to be uppercased, not section labels.
      const smallSizes = ['text-label', 'text-sm', 'text-xs', 'text-chip', 'text-micro', 'text-2xs']
      const violations: string[] = []

      for (const file of vueFiles) {
        const template = /<template>([\s\S]*)<\/template>/.exec(readFileSync(file, 'utf-8'))?.[1]
        if (!template) continue
        for (const match of template.matchAll(/class="([^"]*\buppercase\b[^"]*)"/g)) {
          const tokens = match[1].split(/\s+/)
          if (!tokens.some((token) => smallSizes.includes(token))) continue

          const wrong: string[] = []
          const size = tokens.find((token) => smallSizes.includes(token))
          if (size !== 'text-label') wrong.push(`${size} (want text-label)`)
          if (!tokens.includes('tracking-[0.08em]')) wrong.push('missing tracking-[0.08em]')
          if (!tokens.includes('font-semibold')) wrong.push('missing font-semibold')
          if (wrong.length > 0) {
            violations.push(`${relative(SRC_DIR, file)}: ${wrong.join(', ')}`)
          }
        }
      }

      expect(
        violations,
        `Uppercase labels must read "text-label font-semibold tracking-[0.08em] uppercase":\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })

    it('styles table headers through .data-table rather than inline recipes', () => {
      // The header recipe had drifted into five near-identical variants across
      // 27 tables. It now lives in one place, in main.css. Padding stays on
      // the element because density is a per-table decision.
      //
      // Two spellings have to be caught, because the recipe was inlined both
      // ways. The `[&_th]:` variant on the table, and — the one that slipped
      // past the first version of this guard — the same declarations repeated
      // on every individual `th`. The second form is why this also checks
      // that every `<table>` carries `data-table` or sits inside a container
      // that does.
      //
      // The containment check is file-scoped, not per-table: it asks whether
      // the template mentions `data-table` at all. A file holding one migrated
      // and one unmigrated table would slip through. That is the price of not
      // parsing the template, and HelpSection.vue needs it — its tables are
      // rendered from markdown and can only be reached through a wrapper.
      const violations: string[] = []

      for (const file of vueFiles) {
        const template = /<template>([\s\S]*)<\/template>/.exec(readFileSync(file, 'utf-8'))?.[1]
        if (!template) continue
        const name = relative(SRC_DIR, file)

        for (const match of template.matchAll(
          /\[&_th\]:(uppercase|tracking-\[[^\]]+\]|text-text-muted|font-semibold)/g
        )) {
          violations.push(`${name}: ${match[0]} — apply "data-table" instead`)
        }

        const scoped = /\bdata-table\b/.test(template)
        for (const match of template.matchAll(/<table\b[^>]*>/g)) {
          if (scoped) continue
          violations.push(`${name}: ${match[0].slice(0, 60)} — table is not inside a "data-table"`)
        }

        for (const match of template.matchAll(/<th\b[^>]*class="([^"]*)"/g)) {
          const tokens = match[1].split(/\s+/)
          const inlined = ['uppercase', 'text-text-muted', 'font-semibold', 'text-label'].filter((token) =>
            tokens.includes(token)
          )
          if (inlined.length >= 2) {
            violations.push(`${name}: <th> repeats the header recipe (${inlined.join(' ')})`)
          }
        }
      }

      expect(
        violations,
        `Table headers come from the .data-table class in main.css:\n  ${violations.join('\n  ')}`
      ).toEqual([])
    })
  })

  describe('template classes resolve to real styles', () => {
    const mainCss = readFileSync(join(STYLES_DIR, 'main.css'), 'utf-8')
    const themeColors = new Set([...mainCss.matchAll(/--color-([a-z0-9-]+)\s*:/g)].map((m) => m[1]))
    const themeSizes = new Set([...mainCss.matchAll(/--text-([a-z0-9]+)\s*:/g)].map((m) => m[1]))
    const tailwindSizes = ['xs', '3xl', '4xl', '5xl', '6xl', '7xl', '8xl', '9xl']
    const textUtilities = ['left', 'center', 'right', 'justify', 'start', 'end', 'wrap', 'nowrap', 'balance', 'pretty', 'ellipsis', 'clip']
    const colorKeywords = ['white', 'black', 'transparent', 'current', 'inherit']
    const bgUtilities = ['none', 'cover', 'contain', 'auto', 'center', 'top', 'bottom', 'fixed', 'local', 'scroll', 'repeat', 'no-repeat', 'clip', 'origin', 'blend']
    const semanticPrefix = /^(form|section|info|setting|settings|btn)-/
    const globalSelectors = new Set([...mainCss.matchAll(/\.([a-zA-Z][a-zA-Z0-9_-]*)/g)].map((m) => m[1]))

    function templateClassTokens(content: string): string[] {
      const template = content.match(/<template>[\s\S]*<\/template>/)?.[0] ?? ''
      const tokens: string[] = []
      for (const m of template.matchAll(/\sclass="([^"]*)"/g)) tokens.push(...m[1].split(/\s+/))
      for (const m of template.matchAll(/\s:class="([^"]*)"/g)) {
        for (const lit of m[1].matchAll(/'([^']*)'/g)) tokens.push(...lit[1].split(/\s+/))
      }
      return tokens.filter(Boolean)
    }

    function isKnownColor(name: string): boolean {
      return themeColors.has(name) || colorKeywords.includes(name) || /^[a-z]+-\d{2,3}$/.test(name)
    }

    function problem(token: string, scoped: Set<string>): string | null {
      const opens = (token.match(/\[/g) ?? []).length
      const closes = (token.match(/\]/g) ?? []).length
      if (opens !== closes) return 'unbalanced brackets'

      const utility = token.replace(/^!/, '').split(':').pop()!.replace(/\/\d+$/, '')
      if (utility.includes('[')) return null

      const text = utility.match(/^text-(.+)$/)
      if (text && !themeSizes.has(text[1]) && !tailwindSizes.includes(text[1]) && !textUtilities.includes(text[1]) && !isKnownColor(text[1])) {
        return 'text-* is not a typography token or theme color'
      }
      const bg = utility.match(/^bg-(.+)$/)
      if (bg && !isKnownColor(bg[1]) && !bgUtilities.includes(bg[1]) && !bg[1].startsWith('gradient') && !bg[1].startsWith('linear')) {
        return 'bg-* is not a theme color'
      }
      if (semanticPrefix.test(utility) && !globalSelectors.has(utility) && !scoped.has(utility)) {
        return 'shared class is not defined in main.css or scoped styles'
      }
      return null
    }

    it('uses only defined text sizes, colors and shared form/section classes', () => {
      const violations: string[] = []
      for (const file of collectVueFiles(SRC_DIR)) {
        const content = readFileSync(file, 'utf-8')
        const scoped = new Set(
          [...content.matchAll(/<style[\s\S]*?<\/style>/g)].flatMap((s) =>
            [...s[0].matchAll(/\.([a-zA-Z][a-zA-Z0-9_-]*)/g)].map((m) => m[1])
          )
        )
        for (const token of new Set(templateClassTokens(content))) {
          const reason = problem(token, scoped)
          if (reason) violations.push(`${relative(SRC_DIR, file)}: ${token} (${reason})`)
        }
      }

      expect(violations, `Undefined template classes render with browser defaults:\n  ${violations.join('\n  ')}`).toEqual([])
    })
  })

  describe('variables.css and main.css define required tokens', () => {
    it('Louvre theme uses official brand colors', () => {
      const vars = readFileSync(join(STYLES_DIR, 'variables.css'), 'utf-8')
      const louvreTheme = vars.match(/\[data-theme="louvre"\]\s*\{[\s\S]*?\n\}/)?.[0]

      expect(louvreTheme).toBeTruthy()
      expect(louvreTheme).toContain('--accent-gold: #977b3c;')
      expect(louvreTheme).toContain('--accent-bronze: #843c00;')
      expect(louvreTheme).toContain('--text-primary: #ffffff;')
    })

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

    it('variables.css defines the approved font-family tokens', () => {
      const vars = readFileSync(join(STYLES_DIR, 'variables.css'), 'utf-8')
      expect(vars).toContain('--font-family-sans')
      expect(vars).toContain('--font-family-display')
    })

    it('main.css defines global chip and button classes', () => {
      const main = readFileSync(join(STYLES_DIR, 'main.css'), 'utf-8')
      expect(main).toContain('.chip')
      expect(main).toContain('.btn')
      expect(main).toContain('.btn-primary')
    })
  })
  describe('every theme meets WCAG AA for status foregrounds', () => {
    // The status colours are tuned for a near-black surface. The light theme
    // used to override none of them, so every badge, gain/loss figure and
    // confidence marker inherited a value that failed AA on a white card.
    //
    // The first version of this guard only checked the light theme, which was
    // the wrong shape: --color-negative also failed on `louvre` (4.10:1) and
    // `modern-greek` (4.48:1), and nothing would have caught it. Every theme
    // is checked, against both surfaces status text actually appears on, and
    // the ratios are computed rather than asserted so a colour edit cannot
    // quietly reintroduce the gap.
    const variables = readFileSync(join(STYLES_DIR, 'variables.css'), 'utf-8')

    function relativeLuminance(hex: string): number {
      const value = hex.replace('#', '')
      const channels = [0, 2, 4]
        .map((i) => parseInt(value.slice(i, i + 2), 16) / 255)
        .map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4))
      return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2]
    }

    function contrast(a: string, b: string): number {
      const [lighter, darker] = [relativeLuminance(a), relativeLuminance(b)].sort((x, y) => y - x)
      return (lighter + 0.05) / (darker + 0.05)
    }

    const rootScope = /:root\s*\{([\s\S]*?)\n\}/.exec(variables)?.[1] ?? ''
    const themes = new Map<string, string>()
    for (const match of variables.matchAll(/\[data-theme="([a-z-]+)"\]\s*\{([\s\S]*?)\n\}/g)) {
      themes.set(match[1], match[2])
    }

    // A theme inherits any token it does not override, so the effective value
    // is the theme's own or :root's. Resolving it this way is what catches a
    // dark-theme token that was never overridden because nobody looked.
    function resolve(scope: string, name: string): string | undefined {
      const own = new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})\\s*;`).exec(scope)?.[1]
      return own ?? new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})\\s*;`).exec(rootScope)?.[1]
    }

    const foregrounds = [
      'color-positive',
      'color-negative',
      'text-warning',
      'status-info-fg',
      'status-neutral-fg',
      'confidence-high',
      'confidence-medium',
      'confidence-low',
    ]

    it('defines every theme and every status foreground as a resolvable hex', () => {
      expect(themes.size, 'no [data-theme] blocks were parsed out of variables.css').toBeGreaterThan(0)
      const missing: string[] = []
      for (const [theme, scope] of themes) {
        for (const name of [...foregrounds, 'bg-card', 'bg-primary']) {
          if (!resolve(scope, name)) missing.push(`${theme}: --${name}`)
        }
      }
      expect(missing, `Unresolvable tokens:\n  ${missing.join('\n  ')}`).toEqual([])
    })

    it('clears 4.5:1 against the card and the page background in every theme', () => {
      const failures: string[] = []

      for (const [theme, scope] of themes) {
        const card = resolve(scope, 'bg-card')!
        const page = resolve(scope, 'bg-primary')!
        for (const name of foregrounds) {
          const value = resolve(scope, name)!
          const onCard = contrast(value, card)
          const onPage = contrast(value, page)
          if (onCard < 4.5) {
            failures.push(`${theme}: --${name} ${value} on --bg-card ${card} is ${onCard.toFixed(2)}:1`)
          }
          if (onPage < 4.5) {
            failures.push(`${theme}: --${name} ${value} on --bg-primary ${page} is ${onPage.toFixed(2)}:1`)
          }
        }
      }

      expect(
        failures,
        `Status foregrounds must reach WCAG AA (4.5:1) in every theme:\n  ${failures.join('\n  ')}`
      ).toEqual([])
    })
  })
})
