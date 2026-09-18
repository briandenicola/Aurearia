import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createCoinCopilotSSEParser } from '@/api/endpoints/agent'
import type { CoinCopilotEvent, CoinCopilotRun } from '@/types'

const mocks = vi.hoisted(() => ({
  getCapability: vi.fn(),
  startRun: vi.fn(),
  getRun: vi.fn(),
  getThread: vi.fn(),
  cancelRun: vi.fn(),
  resumeRun: vi.fn(),
  streamEvents: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  getCoinCopilotCapability: (...args: unknown[]) => mocks.getCapability(...args),
  startCoinCopilotRun: (...args: unknown[]) => mocks.startRun(...args),
  getCoinCopilotRun: (...args: unknown[]) => mocks.getRun(...args),
  getCoinCopilotThread: (...args: unknown[]) => mocks.getThread(...args),
  cancelCoinCopilotRun: (...args: unknown[]) => mocks.cancelRun(...args),
  resumeCoinCopilotRun: (...args: unknown[]) => mocks.resumeRun(...args),
  streamCoinCopilotRunEvents: (...args: unknown[]) => mocks.streamEvents(...args),
  getApiErrorMessage: (error: unknown) => error instanceof Error ? error.message : '',
}))

import { useCoinCopilot } from '../useCoinCopilot'

function run(overrides: Partial<CoinCopilotRun> = {}): CoinCopilotRun {
  return {
    id: 'ccr_test',
    threadId: 'cct_test',
    status: 'queued',
    goal: 'Review my collection',
    checkpointVersion: 0,
    lastSeq: 0,
    attempt: 0,
    finalAnswer: null,
    failureCode: null,
    failureMessage: null,
    resumeDeadline: null,
    usage: { iterations: 0, toolCalls: 0, inputTokens: 0, outputTokens: 0 },
    createdAt: '2026-09-17T00:00:00Z',
    updatedAt: '2026-09-17T00:00:00Z',
    ...overrides,
  }
}

function event<T extends CoinCopilotEvent>(value: T): T {
  return value
}

