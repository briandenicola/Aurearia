import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { join } from 'path'

const REPO_ROOT = join(__dirname, '..', '..', '..', '..')
const SRC_DIR = join(REPO_ROOT, 'src', 'web', 'src')
const COPILOT_INSTRUCTIONS = join(REPO_ROOT, '.github', 'copilot-instructions.md')

function readRepoFile(pathFromSrc: string): string {
  return readFileSync(join(SRC_DIR, pathFromSrc), 'utf-8')
}

function extractCssBlock(content: string, selector: string): string {
  const start = content.indexOf(`${selector} {`)
  if (start === -1) return ''
  const bodyStart = content.indexOf('{', start)
  const bodyEnd = content.indexOf('}', bodyStart)
  return content.slice(bodyStart + 1, bodyEnd)
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
    const drawerNavigation = extractCssBlock(trayControls, '.drawer-navigation')
    const drawerLabel = extractCssBlock(trayControls, '.drawer-label')

    expect(trayControls).toContain('Prev')
    expect(trayControls).toContain('Tray {{ drawerIndex + 1 }} of {{ totalDrawers }}')
    expect(trayControls).toContain('Next')
    expect(trayControls).toContain('nav-btn')
    expect(trayControls).toContain('position: fixed')
    expect(trayControls).toContain('bottom: calc(1rem + env(safe-area-inset-bottom))')
    expect(drawerNavigation).toContain('flex-wrap: nowrap')
    expect(drawerNavigation).not.toContain('flex-direction: column')
    expect(drawerLabel).not.toContain('order: -1')
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

    expect(timelinePage).toContain('grid-template-columns: minmax(0, 1fr)')
    expect(timelinePage).toContain('min-width: 0')
    expect(timelinePage).toContain('overflow: hidden')
    expect(setDetailPage).toContain('set-coin-action-btn')
    expect(setDetailPage).toContain('<ChevronUp :size="16" />')
    expect(setDetailPage).toContain('<ChevronDown :size="16" />')
    expect(setDetailPage).toContain('<X :size="16" />')
    expect(setDetailPage).toContain('flex-wrap: nowrap')
    expect(setDetailPage).not.toContain('>Up<')
    expect(setDetailPage).not.toContain('>Down<')
    expect(setDetailPage).not.toContain('>Remove<')
  })

  it('keeps Sets list cards refined and count-forward', () => {
    const setCard = readRepoFile(join('components', 'sets', 'SetDashboardCard.vue'))

    expect(setCard).toContain('Curated group')
    expect(setCard).toContain('min-height: 5rem')
    expect(setCard).toContain('height: 4rem')
    expect(setCard).toContain('align-items: flex-end')
    expect(setCard).toContain('font-size: 2.75rem')
    expect(setCard).toContain('border-radius: var(--radius-md)')
    expect(setCard).not.toContain('completion-meter')
    expect(setCard).not.toContain('Completion set')
  })

  it('keeps collection coin images from over-zooming or clipping', () => {
    const coinCard = readRepoFile('components/CoinCard.vue')
    const swipeGallery = readRepoFile('components/SwipeGallery.vue')

    expect(coinCard).toContain('object-fit: contain')
    expect(coinCard).toContain('transform: scale(1.02)')
    expect(coinCard).not.toContain('object-fit: cover')
    expect(swipeGallery).toContain('object-fit: contain')
    expect(swipeGallery).toContain('transform: scale(1.05)')
    expect(swipeGallery).not.toContain('transform: scale(1.28)')
  })

  it('keeps the PWA agent button viewport-fixed globally', () => {
    const app = readRepoFile('App.vue')
    const mainCss = readRepoFile(join('assets', 'styles', 'main.css'))
    const agentFabCss = extractCssBlock(mainCss, '.agent-fab')

    expect(app).toContain('<Teleport to="body">')
    expect(app).toContain('class="agent-fab"')
    expect(app).not.toContain('.agent-fab {')
    expect(agentFabCss).toContain('position: fixed')
    expect(agentFabCss).toContain('bottom: calc(24px + env(safe-area-inset-bottom))')
    expect(agentFabCss).toContain('right: calc(24px + env(safe-area-inset-right))')
  })

  it('keeps the agent chat overlay above tray pagination controls', () => {
    const chat = readRepoFile(join('components', 'CoinSearchChat.vue'))
    const trayControls = readRepoFile(join('components', 'tray', 'TrayControls.vue'))
    const chatOverlayCss = extractCssBlock(chat, '.chat-overlay')
    const trayControlsCss = extractCssBlock(trayControls, '.tray-controls')
    const chatZIndex = Number(chatOverlayCss.match(/z-index:\s*(\d+)/)?.[1] ?? 0)
    const trayZIndex = Number(trayControlsCss.match(/z-index:\s*(\d+)/)?.[1] ?? 0)

    expect(chatOverlayCss).toContain('position: fixed')
    expect(trayControlsCss).toContain('position: fixed')
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
