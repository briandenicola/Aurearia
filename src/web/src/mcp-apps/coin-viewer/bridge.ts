// View side of the MCP Apps postMessage protocol (io.modelcontextprotocol/ui,
// spec 2026-01-26). Kept dependency-free on purpose; its handshake matches the
// official @modelcontextprotocol/ext-apps App class and was exercised against
// that package's AppBridge host.
import type { CallToolResult, HostContext } from './types'

export const PROTOCOL_VERSION = '2026-01-26'

type JsonRpcId = number | string

interface JsonRpcMessage {
  jsonrpc?: string
  id?: JsonRpcId
  method?: string
  params?: unknown
  result?: unknown
  error?: { code: number; message?: string }
}

export interface ViewerBridge {
  callTool(name: string, args: Record<string, unknown>): Promise<CallToolResult>
}

export interface BridgeHandlers {
  onToolInput?: () => void
  onToolResult?: (result: CallToolResult) => void
  onHostContext?: (context: HostContext) => void
}

interface PendingRequest {
  resolve: (value: unknown) => void
  reject: (reason: Error) => void
  timer: ReturnType<typeof setTimeout>
}

export class McpAppBridge implements ViewerBridge {
  private nextId = 1
  private readonly pending = new Map<JsonRpcId, PendingRequest>()

  constructor(
    private readonly host: Pick<Window, 'postMessage'>,
    private readonly handlers: BridgeHandlers,
    private readonly timeoutMs = 30_000,
  ) {}

  // Accepts only messages from the host window; anything else is ignored.
  handleMessage = (event: Pick<MessageEvent, 'source' | 'data'>): void => {
    if (event.source !== this.host) return
    const message = event.data as JsonRpcMessage | null
    if (!message || message.jsonrpc !== '2.0') return

    if (message.id !== undefined && message.method === undefined) {
      const waiter = this.pending.get(message.id)
      if (!waiter) return
      this.pending.delete(message.id)
      clearTimeout(waiter.timer)
      if (message.error) waiter.reject(new Error(message.error.message || 'request failed'))
      else waiter.resolve(message.result)
      return
    }

    switch (message.method) {
      case 'ui/notifications/tool-input':
        this.handlers.onToolInput?.()
        break
      case 'ui/notifications/tool-result':
        this.handlers.onToolResult?.(message.params as CallToolResult)
        break
      case 'ui/notifications/host-context-changed':
        this.handlers.onHostContext?.(message.params as HostContext)
        break
      case 'ui/resource-teardown':
        if (message.id !== undefined) this.send({ jsonrpc: '2.0', id: message.id, result: {} })
        break
      default:
        if (message.id !== undefined) {
          this.send({ jsonrpc: '2.0', id: message.id, error: { code: -32601, message: 'Method not found' } })
        }
    }
  }

  async connect(listenOn: Pick<Window, 'addEventListener'> = window): Promise<void> {
    listenOn.addEventListener('message', this.handleMessage as (event: MessageEvent) => void)
    const result = (await this.request('ui/initialize', {
      protocolVersion: PROTOCOL_VERSION,
      appInfo: { name: 'aurearia-coin-viewer', version: '1.0.0' },
      appCapabilities: { availableDisplayModes: ['inline'] },
    })) as { hostContext?: HostContext } | undefined
    if (result?.hostContext) this.handlers.onHostContext?.(result.hostContext)
    this.notify('ui/notifications/initialized', {})
  }

  async callTool(name: string, args: Record<string, unknown>): Promise<CallToolResult> {
    const result = (await this.request('tools/call', { name, arguments: args })) as CallToolResult | undefined
    if (!result || result.isError) throw new Error(`${name} failed`)
    return result
  }

  notifySize(width: number, height: number): void {
    this.notify('ui/notifications/size-changed', { width, height })
  }

  private request(method: string, params: unknown): Promise<unknown> {
    const id = this.nextId++
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        if (this.pending.delete(id)) reject(new Error(`${method} timed out`))
      }, this.timeoutMs)
      this.pending.set(id, { resolve, reject, timer })
      this.send({ jsonrpc: '2.0', id, method, params })
    })
  }

  private notify(method: string, params: unknown): void {
    this.send({ jsonrpc: '2.0', method, params })
  }

  private send(message: JsonRpcMessage): void {
    this.host.postMessage(message, '*')
  }
}

// Reports the document's natural size to the host, measured the way the
// official SDK does: html is briefly set to max-content so the measurement is
// not pinned to the iframe's current height, which would otherwise stop the
// frame from shrinking or leave a scrollbar after images load.
export function startAutoResize(report: (width: number, height: number) => void): () => void {
  let scheduled = false
  let lastWidth = 0
  let lastHeight = 0
  const measure = () => {
    if (scheduled) return
    scheduled = true
    requestAnimationFrame(() => {
      scheduled = false
      const root = document.documentElement
      const previous = root.style.height
      root.style.height = 'max-content'
      const height = Math.ceil(root.getBoundingClientRect().height)
      root.style.height = previous
      const width = Math.ceil(window.innerWidth)
      if (width !== lastWidth || height !== lastHeight) {
        lastWidth = width
        lastHeight = height
        report(width, height)
      }
    })
  }
  measure()
  const observer = new ResizeObserver(measure)
  observer.observe(document.documentElement)
  observer.observe(document.body)
  return () => observer.disconnect()
}

export function applyHostContext(context: HostContext | undefined, root: HTMLElement = document.documentElement): void {
  if (!context) return
  if (context.theme === 'light' || context.theme === 'dark') root.dataset.theme = context.theme
  for (const [key, value] of Object.entries(context.styles?.variables ?? {})) {
    if (key.startsWith('--') && typeof value === 'string') root.style.setProperty(key, value)
  }
}
