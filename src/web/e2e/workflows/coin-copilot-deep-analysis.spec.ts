import { expect, test } from '@playwright/test'

import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

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

test.beforeEach(async ({ page }) => {
  await installAuthenticatedSession(page)
  await installWorkflowApiMocks(page)
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

  await link.click()
  await expect(page).toHaveURL('/deep-analysis/17')
})
