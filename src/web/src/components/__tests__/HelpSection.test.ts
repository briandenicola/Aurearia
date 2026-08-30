import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import HelpSection from '../HelpSection.vue'

const srcDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')

function readSource(relativePath: string): string {
  return fs.readFileSync(path.join(srcDir, relativePath), 'utf8')
}

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
}))

describe('HelpSection', () => {
  it('documents auction provider capability differences', () => {
    const wrapper = mount(HelpSection)

    expect(wrapper.text()).toContain('CNG Auctions supports richer hosted-auction sync')
    expect(wrapper.text()).toContain('NumisBids supports watchlist/import tracking only today')
    expect(wrapper.text()).toContain('manually update won/lost')
  })

  it('documents current app help for stats, capture, notifications, and external tools', () => {
    const wrapper = mount(HelpSection)
    const text = wrapper.text()

    expect(text).toContain('Stats Views')
    expect(text).toContain('Value snapshots, allocation, acquisition-year performance')
    expect(text).toContain('Quick Capture, Coin Lookup, and image upload flows start the camera only after you tap')
    expect(text).toContain('The notification badge tracks unread social, auction, wishlist, set milestone')
    expect(text).toContain('Header Name: X-API-Key')
    expect(text).toContain('Owned coins show purchase dates in YYYY-MM-DD')
    expect(text).toContain('Completed assistant answers can be reviewed as markdown')
    expect(text).toContain('Only approval creates the Agentic set')
  })

  // The help section's value is entirely in its instructions being followable. These pin the
  // navigation it names to the navigation that exists, so a settings-tab rename or a moved
  // entry point fails here instead of quietly sending users to the wrong screen.
  it('sends users to the settings tab that actually holds import and export', () => {
    const text = mount(HelpSection).text()
    const settingsPage = readSource('pages/SettingsPage.vue')
    const backups = readSource('components/settings/SettingsBackupsSection.vue')

    expect(settingsPage).toContain("{ id: 'backups', label: 'Backups' }")
    expect(backups).toContain('CSV Template')
    expect(text).toContain('Settings → Backups')
    expect(text).toContain('Import the CSV from Settings → Backups → Import')
    expect(text).not.toContain('Settings → Data → Import')
  })

  it('sends users to the settings tab that actually holds auction credentials', () => {
    const text = mount(HelpSection).text()
    const settingsPage = readSource('pages/SettingsPage.vue')
    const connections = readSource('components/settings/SettingsConnectionsSection.vue')

    expect(settingsPage).toContain("{ id: 'connections', label: 'Connections' }")
    // The Account section renders the provider blocks only when it owns them, which it does
    // not on the settings page — so the help must not point at Account for credentials.
    expect(settingsPage).toContain(':show-connections="false"')
    expect(connections).toContain('NumisBids Integration')
    expect(connections).toContain('CNG Auctions Integration')
    expect(connections).toContain('ParcelApp Integration')
    expect(text).toContain('credentials in Settings → Connections')
  })

  it('names entry points that exist in the sidebar', () => {
    const text = mount(HelpSection).text()
    const app = readSource('App.vue')

    // The agent is a sidebar item, not the retired Wish List → Find Coins button.
    expect(app).toContain("id: 'agent'")
    expect(text).toContain('Open the chat from Agent in the sidebar')
    expect(text).not.toContain('Find Coins')

    // Quick Capture and Coin Lookup are one merged entry point (see AppNavigation.test.ts).
    expect(app).not.toContain("to: '/quick-capture'")
    expect(text).toContain('Both intake flows live under Identify Coin')

    for (const view of ['Time Machine', 'Analysis History', 'Notes', 'Calendar', 'Showcases']) {
      expect(app).toContain(`label: '${view}'`)
      expect(text).toContain(view)
    }
  })

  it('lists every settings tab in its orientation table', () => {
    const text = mount(HelpSection).text()
    const settingsPage = readSource('pages/SettingsPage.vue')

    const tabLabels = [...settingsPage.matchAll(/\{ id: '[a-z]+', label: '([A-Za-z]+)' \}/g)]
      .map(match => match[1] as string)
    const documented = new Set(tabLabels.filter(label => label !== 'Admin'))

    expect(documented.size).toBeGreaterThan(5)
    for (const label of documented) {
      expect(text).toContain(label)
    }
  })

  it('keeps the CSV documentation in step with the importer and its template', () => {
    const text = mount(HelpSection).text()
    const backups = readSource('components/settings/SettingsBackupsSection.vue')

    for (const column of ['purchaseLocation', 'vendorSku', 'vendorInvoice', 'soldPrice', 'soldTo', 'rarityRating']) {
      expect(backups).toContain(column)
      expect(text).toContain(column)
    }

    // era is an admin-configured enum; free-text reign dates there do not match era filters,
    // so neither the template nor the help example may demonstrate that.
    expect(backups).not.toContain("'27 BC - 14 AD', 'Rome'")
    expect(text).not.toContain('Augustus,27 BC - 14 AD')
  })
})
