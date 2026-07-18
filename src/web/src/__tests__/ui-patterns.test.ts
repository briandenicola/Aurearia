import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { join } from 'path'

const REPO_ROOT = join(__dirname, '..', '..', '..', '..')
const SRC_DIR = join(REPO_ROOT, 'src', 'web', 'src')
const COPILOT_INSTRUCTIONS = join(REPO_ROOT, '.github', 'copilot-instructions.md')

function readRepoFile(pathFromSrc: string): string {
  return readFileSync(join(SRC_DIR, pathFromSrc), 'utf-8')
}

describe('UI pattern recipes', () => {
  it('documents reusable UI recipes for future agents', () => {
    const instructions = readFileSync(COPILOT_INSTRUCTIONS, 'utf-8')

    expect(instructions).toContain('#### UI Pattern Recipes')
    expect(instructions).toContain('identify the closest existing page or component pattern')
    expect(instructions).toContain('Keep controls in one row: `< Previous` then current label then `Next >`')
    expect(instructions).toContain('Keep Gallery and Tray under the Collection submenu')
  })

  it('keeps tray pagination in one Previous, drawer label, Next row', () => {
    const trayControls = readRepoFile(join('components', 'tray', 'TrayControls.vue'))

    // TrayControls no longer has scoped `.drawer-navigation`/`.drawer-label`
    // CSS classes or a `nav-btn` class — the fixed positioning, the nowrap
    // row, and the button styling are all inline Tailwind utility classes now.
    expect(trayControls).toContain('Prev')
    expect(trayControls).toContain('Tray {{ drawerIndex + 1 }} of {{ totalDrawers }}')
    expect(trayControls).toContain('Next')
    // Row stays a single non-wrapping flex row (Prev, label, Next side by side).
    expect(trayControls).toContain('flex flex-nowrap items-center justify-center gap-4 rounded-full')
    // Pagination pins to the bottom of the viewport, clearing the safe area.
    expect(trayControls).toContain("fixed bottom-[calc(1rem+env(safe-area-inset-bottom))]")
    expect(trayControls).not.toContain('Felt Color')
    expect(trayControls).not.toContain('felt-theme-selector')
  })

  it('keeps tray felt color in Settings appearance instead of the tray page', () => {
    const settingsAppearance = readRepoFile(join('components', 'settings', 'SettingsAppearanceSection.vue'))
    const trayPage = readRepoFile(join('pages', 'TrayViewPage.vue'))
    const museumTray = readRepoFile(join('components', 'tray', 'MuseumTray.vue'))

    expect(settingsAppearance).toContain('Tray Felt Color')
    expect(settingsAppearance).toContain('set-tray-felt-color')
    expect(trayPage).not.toContain('@update:felt-theme')
    expect(trayPage).toContain('const coinsPerDrawer = 12')
    expect(trayPage).toContain('while (true)')
    expect(trayPage).toContain('limit: trayPageLimit')
    expect(museumTray).toContain('grid-template-columns: repeat(6, minmax(0, 1fr))')
  })

  it('keeps Identify Coin camera-first with Add Coin upload icon pattern', () => {
    const lookupPage = readRepoFile(join('pages', 'CoinLookupPage.vue'))
    const addCoinPage = readRepoFile(join('pages', 'AddCoinPage.vue'))
    const inlineCameraPanel = readRepoFile(join('components', 'InlineCameraCapturePanel.vue'))

    expect(lookupPage).toContain('InlineCameraCapturePanel')
    expect(addCoinPage).toContain('InlineCameraCapturePanel')
    expect(inlineCameraPanel).toContain('ref="cameraVideo"')
    expect(inlineCameraPanel).toContain('Start Camera')
    expect(inlineCameraPanel).toContain('@click="startCamera"')
    expect(addCoinPage).not.toContain('await startCamera()')
    expect(inlineCameraPanel).toContain('class="shutter-btn"')
    expect(inlineCameraPanel).toContain('class="upload-icon-btn"')
    expect(inlineCameraPanel).toContain('<Images :size="20" />')
    expect(lookupPage).not.toContain('CameraCaptureModal')
    expect(lookupPage).not.toContain('Take Photo')
    expect(lookupPage).not.toContain('Upload Image')
    expect(lookupPage).not.toContain('title="Back"')
  })

  it('keeps PWA timeline and set coin actions compact', () => {
    const timelinePage = readRepoFile(join('pages', 'TimelinePage.vue'))
    const setDetailPage = readRepoFile(join('pages', 'SetDetailPage.vue'))

    // Timeline rows are now a flex row (not a grid) but keep the same
    // shrink-to-fit + clip guarantees via `min-w-0` and `overflow-hidden`.
    expect(timelinePage).toContain('min-w-0')
    expect(timelinePage).toContain('overflow-hidden')
    // Set coin action buttons share a common Tailwind class set instead of a
    // `.set-coin-action-btn` name, and their row stays single-line via
    // `flex-nowrap` instead of a `flex-wrap: nowrap` CSS rule.
    expect(setDetailPage).toContain('rounded-full border border-border-subtle bg-input text-text-secondary')
    expect(setDetailPage).toContain('<ChevronUp :size="16" />')
    expect(setDetailPage).toContain('<ChevronDown :size="16" />')
    expect(setDetailPage).toContain('<X :size="16" />')
    expect(setDetailPage).toContain('class="flex flex-nowrap justify-end gap-1.5" aria-label="Set coin actions"')
    expect(setDetailPage).not.toContain('>Up<')
    expect(setDetailPage).not.toContain('>Down<')
    expect(setDetailPage).not.toContain('>Remove<')
  })

  it('keeps Sets list cards refined and count-forward', () => {
    const setCard = readRepoFile(join('components', 'sets', 'SetDashboardCard.vue'))

    // These CSS values now come from Tailwind utilities: `min-h-20` = 5rem,
    // `h-16` = 4rem (Tailwind's default 0.25rem spacing scale), `items-end` =
    // align-items: flex-end, `text-[2.75rem]` is the literal font size, and
    // `rounded-md` resolves to the app's `--radius-md` theme token.
    expect(setCard).toContain('Curated group')
    expect(setCard).toContain('min-[561px]:min-h-20')
    expect(setCard).toContain('min-[561px]:h-16')
    expect(setCard).toContain('items-end')
    expect(setCard).toContain('min-[561px]:text-[2.75rem]')
    expect(setCard).toContain('rounded-md')
    expect(setCard).not.toContain('completion-meter')
    expect(setCard).not.toContain('Completion set')
  })

  it('keeps collection coin images from over-zooming or clipping', () => {
    const coinCard = readRepoFile('components/CoinCard.vue')
    const swipeGallery = readRepoFile('components/SwipeGallery.vue')

    // CoinCard's image sizing moved to inline Tailwind utilities
    // (`object-contain`, `group-hover:scale-[1.02]`); SwipeGallery still uses
    // a scoped <style> block with the literal CSS.
    expect(coinCard).toContain('object-contain')
    expect(coinCard).toContain('scale-[1.02]')
    expect(coinCard).not.toContain('object-cover')
    expect(swipeGallery).toContain('object-fit: contain')
    expect(swipeGallery).toContain('transform: scale(1.05)')
    expect(swipeGallery).not.toContain('transform: scale(1.28)')
  })

  it('keeps the PWA agent button viewport-fixed globally', () => {
    const app = readRepoFile('App.vue')

    // The fab button no longer carries a named `.agent-fab` class (that CSS
    // rule in main.css is now dead/orphaned) — position: fixed, the
    // bottom/right safe-area offsets, and touch-action: none are all inline
    // Tailwind utility classes on the button itself.
    expect(app).toContain('<Teleport to="body">')
    expect(app).toContain(':style="fabPositionStyle"')
    expect(app).toContain('@pointerdown="startAgentFabDrag"')
    expect(app).toContain('@pointermove="moveAgentFabDrag"')
    expect(app).toContain('@pointerup="stopAgentFabDrag"')

    const fabButtonMatch = app.match(/<button\s+v-if="isPwa[^>]*aria-label="Open AI Agent"/s)
    expect(fabButtonMatch).not.toBeNull()
    const fabButtonSource = fabButtonMatch![0]
    expect(fabButtonSource).toContain('fixed')
    expect(fabButtonSource).toContain('touch-none')
    expect(fabButtonSource).toContain('bottom-[calc(24px+env(safe-area-inset-bottom))]')
    expect(fabButtonSource).toContain('right-[calc(24px+env(safe-area-inset-right))]')
  })

  it('keeps the agent chat overlay above tray pagination controls', () => {
    const chat = readRepoFile(join('components', 'CoinSearchChat.vue'))
    const trayControls = readRepoFile(join('components', 'tray', 'TrayControls.vue'))

    // Neither component has a scoped `.chat-overlay`/`.tray-controls` class
    // anymore — both `position: fixed` and their z-index live inline as
    // Tailwind `fixed` + `z-[N]` classes on their root elements.
    const chatZIndexMatch = chat.match(/<div class="fixed inset-0 z-\[(\d+)\]/)
    const trayZIndexMatch = trayControls.match(/class="z-\[(\d+)\][^"]*"/)

    expect(chatZIndexMatch).not.toBeNull()
    expect(trayZIndexMatch).not.toBeNull()
    expect(trayControls).toContain("'fixed bottom-[calc(1rem+env(safe-area-inset-bottom))]")

    const chatZIndex = Number(chatZIndexMatch?.[1] ?? 0)
    const trayZIndex = Number(trayZIndexMatch?.[1] ?? 0)

    expect(chatZIndex).toBeGreaterThan(trayZIndex)
  })

  it('keeps the generated service worker from importing hashed Workbox runtime files', () => {
    const viteConfig = readFileSync(join(REPO_ROOT, 'src', 'web', 'vite.config.ts'), 'utf-8')

    expect(viteConfig).toContain('inlineWorkboxRuntime: true')
  })

  it('keeps Stats and Collection sidebar parents collapsed with submenu children', () => {
    const app = readRepoFile('App.vue')

    expect(app).toContain("const statsExpanded = ref(false)")
    expect(app).toContain("const collectionExpanded = ref(false)")
    expect(app).toContain("id: 'collection'")
    expect(app).toContain("label: 'Gallery'")
    expect(app).toContain("label: 'Tray'")
    expect(app).toContain("id: 'stats'")
    expect(app).toContain("label: 'Timeline'")
    expect(app).toContain("label: 'Map'")
  })
})
