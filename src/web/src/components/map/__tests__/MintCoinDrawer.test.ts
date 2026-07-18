import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { readFileSync } from 'fs'
import { fileURLToPath } from 'url'
import { dirname, resolve } from 'path'
import MintCoinDrawer from '@/components/map/MintCoinDrawer.vue'
import { groupCoinsByMint } from '@/utils/mintMap'
import { buildMintMapFixtureCoins, buildTestMintLocations } from '@/test/fixtures/coins'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('MintCoinDrawer', () => {
  const mintLocations = buildTestMintLocations()

  it('lists only the selected mint group coins', () => {
    const group = groupCoinsByMint(buildMintMapFixtureCoins(), mintLocations).matched.find((item) => item.mint.id === 1) ?? null
    const wrapper = mount(MintCoinDrawer, {
      props: { open: true, group },
      global: { stubs: { RouterLink: routerLinkStub } },
    })

    expect(wrapper.text()).toContain('Trajan Denarius Core')
    expect(wrapper.text()).toContain('Roma Alias Denarius')
    expect(wrapper.text()).not.toContain('Byzantium Alias Solidus')
  })

  it('renders drawer with z-index 1100 to stack above Leaflet map controls (z-index 1000)', () => {
    const group = groupCoinsByMint(buildMintMapFixtureCoins(), mintLocations).matched.find((item) => item.mint.id === 1) ?? null
    const wrapper = mount(MintCoinDrawer, {
      props: { open: true, group },
      global: { stubs: { RouterLink: routerLinkStub } },
    })

    // The drawer no longer has a dedicated `.mint-drawer` class; it's the
    // dialog <aside> with the z-index baked into a Tailwind arbitrary class.
    const drawer = wrapper.find('[role="dialog"]')
    expect(drawer.exists()).toBe(true)
    expect(drawer.classes()).toContain('z-[1100]')

    // Source-level assertion: verify component contains the critical z-index fix
    const componentPath = resolve(__dirname, '..', 'MintCoinDrawer.vue')
    const componentSource = readFileSync(componentPath, 'utf-8')
    expect(componentSource).toContain('z-[1100]')
  })
})
