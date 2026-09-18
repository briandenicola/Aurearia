import { describe, expect, it } from 'vitest'

import { validateExplorationReport } from '../contracts'
import { clone, validReport } from './contract-fixtures'

describe('exploration report contract', () => {
  it('accepts a complete minimal report', () => {
    expect(validateExplorationReport(validReport())).toEqual({ ok: true, errors: [] })
  })

  it.each([
    ['required fields', (report: Record<string, unknown>) => delete report.termination],
    ['enum values', (report: Record<string, unknown>) => (report.status = 'running')],
    [
      'chronological ordering',
      (report: Record<string, unknown>) => (report.endedAt = '2026-09-18T11:59:59Z'),
    ],
    [
      'usage snapshots',
      (report: Record<string, unknown>) =>
        ((report.usage as Record<string, unknown>).modelCalls = 21),
    ],
    [
      'evidence references',
      (report: Record<string, unknown>) =>
        (((report.findings as Array<Record<string, unknown>>)[0].evidenceIds as string[]) = [
          'ev_network_9999',
        ]),
    ],
    [
      'traversal-safe paths',
      (report: Record<string, unknown>) =>
        ((report.evidence as Array<Record<string, unknown>>)[0].artifactPath =
          'evidence/../secrets.txt'),
    ],
    [
      'production isolation',
      (report: Record<string, unknown>) =>
        ((report.environment as Record<string, unknown>).productionDataUsed = true),
    ],
  ])('rejects violations of %s', (_name, mutate) => {
    const report = clone(validReport()) as unknown as Record<string, unknown>
    mutate(report)
    expect(validateExplorationReport(report).ok).toBe(false)
  })

  it('rejects unknown fields throughout the report', () => {
    const report = clone(validReport()) as unknown as Record<string, unknown>
    ;(report.environment as Record<string, unknown>).databasePassword = 'secret'
    expect(validateExplorationReport(report).ok).toBe(false)
  })
})
