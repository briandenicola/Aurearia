import { describe, expect, it, vi } from 'vitest'
import { BudgetGuard } from '../budget'
import {
  authenticateExplorer,
  executeDecisionLoop,
  runExplorationLifecycle,
  waitFor,
} from '../cli'
import { F013_WORKFLOW_INVENTORY } from '../../fixtures/workflow'

describe('exploration orchestration failures', () => {
  it('waits for authenticated navigation before enforcing the route guard', async () => {
    let currentURL = 'http://127.0.0.1:49152/login'
    const page = {
      goto: vi.fn(),
      url: vi.fn(() => currentURL),
      waitForURL: vi.fn(async () => { currentURL = 'http://127.0.0.1:49152/' }),
      getByRole: vi.fn((_role: string, options?: { name?: string }) =>
        options?.name === 'Sign In'
          ? { click: vi.fn() }
          : { first: () => ({ fill: vi.fn() }) }),
      locator: vi.fn(() => ({ fill: vi.fn() })),
    }
    const budget = new BudgetGuard({
      steps: 1,
      wallTimeSeconds: 60,
      modelCalls: 1,
      modelTokens: 10,
      browserActions: 5,
      issueAttempts: 0,
    })
    await expect(authenticateExplorer(
      page as never,
      budget,
      'http://127.0.0.1:49152',
      'explorer',
      'password',
    )).resolves.toBeUndefined()
    expect(page.waitForURL).toHaveBeenCalledWith('http://127.0.0.1:49152/')
    budget.dispose()
  })

  it('reports the last host readiness probe failure', async () => {
    await expect(waitFor(
      'http://127.0.0.1:1/healthz',
      new AbortController().signal,
      1,
      0,
    )).rejects.toThrow(/last probe:/)
  })

  it.each(['app', 'api', 'agent'])('finalizes and tears down when %s readiness fails', async (service) => {
    const events: string[] = []
    await expect(runExplorationLifecycle({
      provision: async () => events.push('provision'),
      seed: async () => events.push('seed'),
      waitReady: async () => { throw new Error(`${service} readiness failed`) },
      explore: async () => events.push('explore'),
      finalize: async () => events.push('finalize'),
      teardown: async () => events.push('teardown'),
    })).rejects.toThrow(`${service} readiness failed`)
    expect(events).toEqual(['provision', 'seed', 'finalize', 'teardown'])
  })

  it.each(['authentication expired', 'no progress', 'modal focus trap', 'cancelled', 'in-flight timeout'])(
    'preserves partial finalization after %s',
    async (message) => {
      const finalize = vi.fn()
      const teardown = vi.fn()
      await expect(runExplorationLifecycle({
        provision: async () => undefined,
        seed: async () => undefined,
        waitReady: async () => undefined,
        explore: async () => { throw new Error(message) },
        finalize,
        teardown,
      })).rejects.toThrow(message)
      expect(finalize).toHaveBeenCalledOnce()
      expect(teardown).toHaveBeenCalledOnce()
    },
  )

  it('executes model decisions until finish', async () => {
    const fill = vi.fn()
    const body = { ariaSnapshot: vi.fn(async () => '- textbox "Notes": updated') }
    const page = {
      url: vi.fn(() => 'http://127.0.0.1:49152/coin/1'),
      locator: vi.fn(() => body),
      getByLabel: vi.fn(() => ({ fill })),
      getByRole: vi.fn(),
      mainFrame: vi.fn(),
      route: vi.fn(),
      unroute: vi.fn(),
    }
    const decisions = [
      {
        schemaVersion: 'aurearia.browser-exploration-decision/v1',
        action: 'fill',
        target: { label: 'Notes', value: 'updated' },
        rationale: 'Exercise the field.',
        suspectedFindings: [],
        usage: { inputTokens: 10, outputTokens: 5 },
      },
      {
        schemaVersion: 'aurearia.browser-exploration-decision/v1',
        action: 'finish',
        target: null,
        rationale: 'Goal complete.',
        suspectedFindings: [],
        usage: { inputTokens: 8, outputTokens: 3 },
      },
    ]
    const budget = new BudgetGuard({
      steps: 5,
      wallTimeSeconds: 60,
      modelCalls: 5,
      modelTokens: 100,
      browserActions: 10,
      issueAttempts: 1,
    })
    let step = 0
    await expect(executeDecisionLoop({
      page: page as never,
      budget,
      workflow: F013_WORKFLOW_INVENTORY['edit-one-field'],
      context: {
        origin: 'http://127.0.0.1:49152',
        allowedRoutes: F013_WORKFLOW_INVENTORY['edit-one-field'].routes,
        uploadFixtures: {},
      },
      runId: 'aibr_20260818T120000Z_123456abcdef',
      provider: 'anthropic',
      model: 'test-model',
      nextStepIndex: () => ++step,
      requestDecision: async () => decisions.shift(),
    })).resolves.toBe('finished')
    expect(fill).toHaveBeenCalledWith('updated')
    expect(budget.snapshot().usage).toMatchObject({
      steps: 2,
      modelCalls: 2,
      modelTokens: 26,
      browserActions: 3,
    })
    budget.dispose()
  })
})
