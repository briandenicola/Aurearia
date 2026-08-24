import type { ImageSource } from '@imgly/background-removal'
import {
  BACKGROUND_REMOVAL_ASSET_PATH,
  backgroundRemovalConfig,
  backgroundRemovalPublicPath,
} from '@/workers/backgroundRemovalConfig'
import type {
  BackgroundRemovalWorkerRequest,
  BackgroundRemovalWorkerResponse,
} from '@/workers/backgroundRemovalWorker'

// Re-exported for back-compat: other modules/tests import the shared config
// from here rather than reaching into src/workers directly.
export { BACKGROUND_REMOVAL_ASSET_PATH, backgroundRemovalConfig, backgroundRemovalPublicPath }

export type BackgroundRemovalErrorStage = 'library' | 'worker-error' | 'message-error' | 'terminated'

export class BackgroundRemovalError extends Error {
  readonly stage: BackgroundRemovalErrorStage

  constructor(stage: BackgroundRemovalErrorStage, message: string, options?: ErrorOptions) {
    super(message, options)
    this.name = 'BackgroundRemovalError'
    this.stage = stage
  }
}

interface PendingRequest {
  resolve: (result: Blob) => void
  reject: (error: Error) => void
}

// Lazily-created page-level singleton: initializing the worker re-downloads
// and re-initializes the ONNX model (tens of seconds), so it is reused across
// both call sites (ImageLightbox.vue, useImageProcessor.ts) rather than
// recreated per component instance.
let worker: Worker | null = null
let requestCounter = 0
const pending = new Map<string, PendingRequest>()

function createWorker(): Worker {
  const instance = new Worker(new URL('../workers/backgroundRemovalWorker.ts', import.meta.url), {
    type: 'module',
  })

  instance.addEventListener('message', (event: MessageEvent<BackgroundRemovalWorkerResponse>) => {
    const data = event.data
    const request = pending.get(data.id)
    if (!request) return
    pending.delete(data.id)
    if (data.ok) {
      request.resolve(data.result)
    } else {
      request.reject(new BackgroundRemovalError('library', data.error.message))
    }
  })

  // A worker-level `error` (e.g. a script failing to load/execute) can't be
  // attributed to a single in-flight request id, so reject everything
  // outstanding rather than let callers hang silently.
  instance.addEventListener('error', (event: ErrorEvent) => {
    rejectAllPending(
      new BackgroundRemovalError('worker-error', event.message || 'Background removal worker crashed'),
    )
  })

  instance.addEventListener('messageerror', () => {
    rejectAllPending(
      new BackgroundRemovalError('message-error', 'Background removal worker returned an unreadable message'),
    )
  })

  return instance
}

function rejectAllPending(error: BackgroundRemovalError) {
  for (const [id, request] of pending) {
    pending.delete(id)
    request.reject(error)
  }
}

function getWorker(): Worker {
  if (!worker) {
    worker = createWorker()
  }
  return worker
}

export function removeCoinBackground(image: ImageSource): Promise<Blob> {
  if (!(image instanceof Blob)) {
    return Promise.reject(
      new BackgroundRemovalError('library', 'Background removal worker requires a Blob input'),
    )
  }

  const id = `bg-removal-${++requestCounter}`
  const activeWorker = getWorker()

  return new Promise<Blob>((resolve, reject) => {
    pending.set(id, { resolve, reject })
    const request: BackgroundRemovalWorkerRequest = { id, image }
    activeWorker.postMessage(request)
  })
}

/** Terminates the singleton worker and rejects any in-flight requests. Used by tests and app teardown. */
export function terminateBackgroundRemovalWorker(): void {
  if (worker) {
    worker.terminate()
    worker = null
  }
  rejectAllPending(new BackgroundRemovalError('terminated', 'Background removal worker was terminated'))
}
