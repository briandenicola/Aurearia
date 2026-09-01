import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const appPath = path.resolve(__dirname, '../App.vue')

const mockIsPwa = vi.hoisted(() => ({ value: false }))
const mockGetSets = vi.hoisted(() => vi.fn())

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: mockIsPwa.value }),
}))

vi.mock('sortablejs', () => ({
  default: { create: vi.fn(() => ({ destroy: vi.fn(), toArray: () => [] })) },
}))

vi.mock('virtual:pwa-register', () => ({
  registerSW: () => vi.fn(async () => undefined),
}))

// Public-dir asset referenced by App.vue's brand <img>; the Vitest module
// runner cannot resolve root-absolute public assets.
vi.mock('/coin-logo.jpg', () => ({ default: '/coin-logo.jpg' }))

vi.mock('@/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/client')>()
  return {
    ...actual,
    getSets: (...args: unknown[]) => mockGetSets(...args),
    getMe: vi.fn(async () => ({ data: { id: 1, emailMissing: false, createdAt: '2020-01-01T00:00:00Z' } })),
    getUnreadNotificationCount: vi.fn(async () => ({ data: { count: 0 } })),
    updateProfile: vi.fn(async () => ({ data: {} })),
  }
})

describe('App sidebar navigation', () => {
  it('renders Stats as a collapsible parent with dedicated route children', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    expect(source).toContain("label: 'Stats'")
    expect(source).toContain("label: 'Timeline', to: '/stats/timeline'")
    expect(source).toContain("label: 'Map', to: '/stats/mint-map'")
    expect(source).toContain("label: 'Health', to: '/stats/health'")
    expect(source).toContain("label: 'Value Details', to: '/stats/value-trends'")
    expect(source).not.toContain("id: 'stats-emperors'")
    expect(source).not.toContain("to: '/stats/emperors'")
    expect(source).not.toContain("id: 'timeline'")
    expect(source).not.toContain("label: 'Collection Distribution'")
    expect(source).not.toContain('#collection-health')
    expect(source).not.toContain('#value-over-time')
  })

  it('renders Sets as a collapsible parent with Emperors gated under it', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    expect(source).toContain("label: 'Sets'")
    expect(source).toContain("label: 'My Sets', to: '/sets'")
    expect(source).toContain("label: 'Emperors', to: '/sets/emperors'")
    expect(source).toContain("child.id !== 'sets-emperors' || auth.user?.emperorTrackerEnabled")
  })

  it('uses Identify Coin as the single merged quick capture entry point', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    expect(source).not.toContain("id: 'quick-capture'")
    expect(source).not.toContain("label: 'Quick Capture'")
    expect(source).not.toContain("to: '/quick-capture'")
    expect(source).toContain("id: 'lookup'")
    expect(source).toContain("label: 'Identify Coin'")
    expect(source).toContain("to: '/lookup'")
    expect(source).toContain("id: 'add-coin'")
    expect(source).toContain("label: 'Add Coin'")
  })

  it('preserves AI intake identification navigation without adding Quick Capture AI expansion', () => {
    const source = fs.readFileSync(appPath, 'utf8')
    const routerSource = fs.readFileSync(path.resolve(__dirname, '../router/index.ts'), 'utf8')

    expect(source).toContain("id: 'lookup'")
    expect(source).toContain("label: 'Identify Coin'")
    expect(source).toContain("to: '/lookup'")
    expect(routerSource).toContain("path: '/lookup'")
    expect(routerSource).not.toContain('/quick-capture/intake')
    expect(routerSource).not.toContain('/quick-capture/ai')
  })

  it('keeps shipment tracker under coin-scoped routes, not top-level sidebar navigation', () => {
    const source = fs.readFileSync(appPath, 'utf8')
    const routerSource = fs.readFileSync(path.resolve(__dirname, '../router/index.ts'), 'utf8')

    expect(source).not.toContain("id: 'shipments'")
    expect(routerSource).toContain("path: '/coin/:id/shipment'")
  })

  it('merges pinned sets into the existing Sets children branch without a second computed', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    // The Emperors gate must survive verbatim (design D5 / risk R6) and the
    // pinned entries must be appended inside that same map branch.
    expect(source).toContain("child.id !== 'sets-emperors' || auth.user?.emperorTrackerEnabled")
    expect(source).toContain('id: `sets-pinned-${s.id}`')
    expect(source).toContain('to: `/sets/${s.id}`')
    expect(source).toContain('const setsExpanded = ref(false)')
    // Exactly one place rewrites the Sets children.
    expect(source.match(/item\.id !== 'sets'/g) ?? []).toHaveLength(1)
  })
})

