import { expect, test } from '@playwright/test'

import { installAuthenticatedSession, installWorkflowApiMocks, type WorkflowApiState } from '../fixtures/workflow'

const run = {
  id: 'ccr_browser',
  threadId: 'cct_browser',
  status: 'running',
  goal: 'Attribute this coin',
  checkpointVersion: 0,
  lastSeq: 0,
  attempt: 1,
  finalAnswer: null,
  failureCode: null,
  failureMessage: null,
  resumeDeadline: null,
  usage: { iterations: 0, toolCalls: 0, inputTokens: 0, outputTokens: 0 },
  createdAt: '2030-01-01T00:00:00Z',
  updatedAt: '2030-01-01T00:00:00Z',
}

let workflowState: WorkflowApiState

test.beforeEach(async ({ page }) => {
  await installAuthenticatedSession(page)
  workflowState = await installWorkflowApiMocks(page)
})

test('mobile Coin Copilot opens the exact existing Deep Analysis review page without write controls', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.route('**/api/agent/status', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ provider: 'anthropic', configured: true }),
  }))
  await page.route('**/api/agent/copilot/capability', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      mode: 'copilot',
      enabled: true,
      modelToolCallingSupported: true,
      reason: null,
    }),
  }))
  await page.route('**/api/agent/copilot/runs', route => route.fulfill({
    status: 202,
    contentType: 'application/json',
    body: JSON.stringify({ run, reused: false }),
  }))
  await page.route('**/api/agent/copilot/runs/ccr_browser/events**', route => {
    const handoff = {
      schema_version: 1,
      operation: 'status',
      outcome: 'status',
      reason: null,
      target: { type: 'coin', id: 1, display_label: 'Golden Aureus' },
      job: {
        id: 17,
        source: 'saved_coin',
        status: 'partial',
        reused: true,
        created_at: '2030-01-01T00:00:00Z',
        completed_at: '2030-01-01T00:01:00Z',
      },
      review_url: '/deep-analysis/17',
      fresh_analysis_available: false,
      result: {
        state: 'partial',
        narrative: 'Attribution is supported with one unresolved conflict.',
        partial_success: true,
        image_only: false,
        fields: [],
        disagreements: [{ field: 'date', summary: 'Two references disagree.' }],
        unresolved_questions: [],
        coverage: [{ provider: 'nomisma', status: 'contributed' }],
        attributions: [],
        limitations: ['Reverse legend is partly unreadable.'],
      },
      truncation: {
        truncated: true,
        original_bytes: 40000,
        persisted_bytes: 32768,
        digest: 'a'.repeat(64),
        omitted_fields: 1,
        omitted_evidence: 2,
        omitted_disagreements: 3,
        omitted_questions: 4,
      },
      limitations: ['Only persisted evidence is shown.'],
    }
    const completedTool = {
      seq: 1,
      threadId: run.threadId,
      runId: run.id,
      executionId: 'cce_browser',
      type: 'tool_completed',
      ts: '2030-01-01T00:01:00Z',
      payload: {
        toolCallId: 'call_01',
        toolName: 'deep_analysis_handoff',
        stepId: 'step_01',
        status: 'succeeded',
        durationMs: 10,
        resultSummary: 'Deep Analysis result available.',
        truncated: false,
        deepAnalysisHandoffResult: handoff,
      },
    }
    const completedRun = {
      seq: 2,
      threadId: run.threadId,
      runId: run.id,
      executionId: 'cce_browser',
      type: 'run_completed',
      ts: '2030-01-01T00:01:01Z',
      payload: {
        answer: 'Open the existing review page to decide what to keep.',
        usage: { iterations: 1, toolCalls: 1, inputTokens: 10, outputTokens: 10 },
      },
    }
    return route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: [
        `event: tool_completed\nid: 1\ndata: ${JSON.stringify(completedTool)}\n\n`,
        `event: run_completed\nid: 2\ndata: ${JSON.stringify(completedRun)}\n\n`,
        'event: end\ndata: {"status":"completed","lastSeq":2}\n\n',
      ].join(''),
    })
  })

  await page.goto('/coin/1')
  await page.evaluate(() => window.dispatchEvent(new CustomEvent('open-agent-chat')))
  const input = page.getByPlaceholder("Describe the coins you're looking for...")
  await input.fill('Attribute this coin')
  await input.locator('xpath=..').getByRole('button').click()

  const card = page.getByTestId('copilot-deep-analysis-result')
  await expect(card).toContainText('Two references disagree.')
  await expect(card).toContainText('Nomisma: Contributed')
  await expect(card).toContainText('10 items omitted')
  await expect(card).not.toContainText('Apply')

  const link = card.getByRole('link', { name: 'Open Deep Analysis' })
  const linkBox = await link.boundingBox()
  const cardBox = await card.boundingBox()
  expect(linkBox?.height).toBeGreaterThanOrEqual(44)
  expect(cardBox?.x).toBeGreaterThanOrEqual(0)
  expect((cardBox?.x ?? 0) + (cardBox?.width ?? 0)).toBeLessThanOrEqual(390)

  await link.focus()
  await expect(link).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL('/deep-analysis/17')
})

