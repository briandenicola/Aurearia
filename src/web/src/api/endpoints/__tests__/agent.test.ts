import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it, vi } from 'vitest'

import {
  createCoinCopilotSSEParser,
  parseDeepAnalysisHandoffResult,
} from '@/api/endpoints/agent'

const fixturesDirectory = resolve(
  process.cwd(),
  '../../specs/362-coin-copilot-attribution/contracts/fixtures',
)

function loadFixture(name: string): Record<string, unknown> {
  return JSON.parse(readFileSync(resolve(fixturesDirectory, name), 'utf8')) as Record<string, unknown>
}

function toolCompletedEvent(resultSummary = '') {
  return {
    seq: 1,
    threadId: 'cct_fixture',
    runId: 'ccr_fixture',
    executionId: 'cce_fixture',
    type: 'tool_completed',
    ts: '2026-09-18T18:02:00Z',
    payload: {
      toolCallId: 'call_01',
      toolName: 'deep_analysis_handoff',
      stepId: 'step_01',
      status: 'succeeded',
      durationMs: 1,
      resultSummary,
      truncated: false,
    },
  }
}

function eventJSONAtSize(size: number) {
  const event = toolCompletedEvent()
  const base = JSON.stringify(event)
  event.payload.resultSummary = 'x'.repeat(size - new TextEncoder().encode(base).length)
  const encoded = JSON.stringify(event)
  expect(new TextEncoder().encode(encoded)).toHaveLength(size)
  return encoded
}

describe('Coin Copilot Deep Analysis handoff contract', () => {
  it('uses outcome as the sole top-level result discriminant', () => {
    const fixture = loadFixture('deep-analysis-handoff-valid.json')
    const results = fixture.results as Record<string, unknown>
    const parsed = parseDeepAnalysisHandoffResult(results.status_complete)
    expect(parsed?.outcome).toBe('status')
    expect(parsed?.job?.status).toBe('completed')

    const invalid = loadFixture('deep-analysis-handoff-invalid.json')
    const cases = invalid.result_cases as Array<{ name: string, payload?: unknown }>
    const statusCase = cases.find(testCase => testCase.name === 'result_level_status_discriminant')
    expect(parseDeepAnalysisHandoffResult(statusCase?.payload)).toBeNull()
  })

  it('rejects malformed, tampered, proposal/apply, and unsafe review-link payloads', () => {
    const invalid = loadFixture('deep-analysis-handoff-invalid.json')
    const cases = invalid.result_cases as Array<{ name: string, payload?: unknown, payload_json?: string }>
    for (const name of [
      'unknown_result_field',
      'forged_apply',
      'unsafe_absolute_review_url',
      'unsafe_protocol_relative_review_url',
      'mismatched_review_url',
    ]) {
      const testCase = cases.find(candidate => candidate.name === name)
      expect(parseDeepAnalysisHandoffResult(testCase?.payload), name).toBeNull()
    }
    const malformed = cases.find(candidate => candidate.name === 'malformed_output')
    expect(parseDeepAnalysisHandoffResult(malformed?.payload_json)).toBeNull()
  })

  it('requires bounded deterministic truncation disclosure', () => {
    const fixture = loadFixture('deep-analysis-handoff-valid.json')
    const results = fixture.results as Record<string, unknown>
    const parsed = parseDeepAnalysisHandoffResult(results.truncated)
    expect(parsed?.truncation).toMatchObject({
      truncated: true,
      persisted_bytes: 32768,
      omitted_evidence: 18,
    })

    const tampered = structuredClone(results.truncated) as Record<string, unknown>
    const truncation = tampered.truncation as Record<string, unknown>
    truncation.persisted_bytes = 32769
    expect(parseDeepAnalysisHandoffResult(tampered)).toBeNull()
  })

  it('accepts a 65,536-byte public event and rejects 65,537 bytes', () => {
    const onEvent = vi.fn()
    const parser = createCoinCopilotSSEParser({ onEvent })
    parser.push(`event: tool_completed\nid: 1\ndata: ${eventJSONAtSize(65536)}\n\n`)
    expect(onEvent).toHaveBeenCalledTimes(1)

    const oversized = createCoinCopilotSSEParser({ onEvent })
    oversized.push(`event: tool_completed\nid: 1\ndata: ${eventJSONAtSize(65537)}\n\n`)
    expect(onEvent).toHaveBeenCalledTimes(1)
  })
})
