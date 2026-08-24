import type { Config } from '@imgly/background-removal'

// Shared between the main thread (src/utils/backgroundRemoval.ts, the worker
// client) and the worker itself (backgroundRemovalWorker.ts). Must stay
// worker-safe: no `window`/DOM APIs, only globals available in both a
// Window and a WorkerGlobalScope (`location`, `URL`).
export const BACKGROUND_REMOVAL_ASSET_PATH = '/imgly-background-removal/'

export function backgroundRemovalPublicPath(): string {
  return new URL(BACKGROUND_REMOVAL_ASSET_PATH, location.origin).href
}

export const backgroundRemovalConfig: Config = {
  publicPath: backgroundRemovalPublicPath(),
  model: 'isnet_quint8',
  device: 'cpu',
  proxyToWorker: false,
  output: { format: 'image/png', quality: 1 },
}