describe('useCoinCopilot', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    mocks.getCapability.mockResolvedValue({
      data: { mode: 'copilot', enabled: true, modelToolCallingSupported: true },
    })
    mocks.startRun.mockResolvedValue({ data: { run: run({ status: 'completed', finalAnswer: 'Done.' }), reused: false } })
    mocks.getRun.mockResolvedValue({ data: { run: run({ status: 'running' }) } })
    mocks.getThread.mockResolvedValue({
      data: { thread: { id: 'cct_test', title: 'Review', messages: [], runs: [], createdAt: '', updatedAt: '' } },
    })
    mocks.streamEvents.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it.each([
    ['disabled', false],
    ['provider_unconfigured', false],
    ['model_tool_calling_unsupported', false],
    ['temporarily_unavailable', true],
  ] as const)('uses legacy mode for %s capability fallback', async (reason, enabled) => {
    mocks.getCapability.mockResolvedValue({
      data: { mode: 'legacy', enabled, modelToolCallingSupported: false, reason },
    })
    const copilot = useCoinCopilot()

    const result = await copilot.start('Find gaps')

    expect(result).toEqual({ accepted: false, fallback: true })
    expect(mocks.startRun).not.toHaveBeenCalled()
  })

  it('reuses the same start idempotency key when an unacknowledged request is retried', async () => {
    mocks.startRun
      .mockRejectedValueOnce(new Error('connection reset'))
      .mockResolvedValueOnce({ data: { run: run({ status: 'completed', finalAnswer: 'Done.' }), reused: true } })
    const copilot = useCoinCopilot()

    const result = await copilot.start('Find gaps')

    expect(result).toMatchObject({ accepted: true, status: 'completed', answer: 'Done.' })
    expect(mocks.startRun).toHaveBeenCalledTimes(2)
    expect(mocks.startRun.mock.calls[0]?.[1]).toBe(mocks.startRun.mock.calls[1]?.[1])
  })

  it('reconnects from the last sequence and preserves multiple tool completions', async () => {
    mocks.startRun.mockResolvedValue({ data: { run: run({ status: 'running' }), reused: false } })
    mocks.streamEvents
      .mockImplementationOnce(async (_runId, handlers) => {
        handlers.onEvent(event({
          seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'tool_completed', ts: '',
          payload: { toolCallId: 'call_1', toolName: 'collection_summary', stepId: 'step_1', status: 'succeeded', durationMs: 10, resultSummary: 'Summary returned.', truncated: false },
        }))
        throw new Error('disconnect')
      })
      .mockImplementationOnce(async (_runId, handlers) => {
        handlers.onEvent(event({
          seq: 2, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'tool_completed', ts: '',
          payload: { toolCallId: 'call_2', toolName: 'gap_analysis', stepId: 'step_2', status: 'succeeded', durationMs: 12, resultSummary: 'No matching gaps were found.', truncated: false },
        }))
        handlers.onEvent(event({
          seq: 3, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'run_completed', ts: '',
          payload: { answer: 'No gaps found.', usage: { iterations: 2, toolCalls: 2, inputTokens: 20, outputTokens: 10 } },
        }))
      })
    const copilot = useCoinCopilot()

    const result = await copilot.start('Find gaps')

    expect(result).toMatchObject({ accepted: true, status: 'completed', answer: 'No gaps found.' })
    expect(mocks.streamEvents).toHaveBeenCalledTimes(2)
    expect(mocks.streamEvents.mock.calls[1]?.[2]).toMatchObject({ since: 1 })
    expect(copilot.tools.value.map(tool => tool.toolCallId)).toEqual(['call_1', 'call_2'])
    expect(sessionStorage.getItem('coinCopilot:activeRun')).toBeNull()
  })

  it('replays missed clarification events after refresh reports the run paused', async () => {
    const appliedSeqs: number[] = []
    mocks.startRun.mockResolvedValue({ data: { run: run({ status: 'running' }), reused: false } })
    mocks.getRun.mockResolvedValue({
      data: { run: run({ status: 'paused', checkpointVersion: 4, lastSeq: 3 }) },
    })
    mocks.streamEvents
      .mockImplementationOnce(async (_runId, handlers, options) => {
        const first = event({
          seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'plan_updated', ts: '',
          payload: { plan: [{ id: 'step_1', title: 'Clarify scope', status: 'in_progress' }] },
        })
        options.seenSeqs.add(first.seq)
        appliedSeqs.push(first.seq)
        handlers.onEvent(first)
        throw new Error('disconnect')
      })
      .mockImplementationOnce(async (_runId, handlers, options) => {
        expect(JSON.parse(sessionStorage.getItem('coinCopilot:activeRun') ?? '{}')).toMatchObject({
          lastSeq: 1,
        })
        const replay = [
          event({
            seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'plan_updated', ts: '',
            payload: { plan: [{ id: 'step_1', title: 'Clarify scope', status: 'in_progress' }] },
          }),
          event({
            seq: 2, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'clarification_required', ts: '',
            payload: { question: 'Include sold coins?', inputType: 'boolean', choices: [], checkpointVersion: 4 },
          }),
          event({
            seq: 3, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'run_paused', ts: '',
            payload: { reason: 'clarification_required', checkpointVersion: 4, resumeDeadline: '2026-09-24T00:00:00Z' },
          }),
        ] as CoinCopilotEvent[]
        for (const replayedEvent of replay) {
          if (options.seenSeqs.has(replayedEvent.seq)) continue
          options.seenSeqs.add(replayedEvent.seq)
          appliedSeqs.push(replayedEvent.seq)
          if (handlers.onEvent(replayedEvent) === false) break
        }
      })
    const copilot = useCoinCopilot()

    const result = await copilot.start('Compare bronzes')

    expect(result).toMatchObject({ accepted: true, status: 'paused' })
    expect(mocks.getRun).toHaveBeenCalledTimes(1)
    expect(mocks.streamEvents).toHaveBeenCalledTimes(2)
    expect(mocks.streamEvents.mock.calls[1]?.[2]).toMatchObject({ since: 1 })
    expect(appliedSeqs).toEqual([1, 2, 3])
    expect(copilot.lastSeq.value).toBe(3)
    expect(copilot.clarification.value).toMatchObject({
      question: 'Include sold coins?',
      checkpointVersion: 4,
    })
    expect(copilot.canResume.value).toBe(true)
  })

  it('replays missed terminal events after refresh reports the run completed', async () => {
    const appliedSeqs: number[] = []
    mocks.startRun.mockResolvedValue({ data: { run: run({ status: 'running' }), reused: false } })
    mocks.getRun.mockResolvedValue({
      data: { run: run({ status: 'completed', finalAnswer: 'Recovered answer.', lastSeq: 3 }) },
    })
    mocks.streamEvents
      .mockImplementationOnce(async (_runId, handlers, options) => {
        const first = event({
          seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'tool_completed', ts: '',
          payload: { toolCallId: 'call_1', toolName: 'collection_summary', stepId: 'step_1', status: 'succeeded', durationMs: 10, resultSummary: 'Summary returned.', truncated: false },
        })
        options.seenSeqs.add(first.seq)
        appliedSeqs.push(first.seq)
        handlers.onEvent(first)
        throw new Error('disconnect')
      })
      .mockImplementationOnce(async (_runId, handlers, options) => {
        expect(JSON.parse(sessionStorage.getItem('coinCopilot:activeRun') ?? '{}')).toMatchObject({
          lastSeq: 1,
        })
        const replay = [
          event({
            seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'tool_completed', ts: '',
            payload: { toolCallId: 'call_1', toolName: 'collection_summary', stepId: 'step_1', status: 'succeeded', durationMs: 10, resultSummary: 'Summary returned.', truncated: false },
          }),
          event({
            seq: 2, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'tool_completed', ts: '',
            payload: { toolCallId: 'call_2', toolName: 'portfolio_review', stepId: 'step_2', status: 'succeeded', durationMs: 12, resultSummary: 'Portfolio reviewed.', truncated: false },
          }),
          event({
            seq: 3, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'run_completed', ts: '',
            payload: { answer: 'Recovered answer.', usage: { iterations: 2, toolCalls: 2, inputTokens: 20, outputTokens: 10 } },
          }),
        ] as CoinCopilotEvent[]
        for (const replayedEvent of replay) {
          if (options.seenSeqs.has(replayedEvent.seq)) continue
          options.seenSeqs.add(replayedEvent.seq)
          appliedSeqs.push(replayedEvent.seq)
          if (handlers.onEvent(replayedEvent) === false) break
        }
      })
    const copilot = useCoinCopilot()

    const result = await copilot.start('Review my collection')

    expect(result).toMatchObject({ accepted: true, status: 'completed', answer: 'Recovered answer.' })
    expect(mocks.getRun).toHaveBeenCalledTimes(1)
    expect(mocks.streamEvents).toHaveBeenCalledTimes(2)
    expect(mocks.streamEvents.mock.calls[1]?.[2]).toMatchObject({ since: 1 })
    expect(appliedSeqs).toEqual([1, 2, 3])
    expect(copilot.lastSeq.value).toBe(3)
    expect(copilot.tools.value.map(tool => tool.toolCallId)).toEqual(['call_1', 'call_2'])
    expect(sessionStorage.getItem('coinCopilot:activeRun')).toBeNull()
  })

  it('surfaces a typed failure and closes terminal state', async () => {
    mocks.startRun.mockResolvedValue({ data: { run: run({ status: 'running' }), reused: false } })
    mocks.streamEvents.mockImplementation(async (_runId, handlers) => {
      handlers.onEvent(event({
        seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'run_failed', ts: '',
        payload: {
          code: 'tool_limit_exceeded',
          message: 'Coin Copilot reached its tool-call limit.',
          retryable: true,
          usage: { iterations: 8, toolCalls: 12, inputTokens: 1, outputTokens: 1 },
        },
      }))
    })
    const copilot = useCoinCopilot()

    const result = await copilot.start('Broad review')

    expect(result).toMatchObject({ accepted: true, status: 'failed', error: 'Coin Copilot reached its tool-call limit.' })
    expect(copilot.terminal.value).toBe(true)
    expect(copilot.connected.value).toBe(false)
  })

  it('gates clarification resume and cancellation by durable run state', async () => {
    mocks.startRun.mockResolvedValue({ data: { run: run({ status: 'running' }), reused: false } })
    mocks.streamEvents.mockImplementationOnce(async (_runId, handlers) => {
      handlers.onEvent(event({
        seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'clarification_required', ts: '',
        payload: { question: 'Include sold coins?', inputType: 'boolean', choices: [], checkpointVersion: 4 },
      }))
      handlers.onEvent(event({
        seq: 2, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'run_paused', ts: '',
        payload: { reason: 'clarification_required', checkpointVersion: 4, resumeDeadline: '2026-09-24T00:00:00Z' },
      }))
    })
    const copilot = useCoinCopilot()
    await copilot.start('Compare bronzes')

    expect(copilot.canResume.value).toBe(true)
    expect(copilot.canCancel.value).toBe(true)
    expect(await copilot.resume('')).toBe(false)
    expect(mocks.resumeRun).not.toHaveBeenCalled()

    mocks.cancelRun.mockResolvedValue({ data: { run: run({ status: 'cancelled', checkpointVersion: 4 }) } })
    expect(await copilot.cancel()).toBe(true)
    expect(copilot.canResume.value).toBe(false)
    expect(await copilot.resume('Yes')).toBe(false)
    expect(mocks.resumeRun).not.toHaveBeenCalled()
  })

  it('reuses the same resume idempotency key after a network failure', async () => {
    mocks.startRun.mockResolvedValue({ data: { run: run({ status: 'running' }), reused: false } })
    mocks.streamEvents
      .mockImplementationOnce(async (_runId, handlers) => {
        handlers.onEvent(event({
          seq: 1, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'clarification_required', ts: '',
          payload: { question: 'Include sold coins?', inputType: 'boolean', choices: [], checkpointVersion: 4 },
        }))
        handlers.onEvent(event({
          seq: 2, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'run_paused', ts: '',
          payload: { reason: 'clarification_required', checkpointVersion: 4, resumeDeadline: '2026-09-24T00:00:00Z' },
        }))
      })
      .mockImplementationOnce(async (_runId, handlers) => {
        handlers.onEvent(event({
          seq: 3, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_2', type: 'run_completed', ts: '',
          payload: { answer: 'Resumed.', usage: { iterations: 2, toolCalls: 1, inputTokens: 1, outputTokens: 1 } },
        }))
      })
    mocks.resumeRun
      .mockRejectedValueOnce(new Error('connection reset'))
      .mockResolvedValueOnce({ data: { run: run({ status: 'queued', checkpointVersion: 5, attempt: 2 }), reused: true } })
    const copilot = useCoinCopilot()
    await copilot.start('Compare bronzes')

    expect(await copilot.resume('Yes')).toBe(true)
    expect(mocks.resumeRun).toHaveBeenCalledTimes(2)
    expect(mocks.resumeRun.mock.calls[0]?.[2]).toBe(mocks.resumeRun.mock.calls[1]?.[2])
    expect(copilot.run.value?.finalAnswer).toBe('Resumed.')
  })

  it('replays a paused run from retained events so clarification survives drawer reopen', async () => {
    sessionStorage.setItem('coinCopilot:activeRun', JSON.stringify({
      runId: 'ccr_test',
      threadId: 'cct_test',
      lastSeq: 8,
    }))
    mocks.getRun.mockResolvedValue({
      data: { run: run({ status: 'paused', checkpointVersion: 4, lastSeq: 8 }) },
    })
    mocks.streamEvents.mockImplementation(async (_runId, handlers) => {
      handlers.onEvent(event({
        seq: 7, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'clarification_required', ts: '',
        payload: { question: 'Include sold coins?', inputType: 'boolean', choices: [], checkpointVersion: 4 },
      }))
      handlers.onEvent(event({
        seq: 8, threadId: 'cct_test', runId: 'ccr_test', executionId: 'cce_1', type: 'run_paused', ts: '',
        payload: { reason: 'clarification_required', checkpointVersion: 4, resumeDeadline: '2026-09-24T00:00:00Z' },
      }))
    })
    const copilot = useCoinCopilot()

    expect(await copilot.restoreActiveRun()).toBe(true)
    expect(mocks.streamEvents.mock.calls[0]?.[2]).toMatchObject({ since: 0 })
    expect(copilot.clarification.value?.question).toBe('Include sold coins?')
    expect(copilot.canResume.value).toBe(true)
  })
})

