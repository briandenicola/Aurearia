import { describe, expect, it, vi } from 'vitest'
import { BudgetExceededError, BudgetGuard } from '../budget'
import type { ExplorationLimits } from '../contracts'

const limits: ExplorationLimits = {
  steps: 25,
  wallTimeSeconds: 900,
  modelCalls: 20,
  modelTokens: 60_000,
  browserActions: 100,
  issueAttempts: 3,
}

describe('BudgetGuard', () => {
  it.each([
    ['steps', 25],
    ['modelCalls', 20],
    ['browserActions', 100],
    ['issueAttempts', 3],
  ] as const)('allows the exact %s boundary and refuses one over', (kind, maximum) => {
    const guard = new BudgetGuard(limits)
    guard.reserve(kind, maximum)
    expect(guard.snapshot().usage[kind]).toBe(maximum)
    expect(() => guard.reserve(kind)).toThrow(BudgetExceededError)
    expect(guard.snapshot().usage[kind]).toBe(maximum)
  })

  it('allows exactly 60,000 provider-reported tokens and fails closed one over', () => {
    const guard = new BudgetGuard(limits)
    guard.reserve('modelCalls')
    guard.reconcileModelUsage({ inputTokens: 40_000, outputTokens: 20_000 })
    expect(guard.snapshot().usage.modelTokens).toBe(60_000)
    expect(() => guard.reconcileModelUsage({ inputTokens: 1, outputTokens: 0 })).toThrow(BudgetExceededError)
  })

  it('uses lower configured limits and immutable snapshots', () => {
    const configured = { ...limits, steps: 2 }
    const guard = new BudgetGuard(configured)
    configured.steps = 25
    expect(Object.isFrozen(guard.limits)).toBe(true)
    guard.reserve('steps', 2)
    expect(() => guard.reserve('steps')).toThrow(BudgetExceededError)
  })

  it('uses a monotonic 900 second deadline and aborts in-flight work', async () => {
    let now = 0
    const guard = new BudgetGuard(limits, { now: () => now })
    const work = guard.perform('modelCalls', () => new Promise((_resolve, reject) => {
      guard.signal.addEventListener('abort', () => reject(guard.signal.reason))
    }))
    now = 900_000
    expect(() => guard.checkDeadline()).toThrow(/wallTimeSeconds limit/i)
    await expect(work).rejects.toThrow(/wallTimeSeconds|wall time/i)
    guard.dispose()
  })

  it('counts attempted work even when it fails', async () => {
    const guard = new BudgetGuard(limits)
    await expect(guard.perform('browserActions', async () => { throw new Error('boom') })).rejects.toThrow('boom')
    expect(guard.snapshot().usage.browserActions).toBe(1)
  })

  it('rejects missing or malformed provider usage before another model call', () => {
    const guard = new BudgetGuard(limits)
    guard.reserve('modelCalls')
    expect(() => guard.reconcileModelUsage(undefined)).toThrow(/provider usage/i)
    expect(() => guard.reserve('modelCalls')).toThrow(/provider usage/i)
  })

  it('refuses another model call after exact token exhaustion', () => {
    const guard = new BudgetGuard(limits)
    guard.reserve('modelCalls')
    guard.reconcileModelUsage({ inputTokens: 40_000, outputTokens: 20_000 })
    expect(() => guard.reserve('modelCalls')).toThrow(/modelTokens limit/i)
    guard.dispose()
  })

  it('marks an exact terminal boundary as bounded', () => {
    const guard = new BudgetGuard({ ...limits, steps: 1 })
    guard.reserve('steps')
    expect(guard.termination()).toEqual({
      reason: 'step_limit',
      reachedLimits: ['steps'],
    })
    expect(() => guard.throwIfReached()).toThrow(/steps limit/i)
    guard.dispose()
  })

  it('does not mark a disabled zero limit as reached unless work is attempted', () => {
    const guard = new BudgetGuard({ ...limits, issueAttempts: 0 })
    expect(guard.termination()).toEqual({
      reason: 'completed',
      reachedLimits: [],
    })
    expect(() => guard.reserve('issueAttempts')).toThrow(/issueAttempts limit/i)
    expect(guard.termination()).toEqual({
      reason: 'issue_attempt_limit',
      reachedLimits: ['issueAttempts'],
    })
    guard.dispose()
  })

  it('reports every simultaneous limit with documented precedence', () => {
    const guard = new BudgetGuard({ ...limits, steps: 1, modelCalls: 1, browserActions: 1 })
    guard.reserve('steps')
    guard.reserve('modelCalls')
    guard.reserve('browserActions')
    const termination = guard.termination()
    expect(termination.reason).toBe('model_call_limit')
    expect(termination.reachedLimits).toEqual(['modelCalls', 'browserActions', 'steps'])
    guard.dispose()
  })

  it('returns detached immutable usage snapshots', () => {
    const guard = new BudgetGuard(limits)
    const snapshot = guard.snapshot()
    expect(Object.isFrozen(snapshot.usage)).toBe(true)
    expect(() => Object.assign(snapshot.usage, { steps: 9 })).toThrow()
    expect(guard.snapshot().usage.steps).toBe(0)
    guard.dispose()
    vi.restoreAllMocks()
  })
})