test('renders the closed target, eligibility, lifecycle, and fallback matrix without conversational writes', async ({ page }) => {
  const mutations: string[] = []
  page.on('request', (request) => {
    const url = new URL(request.url())
    if (request.method() !== 'GET' &&
        (/\/api\/coins(?:\/|$)/.test(url.pathname) ||
         /\/api\/quick-capture(?:\/|$)/.test(url.pathname) ||
         /\/api\/deep-identification\/jobs\/\d+\/(?:proposal|apply)$/.test(url.pathname))) {
      mutations.push(`${request.method()} ${url.pathname}`)
    }
  })
  await page.route('**/api/agent/status', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ provider: 'anthropic', configured: true }),
  }))
  await page.route('**/api/agent/copilot/capability', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ mode: 'copilot', enabled: true, modelToolCallingSupported: true, reason: null }),
  }))
  await page.route('**/api/agent/copilot/runs', route => route.fulfill({
    status: 202,
    contentType: 'application/json',
    body: JSON.stringify({ run, reused: false }),
  }))

  const results = [
    {
      operation: 'request', outcome: 'accepted', reason: null,
      target: { type: 'coin', id: 1, display_label: 'Collection coin' },
      job: { id: 21, source: 'saved_coin', status: 'queued', reused: false },
      review_url: '/deep-analysis/21',
    },
    {
      operation: 'request', outcome: 'accepted', reason: null,
      target: { type: 'coin', id: 2, display_label: 'Wishlist coin' },
      job: { id: 22, source: 'saved_coin', status: 'running', reused: false },
      review_url: '/deep-analysis/22',
    },
    {
      operation: 'request', outcome: 'accepted', reason: null,
      target: { type: 'draft', id: 3, display_label: 'Active draft' },
      job: { id: 23, source: 'copilot_draft', status: 'queued', reused: false },
      review_url: '/deep-analysis/23',
    },
    {
      operation: 'status', outcome: 'reused_active', reason: null,
      target: { type: 'coin', id: 1, display_label: 'Collection coin' },
      job: { id: 24, source: 'saved_coin', status: 'running', reused: true },
      review_url: '/deep-analysis/24',
    },
    {
      operation: 'status', outcome: 'reused_result', reason: null,
      target: { type: 'coin', id: 1, display_label: 'Collection coin' },
      job: { id: 25, source: 'saved_coin', status: 'completed', reused: true },
      review_url: '/deep-analysis/25',
      result: {
        state: 'complete', narrative: 'Retained result.', partial_success: false, image_only: false,
        fields: [], disagreements: [], unresolved_questions: [], coverage: [], attributions: [], limitations: [],
      },
    },
    {
      operation: 'status', outcome: 'retry_available', reason: 'result_missing',
      target: { type: 'draft', id: 3, display_label: 'Active draft' },
      job: { id: 26, source: 'copilot_draft', status: 'failed', reused: true },
      review_url: '/deep-analysis/26',
    },
    {
      operation: 'request', outcome: 'missing_images', reason: 'missing_reverse',
      target: { type: 'draft', id: 3, display_label: 'Active draft' },
    },
    { operation: 'status', outcome: 'not_eligible', reason: null },
    { operation: 'status', outcome: 'target_unavailable', reason: null },
    { operation: 'request', outcome: 'unavailable', reason: 'attribution_disabled' },
    {
      operation: 'status', outcome: 'cancelled', reason: 'cancelled',
      target: { type: 'coin', id: 1, display_label: 'Collection coin' },
      job: { id: 27, source: 'saved_coin', status: 'cancelled', reused: true },
      review_url: '/deep-analysis/27',
    },
  ].map((result) => {
    if (result.outcome === 'not_eligible' || result.outcome === 'target_unavailable') {
      return { outcome: result.outcome, reason: null }
    }
    return {
      schema_version: 1,
      fresh_analysis_available: false,
      limitations: [],
      ...result,
      ...('job' in result && result.job
        ? {
            job: {
              ...result.job,
              created_at: '2030-01-01T00:00:00Z',
              completed_at: ['completed', 'failed', 'cancelled'].includes(result.job.status)
                ? '2030-01-01T00:01:00Z'
                : null,
            },
          }
        : {}),
    }
  })

  await page.route('**/api/agent/copilot/runs/ccr_browser/events**', route => {
    const frames = results.map((result, index) => {
      const event = {
        seq: index + 1,
        threadId: run.threadId,
        runId: run.id,
        executionId: 'cce_matrix',
        type: 'tool_completed',
        ts: '2030-01-01T00:01:00Z',
        payload: {
          toolCallId: `call_${index}`,
          toolName: 'deep_analysis_handoff',
          stepId: `step_${index}`,
          status: 'succeeded',
          durationMs: 10,
          resultSummary: 'Deep Analysis handoff.',
          truncated: false,
          deepAnalysisHandoffResult: result,
        },
      }
      return `event: tool_completed\nid: ${index + 1}\ndata: ${JSON.stringify(event)}\n\n`
    })
    const completed = {
      seq: results.length + 1,
      threadId: run.threadId,
      runId: run.id,
      executionId: 'cce_matrix',
      type: 'run_completed',
      ts: '2030-01-01T00:01:01Z',
      payload: {
        answer: 'Review the results in Deep Analysis.',
        usage: { iterations: 1, toolCalls: results.length, inputTokens: 10, outputTokens: 10 },
      },
    }
    frames.push(`event: run_completed\nid: ${completed.seq}\ndata: ${JSON.stringify(completed)}\n\n`)
    frames.push(`event: end\ndata: {"status":"completed","lastSeq":${completed.seq}}\n\n`)
    return route.fulfill({ status: 200, contentType: 'text/event-stream', body: frames.join('') })
  })

  await page.goto('/coin/1')
  await page.evaluate(() => window.dispatchEvent(new CustomEvent('open-agent-chat')))
  const input = page.getByPlaceholder("Describe the coins you're looking for...")
  await input.fill('Check every safe Deep Analysis state')
  await input.locator('xpath=..').getByRole('button').click()

  const cards = page.getByTestId('copilot-deep-analysis-result')
  await expect(cards).toHaveCount(results.length)
  const progress = page.getByTestId('copilot-run-progress')
  for (const text of [
    'Collection coin', 'Wishlist coin', 'Active draft', 'Accepted', 'In progress',
    'Result available', 'Retry available', 'Images required', 'Not eligible',
    'Target unavailable', 'Unavailable', 'Cancelled',
  ]) {
    await expect(progress).toContainText(text)
  }
  await expect(progress).not.toContainText('Apply')
  expect(mutations).toEqual([])
})