describe('Coin Copilot SSE parser', () => {
  it('de-duplicates replayed sequences and ignores unknown events', () => {
    const events: CoinCopilotEvent[] = []
    const parser = createCoinCopilotSSEParser({ onEvent: value => { events.push(value) } })
    const valid = 'id: 2\nevent: tool_started\ndata: {"seq":2,"threadId":"cct_1","runId":"ccr_1","executionId":"cce_1","type":"tool_started","ts":"2026-09-17T00:00:00Z","payload":{"toolCallId":"call_1","toolName":"get_coin","stepId":"step_1"}}\n\n'
    parser.push(valid + valid)
    parser.push('id: 3\nevent: future_event\ndata: {"seq":3,"type":"future_event","payload":{"unsafe":"ignored"}}\n\n')
    parser.finish()

    expect(events).toHaveLength(1)
    expect(events[0]?.seq).toBe(2)
  })

  it('rejects malformed or mismatched envelopes and honors terminal end', () => {
    const events: CoinCopilotEvent[] = []
    const ends: string[] = []
    const parser = createCoinCopilotSSEParser({
      onEvent: value => { events.push(value) },
      onEnd: value => { ends.push(value.status) },
    })
    parser.push('id: 4\nevent: tool_started\ndata: {"seq":5,"threadId":"cct_1","runId":"ccr_1","executionId":"cce_1","type":"tool_started","ts":"","payload":{"toolCallId":"call_1","toolName":"get_coin","stepId":"step_1"}}\n\n')
    parser.push('event: run_completed\ndata: {"seq":6,"threadId":"cct_1","runId":"ccr_1","executionId":"cce_1","type":"run_completed","ts":"","payload":{"answer":{},"usage":{}}}\n\n')
    parser.push('event: end\ndata: {"runId":"ccr_1","status":"completed"}\n\n')
    parser.finish()

    expect(events).toEqual([])
    expect(ends).toEqual(['completed'])
    expect(parser.stopped).toBe(true)
  })

  it('rejects removed cost fields while accepting token usage', () => {
    const events: CoinCopilotEvent[] = []
    const parser = createCoinCopilotSSEParser({ onEvent: value => { events.push(value) } })
    parser.push('id: 1\nevent: run_completed\ndata: {"seq":1,"threadId":"cct_1","runId":"ccr_1","executionId":"cce_1","type":"run_completed","ts":"","payload":{"answer":"Done","usage":{"iterations":1,"toolCalls":0,"inputTokens":12,"outputTokens":4,"estimatedCostMicros":0}}}\n\n')
    parser.push('id: 2\nevent: run_completed\ndata: {"seq":2,"threadId":"cct_1","runId":"ccr_1","executionId":"cce_1","type":"run_completed","ts":"","payload":{"answer":"Done","usage":{"iterations":1,"toolCalls":0,"inputTokens":12,"outputTokens":4}}}\n\n')
    parser.finish()

    expect(events).toHaveLength(1)
    expect(events[0]?.seq).toBe(2)
    expect(events[0]?.payload).toMatchObject({
      usage: { inputTokens: 12, outputTokens: 4 },
    })
  })

  it('accepts strict specialist projections and rejects mismatched or extended results', () => {
    const events: CoinCopilotEvent[] = []
    const parser = createCoinCopilotSSEParser({ onEvent: value => { events.push(value) } })
    const specialistResult = {
      capability: 'market_search',
      outcome: 'complete',
      items: [{
        kind: 'dealer_listing',
        title: 'Domitian denarius',
        sourceUrl: 'https://www.vcoins.com/en/stores/example/1/product/1/1',
        observedAt: '2026-09-18T12:00:00Z',
        confidence: 'high',
        verificationState: 'verified',
        facts: ['USD 250', 'Available'],
        matchedAttributes: [],
        materialDifferences: [],
      }],
      trend: null,
      warnings: [],
      truncation: {
        truncated: false,
        originalBytes: 500,
        persistedBytes: 500,
        digest: 'a'.repeat(64),
        omittedItems: 0,
      },
    }
    const frame = (seq: number, result: unknown) =>
      `id: ${seq}\nevent: tool_completed\ndata: ${JSON.stringify({
        seq,
        threadId: 'cct_1',
        runId: 'ccr_1',
        executionId: 'cce_1',
        type: 'tool_completed',
        ts: '2026-09-18T12:00:00Z',
        payload: {
          toolCallId: `call_${seq}`,
          toolName: 'market_search',
          stepId: `step_${seq}`,
          status: 'succeeded',
          durationMs: 10,
          resultSummary: 'One verified listing.',
          truncated: false,
          specialistResult: result,
        },
      })}\n\n`

    parser.push(frame(1, specialistResult))
    parser.push(frame(2, { ...specialistResult, capability: 'auction_search' }))
    parser.push(frame(3, { ...specialistResult, hiddenReasoning: 'not allowed' }))
    parser.finish()

    expect(events).toHaveLength(1)
    expect(events[0]?.payload).toMatchObject({ specialistResult })
  })
})
