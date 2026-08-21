/**
 * Architectural guard: swipe engine consolidation.
 *
 * Design authority: .squad/decisions/inbox/maximus-pwa-swipe-reliability-review.md §9-§13
 * Purpose: prevent new inline one-finger horizontal swipe state machines from being
 * added to the codebase after Phase 2 migration to the shared `useSwipeGesture` primitive.
 *
 * HEURISTIC (two complementary rules — matched files outside the allowlist are flagged)
 *
 * Rule A (capture-based): file contains BOTH `setPointerCapture` AND `pointercancel`.
 *   Catches engines that explicitly acquire exclusive pointer tracking (Gallery, Tray pattern).
 *
 * Rule B (three-handler): file contains ALL THREE of `pointerdown`, `pointermove`, `pointerup`.
 *   Catches engines that rely on implicit capture or passive listeners without setPointerCapture
 *   (the original useCoinDetailSwipeNav pattern that Rule A alone would have missed).
 *
 * POST-PHASE-2 ALLOWLIST — Gallery (SwipeGallery.vue) and Tray (TrayViewPage.vue) have been
 * migrated to the shared useSwipeGesture primitive. Five files with genuinely distinct purposes remain:
 *
 *   useSwipeGesture.ts      — The shared primitive ITSELF: owns setPointerCapture + cancel handling
 *   App.vue                 — FAB drag-to-reposition; distinct gesture surface/purpose
 *   ZoomableSurface.vue     — Two-finger pinch-zoom and pan; fundamentally distinct from 1-finger nav
 *   ImageProcessor.vue      — Image crop/pan tool; distinct purpose and UX
 *   usePullToRefresh.ts     — Vertical pull-to-refresh gesture; vertical axis, distinct lifecycle
 *
 * SOURCE STRING SCAN: justified here because:
 *   (a) the heuristic is structural/architectural, not logic-level
 *   (b) jsdom cannot execute the actual gesture lifecycle
 *   (c) a behavioural test cannot detect the mere presence of a duplicate engine without running it
 */

import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'

// Root of the Vue/TS source tree
const SRC_ROOT = join(process.cwd(), 'src')

// Post-Phase-2 allowlist: Gallery and Tray have been migrated; only genuinely distinct surfaces remain.
const ALLOWED_FILES = new Set([
  'composables/useSwipeGesture.ts',       // The shared primitive itself; owns the full pointer lifecycle
  'App.vue',                              // FAB drag-to-reposition; distinct gesture surface/purpose
  'components/ZoomableSurface.vue',       // Two-finger pinch/pan; genuinely distinct
  'components/ImageProcessor.vue',        // Image crop/pan tool; distinct purpose
  'composables/usePullToRefresh.ts',      // Vertical pull-to-refresh; different axis
])

/**
 * Recursively walk a directory and collect file paths (relative to SRC_ROOT)
 * matching .vue, .ts, .js extensions (source files only; skip test files and node_modules).
 */
function collectSourceFiles(dir: string): string[] {
  const results: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === '__tests__' || entry.name === 'dist') continue
      results.push(...collectSourceFiles(join(dir, entry.name)))
    } else if (/\.(vue|ts|js)$/.test(entry.name) && !entry.name.endsWith('.test.ts') && !entry.name.endsWith('.spec.ts')) {
      results.push(join(dir, entry.name))
    }
  }
  return results
}

describe('swipe engine consolidation guard', () => {
  it('no source file outside the allowlist owns an inline pointer-based gesture state machine', () => {
    const allFiles = collectSourceFiles(SRC_ROOT)
    const violations: string[] = []

    for (const absPath of allFiles) {
      const relPath = relative(SRC_ROOT, absPath).replace(/\\/g, '/')
      if (ALLOWED_FILES.has(relPath)) continue

      const source = readFileSync(absPath, 'utf-8')
      // Rule A: explicit capture + cancel — the Gallery/Tray engine pattern.
      const isRuleA = source.includes('setPointerCapture') && source.includes('pointercancel')
      // Rule B: all three pointer event handlers — the original coin-detail implicit-capture pattern.
      const isRuleB = source.includes('pointerdown') && source.includes('pointermove') && source.includes('pointerup')

      if (isRuleA || isRuleB) {
        violations.push(relPath)
      }
    }

    if (violations.length > 0) {
      expect.fail(
        'The following source files appear to contain a new inline pointer-based gesture engine.\n'
        + 'If this is intentional (e.g. a genuinely distinct gesture surface), add the file to\n'
        + 'ALLOWED_FILES in swipe-engine-consolidation.guard.test.ts with an explanation.\n'
        + 'Otherwise, extract the gesture into the shared useSwipeGesture primitive.\n\n'
        + violations.map(f => `  ${f}`).join('\n'),
      )
    }
  })

  it('allowlisted files still exist (guard stays relevant; fail = file was renamed or removed)', () => {
    const missingFiles: string[] = []
    for (const relPath of ALLOWED_FILES) {
      // join() is separator-agnostic when given forward-slash path segments on any platform.
      const absPath = join(SRC_ROOT, relPath)
      try {
        readFileSync(absPath, 'utf-8')
      } catch {
        missingFiles.push(relPath)
      }
    }

    if (missingFiles.length > 0) {
      expect.fail(
        'The following allowlisted gesture-engine files no longer exist.\n'
        + 'If they were migrated to useSwipeGesture (Phase 2 complete), remove them from ALLOWED_FILES.\n'
        + 'If they were renamed, update ALLOWED_FILES to match the new path.\n\n'
        + missingFiles.map(f => `  ${f}`).join('\n'),
      )
    }
  })

  // To add a future genuinely distinct gesture surface (e.g. a new pinch-zoom component):
  //   1. Add the file to ALLOWED_FILES with a clear explanation of why it differs.
  //   2. Verify both tests above still pass.
  it('shared primitive is always in the allowlist (guard must not flag its own engine)', () => {
    // The shared primitive owns all three pointer events and setPointerCapture; it must be allowlisted
    // or the violation scan would flag the engine it was written to protect.
    expect(ALLOWED_FILES.has('composables/useSwipeGesture.ts')).toBe(true)
  })
})