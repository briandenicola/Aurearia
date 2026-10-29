import { describe, expect, it, vi } from 'vitest'
import { McpAppBridge, PROTOCOL_VERSION, applyHostContext } from '../bridge'

function setup(handlers = {}) {
  const host = { postMessage: vi.fn() }
  let listener: ((event: MessageEvent) => void) | undefined
  const listenOn = {
    addEventListener: vi.fn((_type: string, fn: (event: MessageEvent) => void) => {
      listener = fn
    }),
  }
  const bridge = new McpAppBridge(host, handlers, 1_000)
  const deliver = (data: unknown, source: unknown = host) =>
    listener?.({ source, data } as unknown as MessageEvent)
  const sent = () => host.postMessage.mock.calls.map((call) => call[0])
  return { bridge, host, listenOn, deliver, sent }
}

async function connected(handlers = {}) {
  const context = setup(handlers)
  const connecting = context.bridge.connect(context.listenOn as unknown as Window)
  const init = context.sent()[0]
  context.deliver({ jsonrpc: '2.0', id: init.id, result: { protocolVersion: PROTOCOL_VERSION, hostContext: { theme: 'dark' } } })
  await connecting
  return context
}

describe('McpAppBridge', () => {
  it('initializes with the spec fields and confirms after the host answers', async () => {
    const onHostContext = vi.fn()
    const { sent } = await connected({ onHostContext })
    const [init, initialized] = sent()
    expect(init).toMatchObject({
      jsonrpc: '2.0',
      method: 'ui/initialize',
      params: {
        protocolVersion: '2026-01-26',
        appInfo: { name: 'aurearia-coin-viewer' },
        appCapabilities: { availableDisplayModes: ['inline'] },
      },
    })
    expect(initialized).toEqual({ jsonrpc: '2.0', method: 'ui/notifications/initialized', params: {} })
    expect(onHostContext).toHaveBeenCalledWith({ theme: 'dark' })
  })

  it('ignores messages that do not come from the host window', async () => {
    const onToolResult = vi.fn()
    const { deliver } = await connected({ onToolResult })
    deliver({ jsonrpc: '2.0', method: 'ui/notifications/tool-result', params: {} }, { postMessage: vi.fn() })
    expect(onToolResult).not.toHaveBeenCalled()
  })

  it('dispatches tool input and tool results', async () => {
    const onToolInput = vi.fn()
    const onToolResult = vi.fn()
    const { deliver } = await connected({ onToolInput, onToolResult })
    deliver({ jsonrpc: '2.0', method: 'ui/notifications/tool-input', params: { arguments: {} } })
    deliver({ jsonrpc: '2.0', method: 'ui/notifications/tool-result', params: { structuredContent: { coins: [] } } })
    expect(onToolInput).toHaveBeenCalledOnce()
    expect(onToolResult).toHaveBeenCalledWith({ structuredContent: { coins: [] } })
  })

  it('calls server tools through the host and surfaces failures', async () => {
    const { bridge, deliver, sent } = await connected()

    const ok = bridge.callTool('get_coin', { coinId: 5 })
    const okRequest = sent().at(-1)
    expect(okRequest).toMatchObject({ method: 'tools/call', params: { name: 'get_coin', arguments: { coinId: 5 } } })
    deliver({ jsonrpc: '2.0', id: okRequest.id, result: { structuredContent: { coin: { id: 5, name: 'x' } } } })
    await expect(ok).resolves.toEqual({ structuredContent: { coin: { id: 5, name: 'x' } } })

    const toolError = bridge.callTool('get_coin', { coinId: 6 })
    deliver({ jsonrpc: '2.0', id: sent().at(-1).id, result: { isError: true, content: [] } })
    await expect(toolError).rejects.toThrow('get_coin failed')

    const rpcError = bridge.callTool('get_coin', { coinId: 7 })
    deliver({ jsonrpc: '2.0', id: sent().at(-1).id, error: { code: -32000, message: 'denied' } })
    await expect(rpcError).rejects.toThrow('denied')
  })

  it('answers teardown and rejects unknown host requests', async () => {
    const { deliver, sent } = await connected()
    deliver({ jsonrpc: '2.0', id: 'td', method: 'ui/resource-teardown', params: { reason: 'closed' } })
    deliver({ jsonrpc: '2.0', id: 'x', method: 'ui/unknown', params: {} })
    expect(sent().slice(-2)).toEqual([
      { jsonrpc: '2.0', id: 'td', result: {} },
      { jsonrpc: '2.0', id: 'x', error: { code: -32601, message: 'Method not found' } },
    ])
  })
})

describe('applyHostContext', () => {
  it('applies the theme and only CSS custom properties', () => {
    const root = document.createElement('div')
    applyHostContext({ theme: 'light', styles: { variables: { '--color-text-primary': '#111', color: 'red' } } }, root)
    expect(root.dataset.theme).toBe('light')
    expect(root.style.getPropertyValue('--color-text-primary')).toBe('#111')
    expect(root.style.getPropertyValue('color')).toBe('')
  })
})