// Behavioural coverage for pinned sets in the Sets submenu. These mount the
// real App.vue with the real usePinnedSets singleton so the assertions prove
// rendered DOM, not source strings.
describe('App sidebar pinned sets', () => {
  let router: Router
  let wrapper: VueWrapper | null = null

  function pinnedSetPayload(sets: Array<Record<string, unknown>>) {
    return { data: { sets } }
  }

  function buildSet(overrides: Record<string, unknown> = {}) {
    return {
      id: 1,
      name: 'Pinned Set',
      color: '#c9a84c',
      setType: 'standard',
      coinCount: 1,
      totalValue: 10,
      pinned: true,
      pinnedAt: '2026-01-01T00:00:00Z',
      ...overrides,
    }
  }

  async function mountApp() {
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'collection', component: { template: '<div />' } },
        { path: '/sets', name: 'sets', component: { template: '<div />' } },
        { path: '/sets/emperors', name: 'emperors', component: { template: '<div />' } },
        { path: '/sets/:id', name: 'set-detail', component: { template: '<div />' } },
        { path: '/login', name: 'login', component: { template: '<div />' } },
        { path: '/:pathMatch(.*)*', name: 'catch-all', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const App = (await import('../App.vue')).default
    wrapper = mount(App, {
      attachTo: document.body,
      global: {
        plugins: [router],
        stubs: {
          CoinSearchChat: true,
          AppDialog: true,
          AppToasts: true,
          PwaInstallPrompt: true,
          PwaUpdateBanner: true,
        },
      },
    })
    await flushPromises()
    return wrapper
  }

  async function openSetsSubmenu() {
    // Open the drawer via the brand button, then expand the Sets parent.
    await wrapper!.find('nav button').trigger('click')
    await flushPromises()
    const setsParent = wrapper!.findAll('.sidebar-link').find((el) => el.text() === 'Sets')
    expect(setsParent, 'Sets parent nav item should render').toBeTruthy()
    await setsParent!.trigger('click')
    await flushPromises()
    return setsParent!
  }

  function setsSubmenuLinks() {
    const submenu = wrapper!.find('[aria-label="Sets views"]')
    expect(submenu.exists(), 'Sets submenu should be expanded').toBe(true)
    return submenu.findAll('a')
  }

  beforeEach(async () => {
    vi.clearAllMocks()
    localStorage.clear()
    localStorage.setItem('token', 'test-token')
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'tester', role: 'user', emperorTrackerEnabled: false }))
    setActivePinia(createPinia())
    const { usePinnedSets } = await import('@/composables/usePinnedSets')
    usePinnedSets().clear()
    mockIsPwa.value = false
    mockGetSets.mockResolvedValue(pinnedSetPayload([]))
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    document.body.innerHTML = ''
  })

  it('appends pinned sets after the static children and links them to /sets/:id', async () => {
    mockGetSets.mockResolvedValue(pinnedSetPayload([
      buildSet({ id: 41, name: 'Later Pin', pinnedAt: '2026-02-01T00:00:00Z' }),
      buildSet({ id: 40, name: 'Earlier Pin', pinnedAt: '2026-01-01T00:00:00Z' }),
      buildSet({ id: 99, name: 'Not Pinned', pinned: false, pinnedAt: null }),
    ]))

    await mountApp()
    await openSetsSubmenu()

    const links = setsSubmenuLinks()
    expect(links.map((l) => l.text())).toEqual(['My Sets', 'Earlier Pin', 'Later Pin'])
    expect(links.map((l) => l.attributes('href'))).toEqual(['/sets', '/sets/40', '/sets/41'])
  })

  it('keeps pinned sets after Emperors when the Emperor Tracker is enabled', async () => {
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'tester', role: 'user', emperorTrackerEnabled: true }))
    setActivePinia(createPinia())
    mockGetSets.mockResolvedValue(pinnedSetPayload([buildSet({ id: 40, name: 'Earlier Pin' })]))

    await mountApp()
    await openSetsSubmenu()

    const links = setsSubmenuLinks()
    expect(links.map((l) => l.text())).toEqual(['My Sets', 'Emperors', 'Earlier Pin'])
    expect(links.map((l) => l.attributes('href'))).toEqual(['/sets', '/sets/emperors', '/sets/40'])
  })

  it('renders the submenu unchanged when no sets are pinned', async () => {
    mockGetSets.mockResolvedValue(pinnedSetPayload([buildSet({ id: 7, pinned: false, pinnedAt: null })]))

    await mountApp()
    await openSetsSubmenu()

    const links = setsSubmenuLinks()
    expect(links).toHaveLength(1)
    expect(links[0]?.text()).toBe('My Sets')
    expect(wrapper!.html()).not.toContain('sets-pinned-')
  })

  it('degrades to the static children when GET /sets fails', async () => {
    mockGetSets.mockRejectedValue(new Error('network down'))

    await mountApp()
    await openSetsSubmenu()

    expect(setsSubmenuLinks().map((l) => l.text())).toEqual(['My Sets'])
  })

  it('truncates long pinned labels and exposes the full name via title', async () => {
    const longName = 'Purchase Records For The Twelve Caesars And Later Julio-Claudian Denarii'
    mockGetSets.mockResolvedValue(pinnedSetPayload([buildSet({ id: 55, name: longName })]))

    await mountApp()
    await openSetsSubmenu()

    const pinnedLink = setsSubmenuLinks()[1]!
    expect(pinnedLink.classes()).toContain('min-w-0')
    const label = pinnedLink.find('span')
    expect(label.classes()).toContain('truncate')
    expect(label.attributes('title')).toBe(longName)
    // Static children keep the same treatment, so nothing regressed for them.
    const staticLabel = setsSubmenuLinks()[0]!.find('span')
    expect(staticLabel.classes()).toContain('truncate')
    expect(staticLabel.attributes('title')).toBe('My Sets')
  })

  it('closes the PWA drawer and navigates when a pinned entry is tapped', async () => {
    mockIsPwa.value = true
    mockGetSets.mockResolvedValue(pinnedSetPayload([buildSet({ id: 40, name: 'Earlier Pin' })]))

    await mountApp()
    await openSetsSubmenu()

    expect(wrapper!.find('aside').exists()).toBe(true)
    await setsSubmenuLinks()[1]!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/sets/40')
    expect(router.currentRoute.value.name).toBe('set-detail')
    expect(wrapper!.find('aside').exists()).toBe(false)
  })

  it('keeps the Sets parent collapsed on load even when sets are pinned', async () => {
    mockGetSets.mockResolvedValue(pinnedSetPayload([buildSet({ id: 40, name: 'Earlier Pin' })]))

    await mountApp()
    await wrapper!.find('nav button').trigger('click')
    await flushPromises()

    expect(wrapper!.find('aside').exists()).toBe(true)
    expect(wrapper!.find('[aria-label="Sets views"]').exists()).toBe(false)
    expect(wrapper!.html()).not.toContain('Earlier Pin')
  })

  it('drops an unpinned entry from the sidebar without a reload', async () => {
    mockGetSets.mockResolvedValue(pinnedSetPayload([
      buildSet({ id: 40, name: 'Earlier Pin' }),
      buildSet({ id: 41, name: 'Later Pin', pinnedAt: '2026-02-01T00:00:00Z' }),
    ]))

    await mountApp()
    await openSetsSubmenu()
    expect(setsSubmenuLinks()).toHaveLength(3)

    const { usePinnedSets } = await import('@/composables/usePinnedSets')
    mockGetSets.mockResolvedValue(pinnedSetPayload([buildSet({ id: 41, name: 'Later Pin', pinnedAt: '2026-02-01T00:00:00Z' })]))
    await usePinnedSets().refresh()
    await flushPromises()

    expect(setsSubmenuLinks().map((l) => l.text())).toEqual(['My Sets', 'Later Pin'])
  })

  it('clears pinned sets on logout so the next user sees none', async () => {
    mockGetSets.mockResolvedValue(pinnedSetPayload([buildSet({ id: 40, name: 'Earlier Pin' })]))

    await mountApp()
    await openSetsSubmenu()
    expect(setsSubmenuLinks()).toHaveLength(2)

    const logoutButton = wrapper!.findAll('.sidebar-link').find((el) => el.text() === 'Logout')
    expect(logoutButton, 'Logout button should render').toBeTruthy()
    await logoutButton!.trigger('click')
    await flushPromises()

    const { usePinnedSets } = await import('@/composables/usePinnedSets')
    expect(usePinnedSets().pinnedSets.value).toEqual([])
  })
})
