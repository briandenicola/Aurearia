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

    expect(trayControls).toContain('&lt; Previous')
    expect(trayControls).toContain('Drawer {{ drawerIndex + 1 }} of {{ totalDrawers }}')
    expect(trayControls).toContain('Next &gt;')
    expect(drawerNavigation).toContain('flex-wrap: nowrap')
    expect(drawerNavigation).not.toContain('flex-direction: column')
    expect(drawerLabel).not.toContain('order: -1')
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
