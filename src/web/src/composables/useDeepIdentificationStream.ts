import { ref, shallowRef } from 'vue'
import { refreshAccessToken } from '@/api/client'
import type { DeepJobStatus, DeepStreamEvent, DeepStreamTerminalPayload } from '@/types'

/**
 * Replayable/resumable SSE reader for a Deep Analysis job's event stream
 * (T098, contracts/sse-events.md). Mirrors the fetch + ReadableStream +
 * Authorization-header pattern already shipped in
 * `src/api/client.ts::agentChatStream` (native `EventSource` cannot set
 * headers, hence `?since=` instead of relying on `Last-Event-ID`).
 *
 * De-dupes by `seq` (defense-in-depth on top of the server's own exactly-
 * once guarantee), tracks `stream_truncated`/`terminal` control frames, and
 * never auto-reconnects after `event: end` (contract §2) - callers that
 * want resume-on-remount behavior call `connect(jobId, { since: lastSeq })`
 * explicitly (see DeepAnalysisPage.vue, T101).
 */
export function useDeepIdentificationStream() {
  const events = shallowRef<DeepStreamEvent[]>([])
  const connected = ref(false)
  const streaming = ref(false)
  const lastSeq = ref(0)
  const truncated = ref(false)
  const terminal = shallowRef<DeepStreamTerminalPayload | null>(null)
  const ended = ref(false)
  const error = ref('')

  const seenSeqs = new Set<number>()
  let abortController: AbortController | null = null

  function reset() {
    events.value = []
    connected.value = false
    streaming.value = false
    lastSeq.value = 0
    truncated.value = false
    terminal.value = null
    ended.value = false
    error.value = ''
    seenSeqs.clear()
  }

  function disconnect() {
    abortController?.abort()
    abortController = null
    connected.value = false
    streaming.value = false
  }

  function appendEvent(ev: DeepStreamEvent) {
    if (seenSeqs.has(ev.seq)) return
    seenSeqs.add(ev.seq)
    events.value = [...events.value, ev]
    lastSeq.value = Math.max(lastSeq.value, ev.seq)
  }

  function handleFrame(frame: string) {
    if (!frame.trim() || frame.startsWith(':')) return
    let eventType = 'message'
    let dataLine = ''
    let idLine: string | undefined
    for (const rawLine of frame.split('\n')) {
      const line = rawLine.replace(/\r$/, '')
      if (line.startsWith(':')) continue
      if (line.startsWith('event:')) eventType = line.slice(6).trim()
      else if (line.startsWith('data:')) dataLine += (dataLine ? '\n' : '') + line.slice(5).trim()
      else if (line.startsWith('id:')) idLine = line.slice(3).trim()
    }
    if (!dataLine) return

    let payload: Record<string, unknown>
    try {
      payload = JSON.parse(dataLine)
    } catch {
      return
    }

    if (eventType === 'stream_truncated') {
      truncated.value = true
      return
    }
    if (eventType === 'end') {
      ended.value = true
      return
    }
    if (eventType === 'terminal') {
      terminal.value = (payload.payload as unknown as DeepStreamTerminalPayload) ?? null
    }

    const seq = idLine !== undefined ? Number(idLine) : Number(payload.seq)
    if (!Number.isFinite(seq)) return
    appendEvent({
      seq,
      jobId: Number(payload.jobId),
      type: eventType,
      ts: String(payload.ts ?? ''),
      payload: (payload.payload as Record<string, unknown>) ?? {},
    })
  }

  async function fetchWithAuthRetry(url: string, signal: AbortSignal): Promise<Response> {
    const baseURL = import.meta.env.VITE_API_BASE_URL || ''
    const buildHeaders = (token: string | null) => {
      const headers = new Headers()
      headers.set('Accept', 'text/event-stream')
      if (token) headers.set('Authorization', 'Bearer ' + token)
      return headers
    }

    const token = localStorage.getItem('token')
    const firstResp = await fetch(`${baseURL}${url}`, { headers: buildHeaders(token), signal })
    if (firstResp.status !== 401) return firstResp

    const refreshed = await refreshAccessToken()
    return fetch(`${baseURL}${url}`, { headers: buildHeaders(refreshed), signal })
  }

  /**
   * Opens (or resumes) the stream for jobId. Pass `since` (the last seq
   * this client has already durably processed) to resume without
   * replaying already-seen events - the server still de-dupes, but
   * skipping the redundant replay window avoids unnecessary bandwidth on
   * long-running jobs.
   */
  async function connect(jobId: number, opts?: { since?: number }): Promise<void> {
    disconnect()
    error.value = ''
    ended.value = false
    truncated.value = false
    abortController = new AbortController()
    streaming.value = true

    const since = opts?.since
    const query = since !== undefined && since > 0 ? `?since=${since}` : ''
    const controllerForThisConnect = abortController

    try {
      const resp = await fetchWithAuthRetry(`/api/deep-identification/jobs/${jobId}/events${query}`, controllerForThisConnect.signal)
      if (!resp.ok) {
        error.value = resp.status === 429
          ? 'Too many active viewers for this job. Please try again shortly.'
          : resp.status === 410
            ? 'This Deep Analysis result has expired.'
            : resp.status === 404
              ? 'Deep Analysis job not found.'
              : `Unable to open the Deep Analysis stream (HTTP ${resp.status}).`
        streaming.value = false
        return
      }
      connected.value = true

      const reader = resp.body?.getReader()
      if (!reader) {
        error.value = 'Streaming is not supported in this browser.'
        return
      }
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const frames = buffer.split('\n\n')
        buffer = frames.pop() || ''
        for (const frame of frames) {
          handleFrame(frame)
        }
      }
      buffer += decoder.decode()
      if (buffer.trim()) handleFrame(buffer)
    } catch (err: unknown) {
      if (controllerForThisConnect.signal.aborted) return
      error.value = err instanceof Error ? err.message : 'Deep Analysis stream failed.'
    } finally {
      connected.value = false
      streaming.value = false
    }
  }

  return {
    events,
    connected,
    streaming,
    lastSeq,
    truncated,
    terminal,
    ended,
    error,
    connect,
    disconnect,
    reset,
  }
}

export type DeepIdentificationStream = ReturnType<typeof useDeepIdentificationStream>
export type { DeepJobStatus }
