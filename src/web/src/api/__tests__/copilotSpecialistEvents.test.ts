import { describe, expect, it } from 'vitest'
import { createCoinCopilotSSEParser } from '@/api/endpoints/agent'
import type { CoinCopilotEvent } from '@/types'

// Mirrors CopilotSpecialistPublicEvidence from the Go API: a dealer listing
// carries dealerName/listedPrice/currency/availability and the coin fields.
function toolCompletedEvent(overrides: Record<string, unknown> = {}) {
  return {
    seq: 4,
    type: 'tool_completed',
    threadId: 'cct_1',
    runId: 'ccr_1',
    executionId: 'cce_1',
    ts: '2026-09-19T23:03:17Z',
    payload: {
      toolCallId: 'call_market',
      toolName: 'market_search',
      stepId: 'step-1',
      status: 'succeeded',
      durationMs: 39473,
      resultSummary: 'Dealer search returned source-backed evidence.',
      truncated: false,
      specialistResult: {
        capability: 'market_search',
        outcome: 'complete',
        items: [{
          kind: 'dealer_listing',
          title: 'Constans II Gold Solidus (640-668 AD)',
          sourceUrl: 'https://auction.catawiki.com/kavels/926753-byzantine-empire-gold-solidus',
          observedAt: '2026-09-19T23:03:17Z',
          confidence: 'medium',
          verificationState: 'partial',
          dealerName: 'Catawiki',
          listedPrice: 450,
          currency: 'USD',
          availability: 'unknown',
          ruler: 'Constans II',
          denomination: 'Solidus',
          era: 'Byzantine',
          material: 'Gold',
          description: 'Byzantine Empire gold solidus.',
          facts: ['Dealer: Catawiki', 'Price: USD 450'],
          matchedAttributes: [],
          materialDifferences: [],
          ...overrides,
        }],
        trend: null,
        warnings: [],
        truncation: {
          truncated: false,
          originalBytes: 11794,
          persistedBytes: 11794,
          digest: 'a'.repeat(64),
          omittedItems: 0,
        },
      },
    },
  }
}

function parse(event: unknown): CoinCopilotEvent[] {
  const received: CoinCopilotEvent[] = []
  const parser = createCoinCopilotSSEParser({ onEvent: (value) => { received.push(value); return true } })
  parser.push(`event: tool_completed\nid: 4\ndata: ${JSON.stringify(event)}\n\n`)
  parser.finish()
  return received
}

describe('coin copilot specialist events', () => {
  it('accepts a dealer listing carrying the fields the Go API projects', () => {
    const received = parse(toolCompletedEvent())

    expect(received).toHaveLength(1)
    const result = received[0]!.payload.specialistResult
    expect(result.items[0]!.dealerName).toBe('Catawiki')
    expect(result.items[0]!.availability).toBe('unknown')
    expect(result.items[0]!.listedPrice).toBe(450)
  })

  it('still rejects an unknown field so the contract cannot drift silently', () => {
    expect(parse(toolCompletedEvent({ secretField: 'x' }))).toHaveLength(0)
  })
})
