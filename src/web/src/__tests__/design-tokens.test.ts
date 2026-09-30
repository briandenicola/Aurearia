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

    // Per-file budget (#787). This was a single net total of 190 against a
    // real count of 7, so a file could add roughly 180 literals unnoticed.
    // #787 then mapped those 7 onto the status tokens, so the map is empty:
    // any file that adds one fails. Add an entry only with a stated reason.
    const HEX_BUDGET = new Map<string, number>()

    it('hex color violations stay within their per-file budget', () => {
      const violations: string[] = []

      for (const file of vueFiles) {
        const content = readFileSync(file, 'utf-8')
        const styles = extractScopedStyles(content)
        if (styles.length === 0) continue
        const name = relative(SRC_DIR, file).split(sep).join('/')
        let count = 0

        for (const style of styles) {
          const cleaned = style.replace(/\/\*[\s\S]*?\*\//g, '')
          const noVarFallback = cleaned.replace(/var\([^)]*\)/g, '')
          for (const match of noVarFallback.matchAll(HEX_COLOR)) {
            if (!ALLOWED_HEX.has(match[0].toLowerCase())) count++
          }
        }

        const allowed = HEX_BUDGET.get(name) ?? 0
        if (count > allowed) violations.push(`${name}: ${count} hex literals, budget ${allowed}`)
      }

      expect(
        violations,
        `Scoped-style hex colours exceed their per-file budget. ` +
        `Use CSS variables from variables.css instead:\n  ${violations.join('\n  ')}`
      ).toEqual([])
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
      //
      // Two entries went *up* in review: the A-F grade ramps in
      // CollectionHealthScorecard and NeedsAttentionQueue had grades B and F
      // converted to status tokens because their values matched, while A, C
      // and D stayed literal. That is the value-identity conversion the slice
      // otherwise refuses, and it half-bound a gradient to status semantics.
      // Reverting it is correct and costs 8 literals.
      //
      // Two entries came *off* the list entirely: CollectionHealthTrendIndicator
      // and WishlistPage rendered status badges with raw palette colours and
      // arbitrary rgba() fills. Binding them to the status tokens both removed
      // four literals and brought those badges under the contrast guard, which
      // cannot measure a colour that is not in the token system.
      //
      // #787 mapped the remaining near-duplicate status colours onto the
      // status tokens. What is left is deliberate: the A-F grade ramps are a
      // gradient, not status semantics; the purple bar has no semantic token;
      // and the white 0.04 lift has no themed equivalent (the overlay tokens
      // are black scrims).
      const COLOR_BUDGET = new Map([
        ['components/stats/CollectionHealthScorecard.vue', 12],
        ['components/collection/NeedsAttentionQueue.vue', 10],
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

    it('keeps hardcoded colour literals in script blocks within budget', () => {
      // A class string built in <script> (e.g. a computed badgeCls) escaped
      // both the template and the scoped-style guards (#787). Same per-file
      // ratchet. What remains is data rather than styling: user-selectable tag
      // colours, fallback set/category colours stored with records, and the
      // BaseButton/BaseBadge primitives, which are out of #787's scope.
      const SCRIPT_BUDGET = new Map([
        ['components/settings/SettingsDataSection.vue', 12],
        ['components/ui/BaseButton.vue', 4],
        ['components/ui/BaseBadge.vue', 3],
        ['pages/FollowerCoinDetailPage.vue', 2],
        ['pages/SetProposalReviewPage.vue', 2],
        ['pages/SetDetailPage.vue', 1],
        ['pages/TimelinePage.vue', 1],
      ])
      const violations: string[] = []

      for (const file of vueFiles) {
        const content = readFileSync(file, 'utf-8')
        const name = relative(SRC_DIR, file).split(sep).join('/')
        let count = 0
        for (const block of content.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script\s*>/gi)) {
          const code = block[1].replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/[^\n]*/g, '$1')
          count += (code.match(/rgba?\(|#[0-9a-fA-F]{3,8}\b/g) ?? []).length
        }
        const allowed = SCRIPT_BUDGET.get(name) ?? 0
        if (count > allowed) {
          violations.push(`${name}: ${count} literals, budget ${allowed} — use a design token`)
        }
      }

      expect(
        violations,
        `Hardcoded script colour literals exceed their per-file budget:\n  ${violations.join('\n  ')}`
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
  describe('status text meets WCAG AA wherever it renders', () => {
    // This guard has now been wrong four times, in the same way every time:
    // it enumerated the cases already found instead of deriving them.
    //
    //   1. Checked only [data-theme="light"]      -> missed louvre, modern-greek.
    //   2. Checked every theme, but measured the  -> certified badges that were
    //      foreground on the *bare* surface          really 4.00:1.
    //   3. Skipped the default palette (bare      -> missed default info at 4.04.
    //      :root carries no data-theme attribute)
    //   4. Hand-listed the fg/fill pairs and two  -> missed --bg-secondary, where
    //      surfaces                                  every tone failed, and missed
    //                                                the confidence-on-fill pills.
    //
    // So this version derives its cases from the source rather than listing
    // them: surfaces come from the --bg-* tokens declared in :root, and the
    // fg/fill pairings come from the templates that actually render them.
    // Adding a badge on a new surface, or a new fg/fill combination, extends
    // this guard automatically instead of silently escaping it.
    // Comments are stripped before scoping: variables.css now carries several
    // multi-line notes that name tokens, and a token written inside a comment
    // would otherwise be read as a declaration.
    const stripComments = (css: string): string => css.replace(/\/\*[\s\S]*?\*\//g, '')
    const variables = stripComments(readFileSync(join(STYLES_DIR, 'variables.css'), 'utf-8'))
    const main = stripComments(readFileSync(join(STYLES_DIR, 'main.css'), 'utf-8'))

    const rootScope = /:root\s*\{([\s\S]*?)\n\}/.exec(variables)?.[1] ?? ''
    // The default palette lives in bare `:root` with no data-theme attribute.
    // It is a theme users actually see, so it is iterated, not just used as
    // the fallback scope.
    const themes = new Map<string, string>([['default', rootScope]])
    for (const match of variables.matchAll(/\[data-theme="([a-z-]+)"\]\s*\{([\s\S]*?)\n\}/g)) {
      // A second block for the same theme would silently replace the first and
      // hide every token in it, so refuse rather than overwrite.
      expect(themes.has(match[1]), `variables.css declares [data-theme="${match[1]}"] more than once`).toBe(false)
      themes.set(match[1], match[2])
    }

    function declaration(scope: string, name: string): string | undefined {
      return new RegExp(`--${name}:\\s*([^;]+);`).exec(scope)?.[1].trim()
    }

    // Follows var() indirection *in the theme's own scope*, because tokens
    // alias each other: --status-error-fg is var(--color-negative), and
    // --status-warning-fg is var(--text-warning). An alias therefore resolves
    // to a different value per theme even when only :root declares it.
    function resolve(scope: string, name: string, depth = 0): string | undefined {
      const value = declaration(scope, name) ?? declaration(rootScope, name)
      if (!value) return undefined
      const indirect = /^var\(--([a-z0-9-]+)\)$/.exec(value)
      if (indirect && depth < 5) return resolve(scope, indirect[1], depth + 1)
      return value
    }

    type Rgba = { rgb: [number, number, number]; alpha: number }

    function parseColor(value: string): Rgba | undefined {
      if (/^#[0-9a-fA-F]{6}$/.test(value)) {
        const hex = value.slice(1)
        return { rgb: [0, 2, 4].map((i) => parseInt(hex.slice(i, i + 2), 16)) as [number, number, number], alpha: 1 }
      }
      const parts = /rgba?\(([^)]+)\)/.exec(value)?.[1].split(',').map((p) => parseFloat(p))
      if (!parts || parts.length < 3 || parts.some((n) => Number.isNaN(n))) return undefined
      return { rgb: parts.slice(0, 3) as [number, number, number], alpha: parts.length > 3 ? parts[3] : 1 }
    }

    function composite(over: Rgba, under: Rgba): Rgba {
      return {
        rgb: over.rgb.map((c, i) => c * over.alpha + under.rgb[i] * (1 - over.alpha)) as [number, number, number],
        alpha: 1,
      }
    }

    function relativeLuminance(rgb: [number, number, number]): number {
      const channels = rgb
        .map((c) => c / 255)
        .map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4))
      return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2]
    }

    // `a` is composited over `b` first: a translucent foreground measured as
    // if it were opaque would be reported as higher contrast than it renders.
    // Every foreground is currently an opaque hex, so this is a guard against
    // a future rgba() foreground rather than a live correction.
    function contrast(a: Rgba, b: Rgba): number {
      const front = a.alpha < 1 ? composite(a, b) : a
      const [lighter, darker] = [relativeLuminance(front.rgb), relativeLuminance(b.rgb)].sort((x, y) => y - x)
      return (lighter + 0.05) / (darker + 0.05)
    }

    // Tailwind utility -> variables.css token, read from main.css's @theme
    // inline block, so `text-loss` is known to mean --color-negative.
    const themeInline = /@theme inline\s*\{([\s\S]*?)\n\}/.exec(main)?.[1] ?? ''
    const utilityToken = new Map<string, string>()
    for (const match of themeInline.matchAll(/--color-([a-z0-9-]+):\s*var\(--([a-z0-9-]+)\);/g)) {
      utilityToken.set(match[1], match[2])
    }

    // Every surface a fill can sit on, derived from :root's --bg-* tokens.
    // All of them are required rather than arguing about which ones host a
    // badge: --bg-input and --bg-secondary are the extreme surfaces in most
    // themes, so clearing them clears the rest, and no future placement can
    // land somewhere unmeasured.
    const SURFACES = [...rootScope.matchAll(/--(bg-[a-z-]+):/g)].map((m) => m[1])
    const AA = 4.5

    // Tailwind text-* utilities that set size or alignment, not colour.
    const NON_COLOUR_TEXT =
      /^(xs|sm|base|lg|xl|[2-9]xl|left|right|center|justify|start|end|ellipsis|clip|wrap|nowrap|balance|pretty|body|chip|label|micro|display|heading)$/

    // The tokens components consume, not the source tokens they alias to.
    const TONES = ['success', 'error', 'warning', 'info', 'neutral']
    const BARE_TEXT = [
      'color-positive',
      'color-negative',
      'text-warning',
      'confidence-high',
      'confidence-medium',
      'confidence-low',
      ...TONES.map((tone) => `status-${tone}-fg`),
    ]

    // A fill is measured when it is a status fill, or when it paints a status
    // foreground token solid (bg-gain, bg-loss). The second set is derived
    // from BARE_TEXT rather than listed.
    function isMeasuredFill(utility: string): boolean {
      const token = utilityToken.get(utility)
      return !!token && (/^status-[a-z]+-(?:bg|tint)$/.test(token) || BARE_TEXT.includes(token))
    }

    type ClassPair = { fill: string; fillAlpha: number; text: string; textAlpha: number }

    // Parses one class list into the fill/text pairs that render together.
    // Pairs are made per variant state: a hover: fill renders with the hover:
    // text colour if there is one, otherwise with the colour of the next less
    // specific state (dark:hover: -> hover: -> resting). It never renders with
    // a resting colour that its own variant replaces. Important markers (a
    // leading or trailing !) are stripped, and an opacity modifier (/NN) is
    // carried as an alpha rather than dropped, so neither spelling escapes.
    function parseClassList(chunk: string, isFill: (utility: string) => boolean): ClassPair[] {
      type Entry = { name: string; alpha: number }
      const fills = new Map<string, Entry>()
      const texts = new Map<string, Entry[]>()
      for (const cls of chunk.split(/\s+/).filter(Boolean)) {
        const cut = cls.lastIndexOf(':')
        const variant = cls.slice(0, cut + 1)
        const utility = cls.slice(cut + 1).replace(/^!|!$/g, '')
        const parsed = /^(bg|text)-([a-z0-9-]+)(?:\/([0-9.]+))?$/.exec(utility)
        if (!parsed) continue
        const entry = { name: parsed[2], alpha: parsed[3] ? parseFloat(parsed[3]) / 100 : 1 }
        if (parsed[1] === 'bg' && isFill(entry.name)) fills.set(variant, entry)
        if (parsed[1] === 'text' && !NON_COLOUR_TEXT.test(entry.name)) {
          texts.set(variant, [...(texts.get(variant) ?? []), entry])
        }
      }
      function lookup<T>(map: Map<string, T>, variant: string): T | undefined {
        for (let v = variant; ; v = v.slice(v.indexOf(':') + 1)) {
          if (map.has(v)) return map.get(v)
          if (v === '') return undefined
        }
      }
      const pairs: ClassPair[] = []
      for (const variant of new Set(['', ...fills.keys(), ...texts.keys()])) {
        const fill = lookup(fills, variant)
        if (!fill) continue
        for (const text of lookup(texts, variant) ?? []) {
          pairs.push({ fill: fill.name, fillAlpha: fill.alpha, text: text.name, textAlpha: text.alpha })
        }
      }
      return pairs
    }

    type Pairing = { fg: string; fgAlpha: number; fill: string; fillAlpha: number; sources: Set<string> }
    const pairings = new Map<string, Pairing>()
    const nonToken: string[] = []

    function addPairing(fg: string, fill: string, source: string, fgAlpha = 1, fillAlpha = 1): void {
      const key = `${fg}/${fgAlpha}|${fill}/${fillAlpha}`
      if (!pairings.has(key)) pairings.set(key, { fg, fgAlpha, fill, fillAlpha, sources: new Set() })
      pairings.get(key)!.sources.add(source)
    }

    // A pairing only exists when the fill and the text colour sit in the same
    // class list on the same element, so scan bounded by element start-tag and
    // treat each quoted literal inside a :class binding as its own branch.
    function classChunks(tag: string): string[] {
      const chunks: string[] = []
      const staticClass = /(?:^|\s)class="([^"]*)"/.exec(tag)
      if (staticClass) chunks.push(staticClass[1])
      const bound = /(?::class|v-bind:class)="([\s\S]*?)"/.exec(tag)
      if (bound) for (const literal of bound[1].matchAll(/'([^']*)'/g)) chunks.push(literal[1])
      return chunks
    }

    // Class strings can also be built in <script>, e.g. a computed
    // `badgeCls: 'bg-status-success-bg text-status-success-fg'` bound with
    // :class. Each string literal there is treated as one class list; one
    // that holds no measured fill yields no pairs.
    function scriptChunks(text: string): string[] {
      const chunks: string[] = []
      for (const block of text.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script\s*>/gi)) {
        // A template literal's ${} expressions are blanked, so its static
        // classes are still read and its backticks cannot pair with another's.
        for (const literal of block[1].matchAll(/'([^'\n]*)'|"([^"\n]*)"|`([^`]*)`/g)) {
          chunks.push(literal[1] ?? literal[2] ?? literal[3].replace(/\$\{[^}]*\}/g, ' '))
        }
      }
      return chunks
    }

    for (const file of collectVueFiles(SRC_DIR)) {
      const text = readFileSync(file, 'utf-8')
      const rel = relative(SRC_DIR, file).split(sep).join('/')
      const chunks = [...[...text.matchAll(/<[a-zA-Z][^>]*>/g)].flatMap((tag) => classChunks(tag[0])), ...scriptChunks(text)]
      for (const chunk of chunks) {
        for (const pair of parseClassList(chunk, isMeasuredFill)) {
          const token = utilityToken.get(pair.text)
          if (!token) {
            nonToken.push(`${rel}: text-${pair.text} on bg-${pair.fill}`)
            continue
          }
          addPairing(token, utilityToken.get(pair.fill)!, rel, pair.textAlpha, pair.fillAlpha)
        }
      }
      // Components may build the token names dynamically, e.g. BaseStatusBadge:
      //   backgroundColor: `var(--status-${props.tone}-bg)`
      //   color:           `var(--status-${props.tone}-fg)`
      // A literal regex cannot see those, so expand across the tone union
      // declared in the same file.
      if (/var\(--status-\$\{[^}]+\}-bg\)/.test(text) && /var\(--status-\$\{[^}]+\}-fg\)/.test(text)) {
        const union = /StatusTone\s*=\s*([^\n]+)/.exec(text)?.[1] ?? ''
        const tones = [...union.matchAll(/'([a-z]+)'/g)].map((m) => m[1])
        expect(tones.length, `${rel} builds status tokens dynamically but declares no tone union`).toBeGreaterThan(0)
        for (const tone of tones) addPairing(`status-${tone}-fg`, `status-${tone}-bg`, rel)
      }
    }

    function required(theme: string, scope: string, name: string): Rgba {
      const value = resolve(scope, name)
      expect(value, `${theme}: --${name} does not resolve to a colour`).toBeTruthy()
      const parsed = parseColor(value!)
      expect(parsed, `${theme}: --${name} = "${value}" is not a parseable colour`).toBeTruthy()
      return parsed!
    }

    it('derives its cases from the stylesheets and the templates', () => {
      expect(themes.size, 'no theme scopes were parsed out of variables.css').toBeGreaterThan(1)
      expect(SURFACES.length, 'no --bg-* surface tokens were parsed out of :root').toBeGreaterThan(1)
      expect(utilityToken.size, 'no colour utilities were parsed out of main.css @theme inline').toBeGreaterThan(1)
      expect(pairings.size, 'no status fg/fill pairings were found in any template').toBeGreaterThan(1)
      for (const [theme, scope] of themes) {
        for (const name of [...BARE_TEXT, ...SURFACES, ...TONES.map((t) => `status-${t}-bg`)]) {
          required(theme, scope, name)
        }
      }
    })

    it('parses class lists into the pairs that actually render', () => {
      const fill = (u: string): boolean => u.startsWith('status-')
      const pairs = (chunk: string): string[] =>
        parseClassList(chunk, fill).map(
          (p) => `${p.text}${p.textAlpha < 1 ? `/${p.textAlpha * 100}` : ''} on ${p.fill}${p.fillAlpha < 1 ? `/${p.fillAlpha * 100}` : ''}`
        )
      // Resting fill and colour.
      expect(pairs('bg-status-error-bg text-loss')).toEqual(['loss on status-error-bg'])
      // Resting fill with only a hover: colour: both states render on the fill.
      expect(pairs('bg-status-error-bg text-muted hover:text-loss').sort()).toEqual(
        ['loss on status-error-bg', 'muted on status-error-bg']
      )
      // hover: fill with a resting colour only: the resting colour shows on it.
      expect(pairs('text-muted hover:bg-status-error-tint')).toEqual(['muted on status-error-tint'])
      // hover: fill with a hover: colour: the resting colour never shows on it.
      expect(pairs('text-muted hover:bg-status-error-tint hover:text-loss')).toEqual(['loss on status-error-tint'])
      // Stacked variants fall back through the chain, not straight to resting.
      expect(pairs('text-muted hover:bg-status-info-bg hover:text-gold dark:hover:text-loss').sort()).toEqual(
        ['gold on status-info-bg', 'loss on status-info-bg']
      )
      expect(pairs('text-muted group-hover:bg-status-info-bg')).toEqual(['muted on status-info-bg'])
      // Important markers in either position.
      expect(pairs('!bg-status-error-bg text-loss!')).toEqual(['loss on status-error-bg'])
      expect(pairs('bg-status-error-bg! !text-loss')).toEqual(['loss on status-error-bg'])
      // Opacity modifiers are carried, not dropped.
      expect(pairs('bg-status-error-bg/50 text-loss/80')).toEqual(['loss/80 on status-error-bg/50'])
      // Size and alignment utilities are not colours; unmeasured fills pair nothing.
      expect(pairs('bg-status-error-bg text-sm text-center')).toEqual([])
      expect(pairs('bg-card text-loss')).toEqual([])
    })

    it('reads class strings built in <script>', () => {
      expect(scriptChunks("<script setup>\nconst c = { badgeCls: 'bg-gain text-surface' }\n</script>")).toContain(
        'bg-gain text-surface'
      )
      expect(scriptChunks('<script>\nconst a = `bg-status-error-bg ${x} text-loss`\n</script>')).toContain(
        'bg-status-error-bg   text-loss'
      )
      expect(isMeasuredFill('gain'), 'a solid status foreground used as a fill is measured').toBe(true)
      expect(isMeasuredFill('status-error-tint')).toBe(true)
      expect(isMeasuredFill('card')).toBe(false)
    })

    it('pairs status fills only with colours from the token system', () => {
      // A raw palette colour (text-red-400) or an arbitrary value on a status
      // fill cannot be themed and cannot be measured here, so it is a failure
      // rather than a silent omission.
      expect(
        nonToken,
        `Status fills must be paired with token colours:\n  ${nonToken.join('\n  ')}`
      ).toEqual([])
    })

    it('clears 4.5:1 as plain text on every surface', () => {
      const failures: string[] = []
      for (const [theme, scope] of themes) {
        for (const surface of SURFACES) {
          const under = required(theme, scope, surface)
          for (const name of BARE_TEXT) {
            const ratio = contrast(required(theme, scope, name), under)
            if (ratio < AA) failures.push(`${theme}: --${name} on --${surface} is ${ratio.toFixed(2)}:1`)
          }
        }
      }
      expect(failures, `Status text must reach WCAG AA (4.5:1):\n  ${failures.join('\n  ')}`).toEqual([])
    })

    it('clears 4.5:1 composited on the translucent fill, on every surface', () => {
      // Status text renders on --status-{tone}-bg, a ~0.15 alpha fill, over
      // the surface. The fill shifts the surface toward the foreground's own
      // hue, so it always *reduces* contrast — measuring the bare surface was
      // not conservative, it was simply the wrong measurement.
      const failures: string[] = []
      for (const [theme, scope] of themes) {
        for (const surface of SURFACES) {
          const under = required(theme, scope, surface)
          for (const { fg, fgAlpha, fill, fillAlpha, sources } of pairings.values()) {
            const fillColour = required(theme, scope, fill)
            const fgColour = required(theme, scope, fg)
            const composited = composite({ ...fillColour, alpha: fillColour.alpha * fillAlpha }, under)
            const ratio = contrast({ ...fgColour, alpha: fgColour.alpha * fgAlpha }, composited)
            if (ratio < AA) {
              const opacity = (alpha: number): string => (alpha < 1 ? ` at ${Math.round(alpha * 100)}%` : '')
              failures.push(
                `${theme}: --${fg}${opacity(fgAlpha)} on --${fill}${opacity(fillAlpha)} over --${surface} is ${ratio.toFixed(2)}:1` +
                  ` [${[...sources].join(', ')}]`
              )
            }
          }
        }
      }
      expect(
        failures,
        `Status text must reach WCAG AA (4.5:1) against the composited fill:\n  ${failures.join('\n  ')}`
      ).toEqual([])
    })
  })
})
