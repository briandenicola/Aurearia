import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CoinViewer from '../CoinViewer.vue'
import type { ViewerBridge } from '../bridge'
import type { CallToolResult, CoinSummary } from '../types'

const PNG = 'iVBORw0KGgo='

function page(coins: CoinSummary[], total = coins.length, query = ''): CallToolResult {
  const data = { coins, total, offset: 0, limit: 20, query }
  return { content: [{ type: 'text', text: JSON.stringify(data) }], structuredContent: data }
}

function imagesResult(coinIds: number[], perCoin = 1): CallToolResult {
  const refs = coinIds.flatMap((coinId) =>
    Array.from({ length: perCoin }, (_, index) => ({
      coinId, imageId: coinId * 10 + index, imageType: index === 0 ? 'obverse' : 'reverse', isPrimary: index === 0, mimeType: 'image/jpeg',
    })),
  )
  return {
    content: refs.map(() => ({ type: 'image', data: PNG, mimeType: 'image/jpeg' })),
    structuredContent: { images: refs, truncated: false },
  }
}

function coins(from: number, count: number, extra: Partial<CoinSummary> = {}): CoinSummary[] {
  return Array.from({ length: count }, (_, index) => ({ id: from + index, name: `Coin ${from + index}`, ...extra }))
}

interface FakeOptions {
  withPhotos?: number[]
  perCoin?: number
  matches?: CoinSummary[]
}

function fakeBridge({ withPhotos = [], perCoin = 1, matches = [] }: FakeOptions = {}) {
  const callTool = vi.fn(async (name: string, args: Record<string, unknown>): Promise<CallToolResult> => {
    if (name === 'get_coin_images') {
      return imagesResult((args.coinIds as number[]).filter((id) => withPhotos.includes(id)), perCoin)
    }
    if (name === 'get_coin') {
      return { structuredContent: { coin: { id: args.coinId, name: `Coin ${args.coinId}`, ruler: 'Trajan', weightGrams: 3.2 } } }
    }
    if (name === 'search_collection') {
      const text = String(args.query ?? '').toLowerCase()
      const hits = matches.filter((coin) => coin.name.toLowerCase().includes(text))
      const offset = Number(args.offset ?? 0)
      return page(hits.slice(offset, offset + Number(args.limit)), hits.length, String(args.query ?? ''))
    }
    throw new Error(`unexpected ${name}`)
  })
  return { callTool } satisfies ViewerBridge
}

