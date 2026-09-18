import { describe, expect, it } from 'vitest'

import { validateRunConfiguration } from '../contracts'
import { clone, validRunConfiguration } from './contract-fixtures'

describe('run configuration contract', () => {
  it('accepts every exact hard maximum', () => {
    expect(validateRunConfiguration(validRunConfiguration())).toEqual({ ok: true, errors: [] })
  })

  it.each([
    ['unknown fields', (value: Record<string, unknown>) => (value.unexpected = true)],
    ['secret-shaped fields', (value: Record<string, unknown>) => (value.apiKey = 'sk-test-secret')],
    ['raw URLs', (value: Record<string, unknown>) => (value.model = 'https://example.test/model')],
    ['empty workflows', (value: Record<string, unknown>) => (value.workflowScope = [])],
    [
      'duplicate workflows',
      (value: Record<string, unknown>) =>
        (value.workflowScope = ['login-session', 'login-session']),
    ],
    ['unsupported providers', (value: Record<string, unknown>) => (value.provider = 'openai')],
  ])('rejects %s', (_name, mutate) => {
    const value = clone(validRunConfiguration()) as unknown as Record<string, unknown>
    mutate(value)
    expect(validateRunConfiguration(value).ok).toBe(false)
  })

  it.each([
    ['steps', 0],
    ['wallTimeSeconds', -1],
    ['modelCalls', 0],
    ['modelTokens', -1],
    ['browserActions', 0],
    ['issueAttempts', -1],
  ])('rejects non-positive %s values except the documented zero issue floor', (field, value) => {
    const config = clone(validRunConfiguration())
    ;(config.limits as unknown as Record<string, number>)[field] = value
    expect(validateRunConfiguration(config).ok).toBe(false)
  })

  it('accepts zero issue attempts', () => {
    const config = clone(validRunConfiguration())
    config.limits.issueAttempts = 0
    expect(validateRunConfiguration(config).ok).toBe(true)
  })

  it.each([
    ['steps', 26],
    ['wallTimeSeconds', 901],
    ['modelCalls', 21],
    ['modelTokens', 60001],
    ['browserActions', 101],
    ['issueAttempts', 4],
  ])('rejects %s above its hard maximum', (field, value) => {
    const config = clone(validRunConfiguration())
    ;(config.limits as unknown as Record<string, number>)[field] = value
    expect(validateRunConfiguration(config).ok).toBe(false)
  })
})
