import { createApp, h, reactive } from 'vue'
import CoinViewer from './CoinViewer.vue'
import { McpAppBridge, applyHostContext, startAutoResize } from './bridge'
import type { CallToolResult } from './types'
import './coin-viewer.css'

const state = reactive({
  result: null as CallToolResult | null,
  pending: false,
  failed: false,
})

if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
  document.documentElement.dataset.theme = 'dark'
}

const bridge = new McpAppBridge(window.parent, {
  onToolInput: () => {
    state.pending = true
  },
  onToolResult: (result) => {
    state.pending = false
    state.result = result
  },
  onHostContext: (context) => applyHostContext(context),
})

createApp({
  render: () => h(CoinViewer, { bridge, result: state.result, pending: state.pending, failed: state.failed }),
}).mount('#app')

bridge
  .connect()
  .then(() => startAutoResize((width, height) => bridge.notifySize(width, height)))
  .catch(() => {
    state.failed = true
  })
