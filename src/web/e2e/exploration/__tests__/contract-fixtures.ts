import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import type {
  ExplorationReport,
  IssuePublicationRequest,
  RunConfiguration,
} from '../contracts'

const repositoryRoot = resolve(process.cwd(), '..', '..')
const contractRoot = resolve(
  repositoryRoot,
  'specs',
  '360-ai-driven-browser-testing',
  'contracts',
)

export const clone = <T>(value: T): T => structuredClone(value)

export function loadContractSchemas(): Record<'config' | 'report' | 'issue', Record<string, unknown>> {
  const load = (name: string): Record<string, unknown> =>
    JSON.parse(readFileSync(resolve(contractRoot, name), 'utf8')) as Record<string, unknown>
  return {
    config: load('run-config.schema.json'),
    report: load('exploration-report.schema.json'),
    issue: load('issue-publication.schema.json'),
  }
}

export function validRunConfiguration(): RunConfiguration {
  return {
    schemaVersion: 'aurearia.browser-exploration-config/v1',
    trigger: 'local',
    provider: 'anthropic',
    model: 'claude-sonnet-5',
    workflowScope: [
      'login-session',
      'add-coin',
      'edit-one-field',
      'storage-location-change-clear',
      'tags-sets-edit',
      'image-upload-delete',
      'collection-search-filter',
      'mobile-edit',
    ],
    limits: {
      steps: 25,
      wallTimeSeconds: 900,
      modelCalls: 20,
      modelTokens: 60_000,
      browserActions: 100,
      issueAttempts: 3,
    },
    createIssues: false,
    seededDefect: null,
  }
}

export function validReport(): ExplorationReport {
  const fingerprint = 'a'.repeat(64)
  return {
    schemaVersion: 'aurearia.browser-exploration-report/v1',
    run: {
      id: 'aibr_20260918T120000Z_012345abcdef',
      trigger: 'local',
      sourceRevision: '0123456789abcdef0123456789abcdef01234567',
    },
    environment: {
      kind: 'ephemeral-full-stack',
      composeProject: 'ai-browser-contract-test',
      appOrigin: 'http://127.0.0.1:49152',
      fixtureSet: 'feature-220-f013-golden-collection',
      testUserId: 'test-user-1',
      productionDataUsed: false,
      services: {
        app: { ready: true, imageId: 'sha256:app' },
        api: { ready: true, imageId: 'sha256:api' },
        agent: { ready: true, imageId: 'sha256:agent' },
      },
    },
    provider: { name: 'fake', model: 'contract-fixture' },
    workflowScope: ['edit-one-field'],
    startedAt: '2026-09-18T12:00:00Z',
    endedAt: '2026-09-18T12:00:01Z',
    status: 'completed',
    termination: {
      reason: 'completed',
      reachedLimits: [],
      message: 'Contract fixture completed.',
      lastCompletedStepId: 'step-001',
    },
    limits: {
      steps: 25,
      wallTimeSeconds: 900,
      modelCalls: 20,
      modelTokens: 60_000,
      browserActions: 100,
      issueAttempts: 3,
    },
    usage: {
      steps: 1,
      modelCalls: 1,
      modelInputTokens: 10,
      modelOutputTokens: 5,
      modelTokens: 15,
      browserActions: 1,
      issueAttempts: 0,
      elapsedMilliseconds: 1000,
    },
    steps: [
      {
        id: 'step-001',
        index: 1,
        workflowId: 'edit-one-field',
        goal: 'Verify a synthetic edit.',
        observationEvidenceIds: ['ev_network_0001'],
        decision: { action: 'finish', rationale: 'The check is complete.' },
        actionResults: [{ action: 'finish', status: 'completed', summary: 'Finished.' }],
        usage: { inputTokens: 10, outputTokens: 5 },
        startedAt: '2026-09-18T12:00:00Z',
        endedAt: '2026-09-18T12:00:01Z',
      },
    ],
    routeHistory: [
      {
        capturedAt: '2026-09-18T12:00:00Z',
        workflowId: 'edit-one-field',
        route: '/coins/1/edit',
        reason: 'Opened the canonical edit workflow.',
      },
    ],
    evidence: [
      {
        id: 'ev_network_0001',
        kind: 'network',
        capturedAt: '2026-09-18T12:00:00Z',
        workflowId: 'edit-one-field',
        route: '/api/coins/{id}',
        summary: 'Synthetic request completed.',
        sha256: 'b'.repeat(64),
        artifactPath: 'evidence/network/ev_network_0001.json',
        truncated: false,
      },
    ],
    findings: [
      {
        id: 'finding-001',
        fingerprint,
        category: 'functional',
        severity: 'info',
        confidence: 1,
        workflowId: 'edit-one-field',
        route: '/coins/1/edit',
        title: 'Contract fixture finding',
        reproductionSteps: ['Open the edit page.'],
        expectedBehavior: 'The synthetic edit is visible.',
        observedBehavior: 'The synthetic edit is visible.',
        evidenceIds: ['ev_network_0001'],
        observedFacts: ['The request completed.'],
        modelTriage: {
          generatedByModel: true,
          summary: 'No defect observed.',
          suggestedCategory: 'functional',
          suggestedSeverity: 'info',
          confidence: 0.5,
          evidenceIds: ['ev_network_0001'],
        },
        publication: { status: 'not_requested', issueUrl: null, message: 'Disabled.' },
      },
    ],
    privacyCheck: {
      status: 'passed',
      checkedChannels: ['prompts', 'screenshots', 'report', 'traces', 'logs', 'proposedIssues'],
      canaryCount: 1,
      violations: [],
      checkedAt: '2026-09-18T12:00:01Z',
    },
    publication: { enabled: false, attempted: 0, created: 0, duplicates: 0, failed: 0 },
  }
}

export function validIssueRequest(): IssuePublicationRequest {
  const fingerprint = 'a'.repeat(64)
  return {
    schemaVersion: 'aurearia.browser-exploration-issue/v1',
    runId: 'aibr_20260918T120000Z_012345abcdef',
    findingId: 'finding-001',
    fingerprint,
    repository: 'briandenicola/Aurearia',
    title: 'Synthetic browser exploration finding',
    body: `Synthetic evidence-backed finding.\n\n<!-- aurearia-ai-finding:v1:${fingerprint} -->`,
    labels: ['ai-browser-finding', 'needs-triage'],
  }
}

// Fail fast if fixture wiring points at the wrong contract generation.
const schemas = loadContractSchemas()
if (
  (schemas.config.properties as Record<string, { const?: string }>).schemaVersion.const !==
    'aurearia.browser-exploration-config/v1' ||
  (schemas.report.properties as Record<string, { const?: string }>).schemaVersion.const !==
    'aurearia.browser-exploration-report/v1' ||
  (schemas.issue.properties as Record<string, { const?: string }>).schemaVersion.const !==
    'aurearia.browser-exploration-issue/v1'
) {
  throw new Error('Feature 360 contract fixture schema versions have drifted')
}
