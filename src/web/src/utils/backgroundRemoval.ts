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
  // The worker instance the request was posted to. Lets a fatal error on one
  // worker reject only *its own* in-flight requests, so a late/duplicate
  // event from an already-replaced worker can't spuriously fail requests
  // that were legitimately posted to (and are still in flight on) a newer,
  // healthy worker.
  worker: Worker
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
  // attributed to a single in-flight request id, so reject everything this
  // worker has outstanding rather than let those callers hang silently. The
  // worker is also fatally broken at this point (its module has
  // failed/thrown), so it must be invalidated here too -- otherwise the
  // singleton keeps handing out the same dead instance to every future call,
  // which then hangs forever with no resolve/reject.
  instance.addEventListener('error', (event: ErrorEvent) => {
    rejectPendingForWorker(
      instance,
      new BackgroundRemovalError('worker-error', event.message || 'Background removal worker crashed'),
    )
    invalidateWorker(instance)
  })

  instance.addEventListener('messageerror', () => {
    rejectPendingForWorker(
      instance,
      new BackgroundRemovalError('message-error', 'Background removal worker returned an unreadable message'),
    )
    invalidateWorker(instance)
  })

  return instance
}

// Rejects only the pending requests that were posted to `instance`, leaving
// requests belonging to any other (e.g. newer, replacement) worker
// untouched. Scoping by instance -- rather than rejecting the entire shared
// `pending` map -- is what makes the crash-recovery path safe against a
// stale/duplicate event arriving after the singleton has already moved on to
// a fresh worker.
function rejectPendingForWorker(instance: Worker, error: BackgroundRemovalError) {
  for (const [id, request] of pending) {
    if (request.worker !== instance) continue
    pending.delete(id)
    request.reject(error)
  }
}

// Terminates and disowns a fatally-broken worker so the next `getWorker()`
// call constructs a fresh one instead of reusing a dead instance. Guarded on
// identity (not a bare `worker = null`) because `error`/`messageerror` are
// asynchronous DOM events: an old, already-replaced worker's event could
// still be in flight when it fires, and must not be allowed to clear a
// newer, healthy singleton that was constructed in the meantime.
function invalidateWorker(instance: Worker) {
  if (worker === instance) {
    worker = null
  }
  instance.terminate()
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
    pending.set(id, { worker: activeWorker, resolve, reject })
    const request: BackgroundRemovalWorkerRequest = { id, image }
    activeWorker.postMessage(request)
  })
}

/** Terminates the singleton worker and rejects any in-flight requests. Used by tests and app teardown. */
export function terminateBackgroundRemovalWorker(): void {
  const terminated = worker
  if (worker) {
    worker.terminate()
    worker = null
  }
  const error = new BackgroundRemovalError('terminated', 'Background removal worker was terminated')
  if (terminated) {
    rejectPendingForWorker(terminated, error)
  } else {
    // No live worker to scope by (e.g. called before first use, or twice in
    // a row) -- fall back to clearing anything left dangling in `pending`.
    for (const [id, request] of pending) {
      pending.delete(id)
      request.reject(error)
    }
  }
}
