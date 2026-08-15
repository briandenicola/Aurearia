import { afterEach, describe, expect, it, vi } from 'vitest'
import { useDeepIdentificationStream } from '../useDeepIdentificationStream'

vi.mock('@/api/client', () => ({
  refreshAccessToken: vi.fn().mockResolvedValue('refreshed-token'),
}))

function sseBody(frames: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  let index = 0
  return new ReadableStream({
    pull(controller) {
      if (index >= frames.length) {
        controller.close()
        return
      }
      controller.enqueue(encoder.encode(frames[index]))
      index += 1
    },
  })
}

function frame(id: number, event: string, data: Record<string, unknown>): string {
  return `id: ${id}\nevent: ${event}\ndata: ${JSON.stringify(data)}\n\n`
}

function controlFrame(event: string, data: Record<string, unknown>): string {
  return `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`
}

afterEach(() => {
  vi.unstubAllGlobals()
  localStorage.clear()
})

describe('useDeepIdentificationStream', () => {
  it('parses replayed frames in order and de-dupes by seq', async () => {
    const body = sseBody([
      frame(1, 'job_accepted', { seq: 1, jobId: 9, type: 'job_accepted', ts: '2030-01-01T00:00:00Z', payload: { status: 'queued' } }),
      frame(1, 'job_accepted', { seq: 1, jobId: 9, type: 'job_accepted', ts: '2030-01-01T00:00:00Z', payload: { status: 'queued' } }),
      frame(2, 'progress', { seq: 2, jobId: 9, type: 'progress', ts: '2030-01-01T00:00:01Z', payload: { message: 'Working' } }),
    ])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(body, { status: 200 })))

    const stream = useDeepIdentificationStream()
    await stream.connect(9)

    expect(stream.events.value).toHaveLength(2)
    expect(stream.events.value[0].seq).toBe(1)
    expect(stream.events.value[1].seq).toBe(2)
    expect(stream.lastSeq.value).toBe(2)
  })

  it('resumes with ?since= when a lastSeq is supplied', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(sseBody([]), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    const stream = useDeepIdentificationStream()
    await stream.connect(9, { since: 5 })

    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain('/deep-identification/jobs/9/events?since=5')
  })

  it('flags stream_truncated without consuming a sequence number', async () => {
    const body = sseBody([
      controlFrame('stream_truncated', { status: 'running', earliestSeq: 4, lastSeq: 10 }),
      frame(4, 'progress', { seq: 4, jobId: 9, type: 'progress', ts: '2030-01-01T00:00:00Z', payload: {} }),
    ])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(body, { status: 200 })))

    const stream = useDeepIdentificationStream()
    await stream.connect(9)

    expect(stream.truncated.value).toBe(true)
    expect(stream.events.value).toHaveLength(1)
    expect(stream.events.value[0].seq).toBe(4)
  })

  it('records the terminal payload and closes on event: end without reconnecting', async () => {
    const body = sseBody([
      frame(1, 'terminal', { seq: 1, jobId: 9, type: 'terminal', ts: '2030-01-01T00:00:00Z', payload: { status: 'completed', partialSuccess: false, hasReport: true, hasProposal: true } }),
      controlFrame('end', { jobId: 9, status: 'completed' }),
    ])
    const fetchMock = vi.fn().mockResolvedValue(new Response(body, { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    const stream = useDeepIdentificationStream()
    await stream.connect(9)

    expect(stream.ended.value).toBe(true)
    expect(stream.terminal.value?.status).toBe('completed')
    expect(stream.connected.value).toBe(false)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('surfaces a friendly error for 429/410/404 without throwing', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 429 })))
    const stream = useDeepIdentificationStream()
    await stream.connect(9)
    expect(stream.error.value).toContain('Too many active viewers')
    expect(stream.connected.value).toBe(false)
  })

  it('disconnect() aborts an in-flight stream', async () => {
    let rejectedWithAbort = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((_url: string, init: RequestInit) => {
      return new Promise((_resolve, reject) => {
        init.signal?.addEventListener('abort', () => {
          rejectedWithAbort = true
          reject(new DOMException('aborted', 'AbortError'))
        })
      })
    }))

    const stream = useDeepIdentificationStream()
    const connectPromise = stream.connect(9)
    stream.disconnect()
    await connectPromise

    expect(rejectedWithAbort).toBe(true)
    expect(stream.error.value).toBe('')
  })
})