test('restores Coin Copilot and Deep Analysis SSE cursors independently after a mobile background session', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const copilotEventURLs: string[] = []
  const deepEventURLs: string[] = []
  const restoredRun = { ...run, status: 'running', lastSeq: 5 }
  await page.route('**/api/agent/status', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ provider: 'anthropic', configured: true }),
  }))
  await page.route('**/api/agent/copilot/capability', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ mode: 'copilot', enabled: true, modelToolCallingSupported: true, reason: null }),
  }))
  await page.route('**/api/agent/copilot/runs/ccr_browser', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ run: restoredRun }),
  }))
  await page.route('**/api/agent/copilot/threads/cct_browser', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      thread: {
        id: 'cct_browser',
        title: 'Restored attribution',
        messages: [{ role: 'user', content: 'Restore this run', runId: 'ccr_browser', createdAt: '2030-01-01T00:00:00Z' }],
        runs: [restoredRun],
        createdAt: '2030-01-01T00:00:00Z',
        updatedAt: '2030-01-01T00:00:00Z',
      },
    }),
  }))
  await page.route('**/api/agent/copilot/runs/ccr_browser/events**', route => {
    copilotEventURLs.push(route.request().url())
    const completed = {
      seq: 5, threadId: run.threadId, runId: run.id, executionId: 'cce_restore',
      type: 'run_completed', ts: '2030-01-01T00:01:01Z',
      payload: {
        answer: 'Restored Coin Copilot.',
        usage: { iterations: 1, toolCalls: 1, inputTokens: 10, outputTokens: 10 },
      },
    }
    return route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: `event: run_completed\nid: 5\ndata: ${JSON.stringify(completed)}\n\nevent: end\ndata: {"status":"completed","lastSeq":5}\n\n`,
    })
  })

  await page.goto('/coin/1')
  await page.evaluate(() => {
    sessionStorage.setItem('coinCopilot:activeRun', JSON.stringify({
      runId: 'ccr_browser',
      threadId: 'cct_browser',
      lastSeq: 4,
    }))
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    document.dispatchEvent(new Event('visibilitychange'))
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    document.dispatchEvent(new Event('visibilitychange'))
    window.dispatchEvent(new CustomEvent('open-agent-chat'))
  })
  await expect(page.getByText('Complete')).toBeVisible()
  expect(copilotEventURLs).toHaveLength(1)
  expect(new URL(copilotEventURLs[0]!).searchParams.get('since')).toBe('4')

  workflowState.deepIdentificationJobs.push({
    id: 17,
    notes: 'restored',
    providers: 'image',
    status: 'running',
    source: 'saved_coin',
  })
  await page.route('**/api/deep-identification/jobs/17', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      job: {
        id: 17,
        source: 'saved_coin',
        status: 'running',
        partialSuccess: false,
        cancelRequested: false,
        lastSeq: 43,
        eventsAvailable: true,
        appliedAt: null,
        appliedCoinId: null,
        appliedDraftId: null,
        expiresAt: '2030-01-01T00:00:00Z',
        createdAt: '2030-01-01T00:00:00Z',
      },
      report: null,
      proposal: null,
    }),
  }))
  await page.route('**/api/deep-identification/jobs/17/events**', async (route) => {
    deepEventURLs.push(route.request().url())
    const terminal = {
      seq: 43, jobId: 17, type: 'terminal', ts: '2030-01-01T00:02:00Z',
      payload: { status: 'completed', partialSuccess: false, hasReport: true, hasProposal: true },
    }
    await route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: `id: 43\nevent: terminal\ndata: ${JSON.stringify(terminal)}\n\n`,
    })
  })
  await page.evaluate(() => sessionStorage.setItem('deep-identification-last-seq:17', '42'))
  await page.goto('/deep-analysis/17')
  await expect.poll(() => deepEventURLs.length).toBe(1)
  expect(new URL(deepEventURLs[0]!).searchParams.get('since')).toBe('42')
})
