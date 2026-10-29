import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CoinViewer from '../CoinViewer.vue'
import type { ViewerBridge } from '../bridge'
import type { CallToolResult, CoinSummary } from '../types'

const PNG = 'iVBORw0KGgo='

function listResult(coins: CoinSummary[]): CallToolResult {
  return { content: [{ type: 'text', text: JSON.stringify({ coins }) }], structuredContent: { coins } }
}

function imagesResult(coinIds: number[]): CallToolResult {
  return {
    content: coinIds.map(() => ({ type: 'image', data: PNG, mimeType: 'image/jpeg' })),
    structuredContent: {
      images: coinIds.map((coinId, index) => ({
        coinId, imageId: 100 + index, imageType: 'obverse', isPrimary: true, mimeType: 'image/jpeg',
      })),
      truncated: false,
    },
  }
}

function fakeBridge(withPhotos: number[] = []) {
  const callTool = vi.fn(async (name: string, args: Record<string, unknown>): Promise<CallToolResult> => {
    if (name === 'get_coin_images') {
      const ids = (args.coinIds as number[]).filter((id) => withPhotos.includes(id))
      return imagesResult(ids)
    }
    if (name === 'get_coin') {
      return { structuredContent: { coin: { id: args.coinId, name: `Coin ${args.coinId}`, ruler: 'Trajan', weightGrams: 3.2 } } }
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

  it('renders search results with thumbnails fetched in one call', async () => {
    const bridge = fakeBridge([1])
    const wrapper = mount(CoinViewer, {
      props: { bridge, result: listResult([{ id: 1, name: 'Denarius', ruler: 'Trajan' }, { id: 2, name: 'AE3' }]) },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('2 coins')
    expect(wrapper.findAll('.card')).toHaveLength(2)
    expect(wrapper.find('.card img').attributes('src')).toBe(`data:image/jpeg;base64,${PNG}`)
    expect(wrapper.text()).toContain('No photo')
    expect(bridge.callTool).toHaveBeenCalledOnce()
    expect(bridge.callTool).toHaveBeenCalledWith('get_coin_images', { coinIds: [1, 2], size: 'thumb' })
  })

  it('opens a coin and returns to cached results without refetching thumbnails', async () => {
    const bridge = fakeBridge([1])
    const wrapper = mount(CoinViewer, { props: { bridge, result: listResult([{ id: 1, name: 'Denarius' }]) } })
    await flushPromises()

    await wrapper.find('.card').trigger('click')
    await flushPromises()
    expect(wrapper.find('h1').text()).toBe('Coin 1')
    expect(wrapper.text()).toContain('3.2 g')
    expect(wrapper.findAll('.gallery img')).toHaveLength(1)
    expect(bridge.callTool).toHaveBeenCalledWith('get_coin', { coinId: 1 })
    expect(bridge.callTool).toHaveBeenCalledWith('get_coin_images', { coinIds: [1], size: 'medium' })

    await wrapper.find('.back').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.card')).toHaveLength(1)
    const thumbCalls = bridge.callTool.mock.calls.filter(([, args]) => args.size === 'thumb')
    expect(thumbCalls).toHaveLength(1)
  })

  it('shows a get_coin result as a detail view without a back button', async () => {
    const bridge = fakeBridge()
    const wrapper = mount(CoinViewer, {
      props: { bridge, result: { structuredContent: { coin: { id: 9, name: 'Solidus', missingFields: ['mint'] } } } },
    })
    await flushPromises()
    expect(wrapper.find('h1').text()).toBe('Solidus')
    expect(wrapper.find('.back').exists()).toBe(false)
    expect(wrapper.text()).toContain('No photos.')
    expect(wrapper.text()).toContain('Missing: mint')
  })

  it('shows readable missing fields, hides rows the server counts as missing, and formats prices', async () => {
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
    const labels = wrapper.findAll('dt').map((node) => node.text())
    expect(labels).toEqual(['Era', 'Category', 'Purchase price'])
    expect(wrapper.findAll('dd').map((node) => node.text())).toEqual(['ancient', 'Other', '$5.00'])
    expect(wrapper.find('.missing').text()).toBe('Missing: material, weight, vendor SKU, reference link, some new field')
  })

  it('reports empty results compactly and errors plainly', async () => {
    const empty = mount(CoinViewer, { props: { bridge: fakeBridge(), result: listResult([]) } })
    await flushPromises()
    expect(empty.text()).toBe('No coins matched.')

    const failed = mount(CoinViewer, { props: { bridge: fakeBridge(), result: { isError: true, content: [] } } })
    expect(failed.text()).toBe('The tool returned an error.')
  })

  it('falls back to the JSON text block and renders coin text as text', async () => {
    const wrapper = mount(CoinViewer, {
      props: {
        bridge: fakeBridge(),
        result: { content: [{ type: 'text', text: JSON.stringify({ coins: [{ id: 3, name: '<img src=x onerror=alert(1)>' }] }) }] },
      },
    })
    await flushPromises()
    expect(wrapper.find('.name').text()).toBe('<img src=x onerror=alert(1)>')
    expect(wrapper.find('.name img').exists()).toBe(false)
  })
})
