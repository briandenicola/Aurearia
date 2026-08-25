import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  BACKGROUND_REMOVAL_ASSET_PATH,
  BackgroundRemovalError,
  backgroundRemovalConfig,
  removeCoinBackground,
  terminateBackgroundRemovalWorker,
} from '@/utils/backgroundRemoval'

type Listener = (event: unknown) => void

// The real removeCoinBackground() now delegates to a dedicated module worker
// (backgroundRemovalWorker.ts) rather than calling @imgly/background-removal
// directly, so these tests exercise the postMessage-based request/response
// protocol via a fake global Worker instead of mocking the library.
class FakeWorker {
  static instances: FakeWorker[] = []
  listeners: Record<string, Listener[]> = { message: [], error: [], messageerror: [] }
  postMessage = vi.fn()
  terminate = vi.fn()

  constructor(
    public url: URL,
    public options: WorkerOptions,
  ) {
    FakeWorker.instances.push(this)
  }

  addEventListener(type: string, listener: Listener) {
    this.listeners[type]?.push(listener)
  }

  removeEventListener(type: string, listener: Listener) {
    const list = this.listeners[type]
    if (!list) return
    const index = list.indexOf(listener)
    if (index !== -1) list.splice(index, 1)
  }

  emitMessage(data: unknown) {
    for (const listener of this.listeners.message) listener({ data })
  }

  emitError(message: string) {
    for (const listener of this.listeners.error) listener({ message })
  }

  emitMessageError() {
    for (const listener of this.listeners.messageerror) listener({})
  }

  lastRequestId(): string {
    const call = this.postMessage.mock.calls.at(-1)
    return (call?.[0] as { id: string }).id
  }
}

beforeEach(() => {
  FakeWorker.instances = []
  vi.stubGlobal('Worker', FakeWorker)
})

afterEach(() => {
  terminateBackgroundRemovalWorker()
  vi.unstubAllGlobals()
})

function currentWorker(): FakeWorker {
  const instance = FakeWorker.instances.at(-1)
  if (!instance) throw new Error('expected a worker to have been constructed')
  return instance
}

describe('backgroundRemoval config', () => {
  it('uses same-origin quantized model assets for production background removal', () => {
    expect(backgroundRemovalConfig).toMatchObject({
      publicPath: `${window.location.origin}${BACKGROUND_REMOVAL_ASSET_PATH}`,
      model: 'isnet_quint8',
      device: 'cpu',
      proxyToWorker: false,
      output: { format: 'image/png', quality: 1 },
    })
  })

  it('uses an absolute publicPath that IMG.LY can resolve resources.json against', () => {
    expect(new URL('resources.json', backgroundRemovalConfig.publicPath).href)
      .toBe(`${window.location.origin}${BACKGROUND_REMOVAL_ASSET_PATH}resources.json`)
  })
})