describe('CoinViewer', () => {
  it('waits for a tool result before rendering', () => {
    const wrapper = mount(CoinViewer, { props: { bridge: fakeBridge(), result: null } })
    expect(wrapper.text()).toContain('Waiting for coin data')
  })

  it('renders a small result as cards with attribution, value, grade and material', async () => {
    const bridge = fakeBridge({ withPhotos: [1] })
    const wrapper = mount(CoinViewer, {
      props: {
        bridge,
        result: page([
          { id: 1, name: 'Denarius', ruler: 'Trajan', denomination: 'Denarius', era: 'ancient', mint: 'Rome', material: 'Silver', grade: 'VF', currentValue: 120, purchasePrice: 100 },
          { id: 2, name: 'AE3', purchasePrice: 15 },
        ]),
      },
    })
    await flushPromises()

    expect(wrapper.find('.count').text()).toBe('2 coins')
    const cards = wrapper.findAll('.card')
    expect(cards).toHaveLength(2)
    expect(cards[0]!.find('.attribution').text()).toBe('Trajan · Denarius')
    expect(cards[0]!.find('.context').text()).toBe('Ancient · Rome')
    expect(cards[0]!.find('.value').text()).toBe('Value $120')
    expect(cards[0]!.find('.grade').text()).toBe('VF')
    expect(cards[0]!.find('.material-dot').classes()).toContain('silver')
    expect(cards[1]!.find('.value').text()).toBe('Paid $15')
    expect(cards[0]!.find('img').attributes('src')).toBe(`data:image/jpeg;base64,${PNG}`)
    expect(cards[1]!.text()).toContain('No photo')
    expect(bridge.callTool).toHaveBeenCalledOnce()
    expect(bridge.callTool).toHaveBeenCalledWith('get_coin_images', { coinIds: [1, 2], size: 'thumb' })
  })

  it('switches to rows for long results and lets the user pick the layout', async () => {
    const wrapper = mount(CoinViewer, { props: { bridge: fakeBridge(), result: page(coins(1, 12)) } })
    await flushPromises()
    expect(wrapper.findAll('.row')).toHaveLength(12)
    expect(wrapper.find('.card').exists()).toBe(false)

    await wrapper.find('button[aria-label="Grid"]').trigger('click')
    expect(wrapper.findAll('.card')).toHaveLength(12)
    expect(wrapper.find('button[aria-label="Grid"]').attributes('aria-pressed')).toBe('true')
  })

  it('says how many coins matched and pages through the rest without the model', async () => {
    const all = coins(1, 45)
    const bridge = fakeBridge({ matches: all })
    const wrapper = mount(CoinViewer, { props: { bridge, result: page(all.slice(0, 20), 45, 'Coin') } })
    await flushPromises()
    expect(wrapper.find('.count').text()).toBe('Showing 20 of 45 coins')
    expect(wrapper.find('.more').text()).toBe('Show 20 more')

    await wrapper.find('.more').trigger('click')
    await flushPromises()
    expect(bridge.callTool).toHaveBeenCalledWith('search_collection', { query: 'Coin', limit: 20, offset: 20 })
    expect(wrapper.find('.count').text()).toBe('Showing 40 of 45 coins')
    expect(wrapper.find('.more').text()).toBe('Show 5 more')

    await wrapper.find('.more').trigger('click')
    await flushPromises()
    expect(wrapper.find('.count').text()).toBe('45 coins')
    expect(wrapper.find('.more').exists()).toBe(false)
    expect(wrapper.findAll('.row')).toHaveLength(45)
    const thumbBatches = bridge.callTool.mock.calls.filter(([name]) => name === 'get_coin_images')
    expect(thumbBatches.every(([, args]) => (args.coinIds as number[]).length <= 20)).toBe(true)
  })

  it('stops offering more when a page adds nothing new', async () => {
    const bridge = fakeBridge({ matches: coins(1, 3) })
    const wrapper = mount(CoinViewer, { props: { bridge, result: page(coins(1, 3), 50, 'Coin') } })
    await flushPromises()
    await wrapper.find('.more').trigger('click')
    await flushPromises()
    expect(wrapper.find('.more').exists()).toBe(false)
    expect(wrapper.find('.count').text()).toBe('3 coins')
  })

  it('filters inside the panel and reports an empty filter', async () => {
    const all = [...coins(1, 3), { id: 9, name: 'Solidus of Justinian' }]
    const bridge = fakeBridge({ matches: all })
    const wrapper = mount(CoinViewer, { props: { bridge, result: page(all) } })
    await flushPromises()

    await wrapper.find('input[type="search"]').setValue('solidus')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(bridge.callTool).toHaveBeenCalledWith('search_collection', { query: 'solidus', limit: 20, offset: 0 })
    expect(wrapper.findAll('.card')).toHaveLength(1)
    expect(wrapper.find('.name').text()).toBe('Solidus of Justinian')

    await wrapper.find('input[type="search"]').setValue('aureus')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('No coins matched “aureus”.')
    expect(wrapper.find('input[type="search"]').exists()).toBe(true)
  })

  it('opens a coin and returns to cached results without refetching thumbnails', async () => {
    const bridge = fakeBridge({ withPhotos: [1] })
    const wrapper = mount(CoinViewer, { props: { bridge, result: page([{ id: 1, name: 'Denarius' }]) } })
    await flushPromises()

    await wrapper.find('.card').trigger('click')
    await flushPromises()
    expect(wrapper.find('h1').text()).toBe('Coin 1')
    expect(wrapper.text()).toContain('3.2 g')
    expect(wrapper.findAll('.stage img')).toHaveLength(1)
    expect(bridge.callTool).toHaveBeenCalledWith('get_coin', { coinId: 1 })
    expect(bridge.callTool).toHaveBeenCalledWith('get_coin_images', { coinIds: [1], size: 'medium' })

    await wrapper.find('.back').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.card')).toHaveLength(1)
    const thumbCalls = bridge.callTool.mock.calls.filter(([, args]) => args.size === 'thumb')
    expect(thumbCalls).toHaveLength(1)
  })

  it('shows value change, grouped facts and switches between photos', async () => {
    const bridge = fakeBridge({ withPhotos: [4], perCoin: 2 })
    const wrapper = mount(CoinViewer, {
      props: {
        bridge,
        result: {
          structuredContent: {
            coin: {
              id: 4, name: 'Roman As', ruler: 'Tiberius', denomination: 'As', era: 'ancient', mint: 'Lugdunum',
              material: 'Bronze', grade: 'VF', diameterMm: 27, purchasePrice: 100, currentValue: 80,
            },
          },
        },
      },
    })
    await flushPromises()

    const figures = wrapper.findAll('.figure').map((node) => node.text())
    expect(figures).toEqual(['Value$80.00', 'Paid$100.00', 'Change−$20.00 (−20%)'])
    expect(wrapper.find('.figure-text.down').exists()).toBe(true)
    expect(wrapper.findAll('h2').map((node) => node.text())).toEqual(['Attribution', 'Physical'])
    expect(wrapper.findAll('dt').map((node) => node.text())).toEqual(['Ruler', 'Denomination', 'Mint', 'Era', 'Material', 'Diameter', 'Grade'])

    expect(wrapper.find('.stage img').attributes('alt')).toBe('Roman As obverse')
    await wrapper.find('button[aria-label="Show reverse"]').trigger('click')
    expect(wrapper.find('.stage img').attributes('alt')).toBe('Roman As reverse')
    expect(wrapper.find('button[aria-label="Show reverse"]').attributes('aria-pressed')).toBe('true')
  })

  it('shows a get_coin result as a detail view without a back button', async () => {
    const wrapper = mount(CoinViewer, {
      props: { bridge: fakeBridge(), result: { structuredContent: { coin: { id: 9, name: 'Solidus', missingFields: ['mint'] } } } },
    })
    await flushPromises()
    expect(wrapper.find('h1').text()).toBe('Solidus')
    expect(wrapper.find('.back').exists()).toBe(false)
    expect(wrapper.text()).toContain('No photos.')
    expect(wrapper.find('.missing').text()).toBe('Not recorded: mint')
  })

  it('shows readable missing fields, hides fields the server counts as missing, and formats prices', async () => {
    const wrapper = mount(CoinViewer, {
      props: {
        bridge: fakeBridge(),
        result: {
          structuredContent: {
            coin: {
              id: 15, name: 'eid mar', era: 'ancient', material: 'Other', category: 'Other', purchasePrice: 5,
              missingFields: ['material', 'weightGrams', 'vendorSku', 'referenceUrl', 'someNewField'],
            },
          },
        },
      },
    })
    await flushPromises()
    expect(wrapper.findAll('dt').map((node) => node.text())).toEqual(['Era', 'Category'])
    expect(wrapper.findAll('.figure').map((node) => node.text())).toEqual(['Paid$5.00'])
    expect(wrapper.find('.missing').text()).toBe('Not recorded: material, weight, vendor SKU, reference link, some new field')
  })

  it('reports empty results compactly and errors plainly', async () => {
    const empty = mount(CoinViewer, { props: { bridge: fakeBridge(), result: page([]) } })
    await flushPromises()
    expect(empty.text()).toBe('No coins matched.')

    const failed = mount(CoinViewer, { props: { bridge: fakeBridge(), result: { isError: true, content: [] } } })
    expect(failed.text()).toBe('The tool returned an error.')
  })

  it('falls back to the JSON text block from older servers and renders coin text as text', async () => {
    const wrapper = mount(CoinViewer, {
      props: {
        bridge: fakeBridge(),
        result: { content: [{ type: 'text', text: JSON.stringify({ coins: [{ id: 3, name: '<img src=x onerror=alert(1)>' }] }) }] },
      },
    })
    await flushPromises()
    expect(wrapper.find('.count').text()).toBe('1 coin')
    expect(wrapper.find('.more').exists()).toBe(false)
    expect(wrapper.find('.name').text()).toBe('<img src=x onerror=alert(1)>')
    expect(wrapper.find('.name img').exists()).toBe(false)
  })
})