describe('removeCoinBackground (worker client)', () => {
  it('constructs a same-origin module worker on first use', async () => {
    const input = new Blob(['coin'], { type: 'image/png' })
    const promise = removeCoinBackground(input)

    const worker = currentWorker()
    expect(worker.options).toMatchObject({ type: 'module' })
    expect(String(worker.url)).toContain('backgroundRemovalWorker')

    worker.emitMessage({ id: worker.lastRequestId(), ok: true, result: new Blob(['out'], { type: 'image/png' }) })
    await expect(promise).resolves.toBeInstanceOf(Blob)
  })

  it('resolves with the PNG blob returned by the worker on success', async () => {
    const input = new Blob(['coin'], { type: 'image/png' })
    const promise = removeCoinBackground(input)
    const worker = currentWorker()
    const output = new Blob(['processed'], { type: 'image/png' })

    worker.emitMessage({ id: worker.lastRequestId(), ok: true, result: output })

    const result = await promise
    expect(result).toBe(output)
  })

  it('rejects with a normalized BackgroundRemovalError when the library fails inside the worker', async () => {
    const input = new Blob(['coin'], { type: 'image/png' })
    const promise = removeCoinBackground(input)
    const worker = currentWorker()

    worker.emitMessage({
      id: worker.lastRequestId(),
      ok: false,
      error: { name: 'EvalError', message: "'unsafe-eval' is not an allowed source of script" },
    })

    await expect(promise).rejects.toMatchObject({
      name: 'BackgroundRemovalError',
      stage: 'library',
      message: expect.stringContaining('unsafe-eval'),
    })
  })

  it('rejects all pending requests with stage "worker-error" on a worker error event, without hanging', async () => {
    const first = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const second = removeCoinBackground(new Blob(['b'], { type: 'image/png' }))
    const worker = currentWorker()

    worker.emitError('script failed to load')

    await expect(first).rejects.toBeInstanceOf(BackgroundRemovalError)
    await expect(first).rejects.toMatchObject({ stage: 'worker-error' })
    await expect(second).rejects.toMatchObject({ stage: 'worker-error' })
  })

  it('rejects pending requests with stage "message-error" when a message cannot be deserialized', async () => {
    const promise = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const worker = currentWorker()

    worker.emitMessageError()

    await expect(promise).rejects.toMatchObject({ stage: 'message-error' })
  })

  it('terminates the crashed worker and constructs+uses a fresh one on the next call after an "error" event', async () => {
    const first = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const deadWorker = currentWorker()

    deadWorker.emitError('script failed to load')
    await expect(first).rejects.toMatchObject({ stage: 'worker-error' })
    expect(deadWorker.terminate).toHaveBeenCalledTimes(1)

    // A later call must not reuse the dead instance -- it must construct a
    // fresh worker, and that fresh worker must actually complete the request
    // (not just exist), proving the singleton was recreated, not just leaked.
    const second = removeCoinBackground(new Blob(['b'], { type: 'image/png' }))
    expect(FakeWorker.instances).toHaveLength(2)
    const freshWorker = currentWorker()
    expect(freshWorker).not.toBe(deadWorker)

    freshWorker.emitMessage({ id: freshWorker.lastRequestId(), ok: true, result: new Blob(['out'], { type: 'image/png' }) })
    await expect(second).resolves.toBeInstanceOf(Blob)
  })

  it('terminates the crashed worker and constructs+uses a fresh one on the next call after a "messageerror" event', async () => {
    const first = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const deadWorker = currentWorker()

    deadWorker.emitMessageError()
    await expect(first).rejects.toMatchObject({ stage: 'message-error' })
    expect(deadWorker.terminate).toHaveBeenCalledTimes(1)

    const second = removeCoinBackground(new Blob(['b'], { type: 'image/png' }))
    expect(FakeWorker.instances).toHaveLength(2)
    const freshWorker = currentWorker()
    expect(freshWorker).not.toBe(deadWorker)

    freshWorker.emitMessage({ id: freshWorker.lastRequestId(), ok: true, result: new Blob(['out'], { type: 'image/png' }) })
    await expect(second).resolves.toBeInstanceOf(Blob)
  })

  it('does not clear a newer worker when a stale event from an already-replaced worker fires late', async () => {
    // Simulate reentrancy/event-ordering: worker A crashes and is replaced by
    // worker B, but A's `error` event listener is invoked again afterward
    // (e.g. a duplicate/late-delivered DOM event). This must not null out the
    // singleton that now points at the healthy worker B.
    const first = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const workerA = currentWorker()
    workerA.emitError('first crash')
    await expect(first).rejects.toMatchObject({ stage: 'worker-error' })

    const second = removeCoinBackground(new Blob(['b'], { type: 'image/png' }))
    const workerB = currentWorker()
    expect(workerB).not.toBe(workerA)

    // Late/duplicate error event from the already-invalidated worker A.
    workerA.emitError('stale late event from replaced worker')

    // worker B must still be the live singleton and able to complete `second`.
    workerB.emitMessage({ id: workerB.lastRequestId(), ok: true, result: new Blob(['out'], { type: 'image/png' }) })
    await expect(second).resolves.toBeInstanceOf(Blob)

    // A subsequent call must still reuse worker B (not be forced to recreate),
    // proving the stale event from A did not null the singleton.
    const third = removeCoinBackground(new Blob(['c'], { type: 'image/png' }))
    expect(FakeWorker.instances).toHaveLength(2)
    workerB.emitMessage({ id: workerB.lastRequestId(), ok: true, result: new Blob(['out2'], { type: 'image/png' }) })
    await expect(third).resolves.toBeInstanceOf(Blob)
  })

  it('correlates concurrent requests by id so responses resolve the matching caller', async () => {
    const first = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const second = removeCoinBackground(new Blob(['b'], { type: 'image/png' }))
    const worker = currentWorker()

    const [firstCall, secondCall] = worker.postMessage.mock.calls
    const firstId = (firstCall[0] as { id: string }).id
    const secondId = (secondCall[0] as { id: string }).id
    expect(firstId).not.toBe(secondId)

    const firstResult = new Blob(['first'], { type: 'image/png' })
    const secondResult = new Blob(['second'], { type: 'image/png' })

    // Resolve out of order to prove correlation isn't relying on call order.
    worker.emitMessage({ id: secondId, ok: true, result: secondResult })
    worker.emitMessage({ id: firstId, ok: true, result: firstResult })

    expect(await first).toBe(firstResult)
    expect(await second).toBe(secondResult)
  })

  it('reuses the same worker instance across multiple calls (singleton, no repeated model init)', async () => {
    const firstPromise = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const worker = currentWorker()
    worker.emitMessage({ id: worker.lastRequestId(), ok: true, result: new Blob(['x'], { type: 'image/png' }) })
    await firstPromise

    const secondPromise = removeCoinBackground(new Blob(['b'], { type: 'image/png' }))
    expect(FakeWorker.instances).toHaveLength(1)
    worker.emitMessage({ id: worker.lastRequestId(), ok: true, result: new Blob(['y'], { type: 'image/png' }) })
    await secondPromise
  })

  it('terminates the worker and rejects in-flight requests on cleanup', async () => {
    const promise = removeCoinBackground(new Blob(['a'], { type: 'image/png' }))
    const worker = currentWorker()

    terminateBackgroundRemovalWorker()

    expect(worker.terminate).toHaveBeenCalledTimes(1)
    await expect(promise).rejects.toMatchObject({ stage: 'terminated' })

    // A subsequent call must construct a fresh worker rather than reuse the terminated one.
    const nextPromise = removeCoinBackground(new Blob(['c'], { type: 'image/png' }))
    expect(FakeWorker.instances).toHaveLength(2)
    const nextWorker = currentWorker()
    nextWorker.emitMessage({ id: nextWorker.lastRequestId(), ok: true, result: new Blob(['z'], { type: 'image/png' }) })
    await nextPromise
  })
})
