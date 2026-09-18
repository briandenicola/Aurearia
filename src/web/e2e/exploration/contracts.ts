/**
 * Feature 360 v1 contract types and dependency-free runtime guards.
 *
 * The constants and bounds mirror the checked-in JSON Schemas. Contract tests
 * load those schemas directly so a schema change cannot silently leave these
 * guards behind.
 */

export const CONFIG_SCHEMA_VERSION = 'aurearia.browser-exploration-config/v1' as const
export const REPORT_SCHEMA_VERSION = 'aurearia.browser-exploration-report/v1' as const
export const ISSUE_SCHEMA_VERSION = 'aurearia.browser-exploration-issue/v1' as const

export const WORKFLOW_IDS = [
  'login-session',
  'add-coin',
  'edit-one-field',
  'storage-location-change-clear',
  'tags-sets-edit',
  'image-upload-delete',
  'collection-search-filter',
  'mobile-edit',
] as const

export type WorkflowId = (typeof WORKFLOW_IDS)[number]
export type Trigger = 'local' | 'manual' | 'nightly'
export type ProviderName = 'anthropic' | 'ollama'
export type ReportProviderName = ProviderName | 'fake'
export type EvidenceKind = 'route' | 'screenshot' | 'console' | 'network' | 'accessibility' | 'ui'
export type FindingCategory =
  | 'functional'
  | 'console'
  | 'network'
  | 'accessibility'
  | 'visual'
  | 'navigation'
  | 'performance'
  | 'security'
  | 'other'
export type FindingSeverity = 'critical' | 'high' | 'medium' | 'low' | 'info'
export type ActionName =
  | 'navigate'
  | 'click'
  | 'fill'
  | 'select'
  | 'upload_fixture'
  | 'set_viewport'
  | 'back'
  | 'wait_for_ui'
  | 'checkpoint'
  | 'finish'

export interface ExplorationLimits {
  steps: number
  wallTimeSeconds: number
  modelCalls: number
  modelTokens: number
  browserActions: number
  issueAttempts: number
}

export interface RunConfiguration {
  schemaVersion: typeof CONFIG_SCHEMA_VERSION
  trigger: Trigger
  provider: ProviderName
  model: string
  workflowScope: WorkflowId[]
  limits: ExplorationLimits
  createIssues: boolean
  seededDefect: 'edit-save-http-500-v1' | null
}

export interface ExplorationUsage {
  steps: number
  modelCalls: number
  modelInputTokens: number
  modelOutputTokens: number
  modelTokens: number
  browserActions: number
  issueAttempts: number
  elapsedMilliseconds: number
}

export interface ModelUsage {
  inputTokens: number
  outputTokens: number
}

export interface BrowserEvidence {
  id: string
  kind: EvidenceKind
  capturedAt: string
  workflowId: WorkflowId
  route: string
  summary: string
  sha256: string
  artifactPath: string | null
  truncated: boolean
}

export interface ModelTriage {
  generatedByModel: true
  summary: string
  suggestedCategory: FindingCategory
  suggestedSeverity: FindingSeverity
  confidence: number
  evidenceIds: string[]
}

export type FindingPublicationStatus =
  | 'not_requested'
  | 'not_actionable'
  | 'queued'
  | 'duplicate'
  | 'created'
  | 'failed'
  | 'blocked_limit'
  | 'blocked_privacy'

export interface ExplorationFinding {
  id: string
  fingerprint: string
  category: FindingCategory
  severity: FindingSeverity
  confidence: number
  workflowId: WorkflowId
  route: string
  title: string
  reproductionSteps: string[]
  expectedBehavior: string
  observedBehavior: string
  evidenceIds: string[]
  observedFacts: string[]
  modelTriage: ModelTriage | null
  publication: {
    status: FindingPublicationStatus
    issueUrl: string | null
    message: string
  }
}

export type TerminationReason =
  | 'completed'
  | 'step_limit'
  | 'wall_time_limit'
  | 'model_call_limit'
  | 'model_token_limit'
  | 'browser_action_limit'
  | 'issue_attempt_limit'
  | 'provider_unavailable'
  | 'provider_usage_invalid'
  | 'model_output_invalid'
  | 'browser_failure'
  | 'stack_failure'
  | 'privacy_failure'
  | 'cancelled'
  | 'internal_error'

export interface ExplorationReport {
  schemaVersion: typeof REPORT_SCHEMA_VERSION
  run: { id: string; trigger: Trigger; sourceRevision: string }
  environment: {
    kind: 'ephemeral-full-stack'
    composeProject: string
    appOrigin: string
    fixtureSet: 'feature-220-f013-golden-collection'
    testUserId: string
    productionDataUsed: false
    services: Record<'app' | 'api' | 'agent', { ready: boolean; imageId: string }>
  }
  provider: { name: ReportProviderName; model: string }
  workflowScope: WorkflowId[]
  startedAt: string
  endedAt: string
  status: 'completed' | 'bounded' | 'failed' | 'privacy_failed' | 'infrastructure_failed'
  termination: {
    reason: TerminationReason
    reachedLimits: Array<keyof ExplorationLimits>
    message: string
    lastCompletedStepId: string | null
  }
  limits: ExplorationLimits
  usage: ExplorationUsage
  steps: Array<{
    id: string
    index: number
    workflowId: WorkflowId
    goal: string
    observationEvidenceIds: string[]
    decision: { action: ActionName; rationale: string } | null
    actionResults: Array<{
      action: string
      status: 'completed' | 'failed' | 'refused' | 'cancelled'
      summary: string
    }>
    usage: ModelUsage | null
    startedAt: string
    endedAt: string
  }>
  routeHistory: Array<{
    capturedAt: string
    workflowId: WorkflowId
    route: string
    reason: string
  }>
  evidence: BrowserEvidence[]
  findings: ExplorationFinding[]
  privacyCheck: {
    status: 'passed' | 'failed'
    checkedChannels: Array<
      'prompts' | 'screenshots' | 'report' | 'traces' | 'logs' | 'proposedIssues'
    >
    canaryCount: number
    violations: Array<{ channel: string; path: string; rule: string }>
    checkedAt: string
  }
  publication: {
    enabled: boolean
    attempted: number
    created: number
    duplicates: number
    failed: number
  }
}

export interface IssuePublicationRequest {
  schemaVersion: typeof ISSUE_SCHEMA_VERSION
  runId: string
  findingId: string
  fingerprint: string
  repository: string
  title: string
  body: string
  labels: Array<'ai-browser-finding' | 'needs-triage' | 'accessibility'>
}

export interface ContractValidation {
  ok: boolean
  errors: string[]
}

type JsonObject = Record<string, unknown>

const actionNames: readonly ActionName[] = [
  'navigate',
  'click',
  'fill',
  'select',
  'upload_fixture',
  'set_viewport',
  'back',
  'wait_for_ui',
  'checkpoint',
  'finish',
]
const categories: readonly FindingCategory[] = [
  'functional',
  'console',
  'network',
  'accessibility',
  'visual',
  'navigation',
  'performance',
  'security',
  'other',
]
const severities: readonly FindingSeverity[] = ['critical', 'high', 'medium', 'low', 'info']
const terminationReasons: readonly TerminationReason[] = [
  'completed',
  'step_limit',
  'wall_time_limit',
  'model_call_limit',
  'model_token_limit',
  'browser_action_limit',
  'issue_attempt_limit',
  'provider_unavailable',
  'provider_usage_invalid',
  'model_output_invalid',
  'browser_failure',
  'stack_failure',
  'privacy_failure',
  'cancelled',
  'internal_error',
]
const limitKeys: ReadonlyArray<keyof ExplorationLimits> = [
  'steps',
  'wallTimeSeconds',
  'modelCalls',
  'modelTokens',
  'browserActions',
  'issueAttempts',
]
const limitMaximums: ExplorationLimits = {
  steps: 25,
  wallTimeSeconds: 900,
  modelCalls: 20,
  modelTokens: 60_000,
  browserActions: 100,
  issueAttempts: 3,
}
const runIdPattern = /^aibr_[0-9]{8}T[0-9]{6}Z_[a-f0-9]{12}$/
const stepIdPattern = /^step-[0-9]{3}$/
const evidenceIdPattern = /^ev_[a-z]+_[0-9]{4}$/
const findingIdPattern = /^finding-[0-9]{3}$/
const sha256Pattern = /^[a-f0-9]{64}$/
const secretKeyPattern = /(api.?key|authorization|cookie|credential|password|secret|token)$/i
const secretValuePattern = /(sk-ant-[A-Za-z0-9_-]+|gh[pousr]_[A-Za-z0-9_-]+|bearer\s+\S+|(?:api.?key|token|password|secret)\s*[:=]\s*\S+)/i

const isObject = (value: unknown): value is JsonObject =>
  typeof value === 'object' && value !== null && !Array.isArray(value)
const isString = (value: unknown, min = 0, max = Number.MAX_SAFE_INTEGER): value is string =>
  typeof value === 'string' && value.length >= min && value.length <= max
const isInteger = (value: unknown, min: number, max: number): value is number =>
  Number.isInteger(value) && (value as number) >= min && (value as number) <= max
const isNumber = (value: unknown, min: number, max: number): value is number =>
  typeof value === 'number' && Number.isFinite(value) && value >= min && value <= max
const isEnum = <T extends string>(value: unknown, values: readonly T[]): value is T =>
  typeof value === 'string' && values.includes(value as T)
const isUniqueStrings = (value: unknown): value is string[] =>
  Array.isArray(value) &&
  value.every((item) => typeof item === 'string') &&
  new Set(value).size === value.length

function exactKeys(value: unknown, expected: readonly string[], path: string, errors: string[]): value is JsonObject {
  if (!isObject(value)) {
    errors.push(`${path} must be an object`)
    return false
  }
  const actual = Object.keys(value)
  for (const key of expected) {
    if (!Object.hasOwn(value, key)) errors.push(`${path}.${key} is required`)
  }
  for (const key of actual) {
    if (!expected.includes(key)) errors.push(`${path}.${key} is not allowed`)
  }
  return true
}

function hasSecret(value: unknown): boolean {
  if (typeof value === 'string') return secretValuePattern.test(value)
  if (Array.isArray(value)) return value.some(hasSecret)
  if (!isObject(value)) return false
  return Object.entries(value).some(([key, child]) => secretKeyPattern.test(key) || hasSecret(child))
}

function validateLimits(value: unknown, path: string, errors: string[]): value is ExplorationLimits {
  if (!exactKeys(value, limitKeys, path, errors)) return false
  for (const key of limitKeys) {
    const minimum = key === 'issueAttempts' ? 0 : 1
    if (!isInteger(value[key], minimum, limitMaximums[key])) {
      errors.push(`${path}.${key} must be an integer from ${minimum} to ${limitMaximums[key]}`)
    }
  }
  return errors.length === 0
}

export function validateRunConfiguration(value: unknown): ContractValidation {
  const errors: string[] = []
  const keys = [
    'schemaVersion',
    'trigger',
    'provider',
    'model',
    'workflowScope',
    'limits',
    'createIssues',
    'seededDefect',
  ]
  if (!exactKeys(value, keys, '$', errors)) return { ok: false, errors }
  if (value.schemaVersion !== CONFIG_SCHEMA_VERSION) errors.push('$.schemaVersion is invalid')
  if (!isEnum(value.trigger, ['local', 'manual', 'nightly'])) errors.push('$.trigger is invalid')
  if (!isEnum(value.provider, ['anthropic', 'ollama'])) errors.push('$.provider is invalid')
  if (!isString(value.model, 1, 120) || /https?:\/\//i.test(value.model)) {
    errors.push('$.model must be a bounded provider model identifier, not a URL')
  }
  if (
    !isUniqueStrings(value.workflowScope) ||
    value.workflowScope.length === 0 ||
    !value.workflowScope.every((item) => isEnum(item, WORKFLOW_IDS))
  ) {
    errors.push('$.workflowScope must contain unique supported workflows')
  }
  validateLimits(value.limits, '$.limits', errors)
  if (typeof value.createIssues !== 'boolean') errors.push('$.createIssues must be boolean')
  if (value.seededDefect !== null && value.seededDefect !== 'edit-save-http-500-v1') {
    errors.push('$.seededDefect is invalid')
  }
  if (hasSecret(value)) errors.push('$ must not contain secret-shaped fields or values')
  return { ok: errors.length === 0, errors }
}

function validateTimestamp(value: unknown, path: string, errors: string[]): value is string {
  if (!isString(value) || Number.isNaN(Date.parse(value))) {
    errors.push(`${path} must be an RFC 3339 timestamp`)
    return false
  }
  return true
}

function validateUsage(value: unknown, limits: ExplorationLimits | undefined, errors: string[]): void {
  const keys = [
    'steps',
    'modelCalls',
    'modelInputTokens',
    'modelOutputTokens',
    'modelTokens',
    'browserActions',
    'issueAttempts',
    'elapsedMilliseconds',
  ]
  if (!exactKeys(value, keys, '$.usage', errors)) return
  const maximums: Record<string, number> = {
    steps: limits?.steps ?? 25,
    modelCalls: limits?.modelCalls ?? 20,
    modelInputTokens: limits?.modelTokens ?? 60_000,
    modelOutputTokens: limits?.modelTokens ?? 60_000,
    modelTokens: limits?.modelTokens ?? 60_000,
    browserActions: limits?.browserActions ?? 100,
    issueAttempts: limits?.issueAttempts ?? 3,
    elapsedMilliseconds: (limits?.wallTimeSeconds ?? 900) * 1000,
  }
  for (const key of keys) {
    if (!isInteger(value[key], 0, maximums[key] ?? 0)) {
      errors.push(`$.usage.${key} exceeds its snapshotted limit or is invalid`)
    }
  }
  if (
    typeof value.modelTokens === 'number' &&
    typeof value.modelInputTokens === 'number' &&
    typeof value.modelOutputTokens === 'number' &&
    value.modelTokens !== value.modelInputTokens + value.modelOutputTokens
  ) {
    errors.push('$.usage.modelTokens must equal input plus output tokens')
  }
}

function validateEvidence(value: unknown, index: number, errors: string[]): void {
  const path = `$.evidence[${index}]`
  const keys = [
    'id',
    'kind',
    'capturedAt',
    'workflowId',
    'route',
    'summary',
    'sha256',
    'artifactPath',
    'truncated',
  ]
  if (!exactKeys(value, keys, path, errors)) return
  if (!isString(value.id) || !evidenceIdPattern.test(value.id)) errors.push(`${path}.id is invalid`)
  if (!isEnum(value.kind, ['route', 'screenshot', 'console', 'network', 'accessibility', 'ui'])) {
    errors.push(`${path}.kind is invalid`)
  }
  validateTimestamp(value.capturedAt, `${path}.capturedAt`, errors)
  if (!isEnum(value.workflowId, WORKFLOW_IDS)) errors.push(`${path}.workflowId is invalid`)
  if (!isString(value.route, 1, 200) || !value.route.startsWith('/')) errors.push(`${path}.route is invalid`)
  if (!isString(value.summary, 0, 2000)) errors.push(`${path}.summary is invalid`)
  if (!isString(value.sha256) || !sha256Pattern.test(value.sha256)) errors.push(`${path}.sha256 is invalid`)
  if (value.artifactPath !== null) {
    if (
      !isString(value.artifactPath, 1, 500) ||
      !/^evidence\/[A-Za-z0-9._/-]+$/.test(value.artifactPath) ||
      value.artifactPath.split('/').includes('..')
    ) {
      errors.push(`${path}.artifactPath must stay beneath evidence/`)
    }
  }
  if (typeof value.truncated !== 'boolean') errors.push(`${path}.truncated must be boolean`)
}

function validateTriage(value: unknown, path: string, errors: string[]): void {
  const keys = [
    'generatedByModel',
    'summary',
    'suggestedCategory',
    'suggestedSeverity',
    'confidence',
    'evidenceIds',
  ]
  if (!exactKeys(value, keys, path, errors)) return
  if (value.generatedByModel !== true) errors.push(`${path}.generatedByModel must be true`)
  if (!isString(value.summary, 0, 2000)) errors.push(`${path}.summary is invalid`)
  if (!isEnum(value.suggestedCategory, categories)) errors.push(`${path}.suggestedCategory is invalid`)
  if (!isEnum(value.suggestedSeverity, severities)) errors.push(`${path}.suggestedSeverity is invalid`)
  if (!isNumber(value.confidence, 0, 1)) errors.push(`${path}.confidence is invalid`)
  if (
    !isUniqueStrings(value.evidenceIds) ||
    value.evidenceIds.length === 0 ||
    !value.evidenceIds.every((id) => evidenceIdPattern.test(id))
  ) {
    errors.push(`${path}.evidenceIds is invalid`)
  }
}

function validateFinding(value: unknown, index: number, evidenceIds: Set<string>, errors: string[]): void {
  const path = `$.findings[${index}]`
  const keys = [
    'id',
    'fingerprint',
    'category',
    'severity',
    'confidence',
    'workflowId',
    'route',
    'title',
    'reproductionSteps',
    'expectedBehavior',
    'observedBehavior',
    'evidenceIds',
    'observedFacts',
    'modelTriage',
    'publication',
  ]
  if (!exactKeys(value, keys, path, errors)) return
  if (!isString(value.id) || !findingIdPattern.test(value.id)) errors.push(`${path}.id is invalid`)
  if (!isString(value.fingerprint) || !sha256Pattern.test(value.fingerprint)) {
    errors.push(`${path}.fingerprint is invalid`)
  }
  if (!isEnum(value.category, categories)) errors.push(`${path}.category is invalid`)
  if (!isEnum(value.severity, severities)) errors.push(`${path}.severity is invalid`)
  if (!isNumber(value.confidence, 0, 1)) errors.push(`${path}.confidence is invalid`)
  if (!isEnum(value.workflowId, WORKFLOW_IDS)) errors.push(`${path}.workflowId is invalid`)
  if (!isString(value.route, 1, 200) || !value.route.startsWith('/')) errors.push(`${path}.route is invalid`)
  if (!isString(value.title, 1, 200)) errors.push(`${path}.title is invalid`)
  if (
    !Array.isArray(value.reproductionSteps) ||
    value.reproductionSteps.length < 1 ||
    value.reproductionSteps.length > 12 ||
    !value.reproductionSteps.every((item) => isString(item, 0, 500))
  ) {
    errors.push(`${path}.reproductionSteps is invalid`)
  }
  if (!isString(value.expectedBehavior, 1, 2000)) errors.push(`${path}.expectedBehavior is invalid`)
  if (!isString(value.observedBehavior, 1, 2000)) errors.push(`${path}.observedBehavior is invalid`)
  if (
    !isUniqueStrings(value.evidenceIds) ||
    value.evidenceIds.length < 1 ||
    value.evidenceIds.length > 50 ||
    !value.evidenceIds.every((id) => evidenceIds.has(id))
  ) {
    errors.push(`${path}.evidenceIds must reference captured evidence`)
  }
  if (
    !Array.isArray(value.observedFacts) ||
    value.observedFacts.length < 1 ||
    value.observedFacts.length > 20 ||
    !value.observedFacts.every((item) => isString(item, 0, 1000))
  ) {
    errors.push(`${path}.observedFacts is invalid`)
  }
  if (value.modelTriage !== null) {
    validateTriage(value.modelTriage, `${path}.modelTriage`, errors)
    if (
      isObject(value.modelTriage) &&
      Array.isArray(value.modelTriage.evidenceIds) &&
      value.modelTriage.evidenceIds.some((id) => typeof id !== 'string' || !evidenceIds.has(id))
    ) {
      errors.push(`${path}.modelTriage references missing evidence`)
    }
  }
  const publicationKeys = ['status', 'issueUrl', 'message']
  if (exactKeys(value.publication, publicationKeys, `${path}.publication`, errors)) {
    if (
      !isEnum(value.publication.status, [
        'not_requested',
        'not_actionable',
        'queued',
        'duplicate',
        'created',
        'failed',
        'blocked_limit',
        'blocked_privacy',
      ])
    ) {
      errors.push(`${path}.publication.status is invalid`)
    }
    if (
      value.publication.issueUrl !== null &&
      (!isString(value.publication.issueUrl, 1, 500) || !/^https?:\/\//.test(value.publication.issueUrl))
    ) {
      errors.push(`${path}.publication.issueUrl is invalid`)
    }
    if (!isString(value.publication.message, 0, 1000)) errors.push(`${path}.publication.message is invalid`)
  }
}

export function validateExplorationReport(value: unknown): ContractValidation {
  const errors: string[] = []
  const keys = [
    'schemaVersion',
    'run',
    'environment',
    'provider',
    'workflowScope',
    'startedAt',
    'endedAt',
    'status',
    'termination',
    'limits',
    'usage',
    'steps',
    'routeHistory',
    'evidence',
    'findings',
    'privacyCheck',
    'publication',
  ]
  if (!exactKeys(value, keys, '$', errors)) return { ok: false, errors }
  if (value.schemaVersion !== REPORT_SCHEMA_VERSION) errors.push('$.schemaVersion is invalid')

  if (exactKeys(value.run, ['id', 'trigger', 'sourceRevision'], '$.run', errors)) {
    if (!isString(value.run.id) || !runIdPattern.test(value.run.id)) errors.push('$.run.id is invalid')
    if (!isEnum(value.run.trigger, ['local', 'manual', 'nightly'])) errors.push('$.run.trigger is invalid')
    if (!isString(value.run.sourceRevision) || !/^[a-fA-F0-9]{40}$/.test(value.run.sourceRevision)) {
      errors.push('$.run.sourceRevision is invalid')
    }
  }

  const environmentKeys = [
    'kind',
    'composeProject',
    'appOrigin',
    'fixtureSet',
    'testUserId',
    'productionDataUsed',
    'services',
  ]
  if (exactKeys(value.environment, environmentKeys, '$.environment', errors)) {
    if (value.environment.kind !== 'ephemeral-full-stack') errors.push('$.environment.kind is invalid')
    if (
      !isString(value.environment.composeProject, 1, 80) ||
      !/^ai-browser-[a-z0-9-]+$/.test(value.environment.composeProject)
    ) {
      errors.push('$.environment.composeProject is invalid')
    }
    if (
      !isString(value.environment.appOrigin) ||
      !/^http:\/\/(127\.0\.0\.1|localhost):[0-9]{1,5}$/.test(value.environment.appOrigin)
    ) {
      errors.push('$.environment.appOrigin must be loopback')
    }
    if (value.environment.fixtureSet !== 'feature-220-f013-golden-collection') {
      errors.push('$.environment.fixtureSet is invalid')
    }
    if (!isString(value.environment.testUserId, 1, 100)) errors.push('$.environment.testUserId is invalid')
    if (value.environment.productionDataUsed !== false) {
      errors.push('$.environment.productionDataUsed must be false')
    }
    if (exactKeys(value.environment.services, ['app', 'api', 'agent'], '$.environment.services', errors)) {
      for (const service of ['app', 'api', 'agent']) {
        const detail = value.environment.services[service]
        if (exactKeys(detail, ['ready', 'imageId'], `$.environment.services.${service}`, errors)) {
          if (typeof detail.ready !== 'boolean') errors.push(`$.environment.services.${service}.ready is invalid`)
          if (!isString(detail.imageId, 0, 200)) errors.push(`$.environment.services.${service}.imageId is invalid`)
        }
      }
    }
  }

  if (exactKeys(value.provider, ['name', 'model'], '$.provider', errors)) {
    if (!isEnum(value.provider.name, ['anthropic', 'ollama', 'fake'])) errors.push('$.provider.name is invalid')
    if (!isString(value.provider.model, 1, 120)) errors.push('$.provider.model is invalid')
  }
  if (
    !isUniqueStrings(value.workflowScope) ||
    value.workflowScope.length < 1 ||
    !value.workflowScope.every((id) => isEnum(id, WORKFLOW_IDS))
  ) {
    errors.push('$.workflowScope is invalid')
  }

  const startedAt = value.startedAt
  const endedAt = value.endedAt
  const startValid = validateTimestamp(startedAt, '$.startedAt', errors)
  const endValid = validateTimestamp(endedAt, '$.endedAt', errors)
  if (startValid && endValid && Date.parse(endedAt) < Date.parse(startedAt)) {
    errors.push('$.endedAt must not precede $.startedAt')
  }
  if (!isEnum(value.status, ['completed', 'bounded', 'failed', 'privacy_failed', 'infrastructure_failed'])) {
    errors.push('$.status is invalid')
  }

  let limits: ExplorationLimits | undefined
  const priorErrorCount = errors.length
  validateLimits(value.limits, '$.limits', errors)
  if (errors.length === priorErrorCount) limits = value.limits as unknown as ExplorationLimits
  validateUsage(value.usage, limits, errors)

  if (
    exactKeys(
      value.termination,
      ['reason', 'reachedLimits', 'message', 'lastCompletedStepId'],
      '$.termination',
      errors,
    )
  ) {
    if (!isEnum(value.termination.reason, terminationReasons)) errors.push('$.termination.reason is invalid')
    if (
      !isUniqueStrings(value.termination.reachedLimits) ||
      !value.termination.reachedLimits.every((key) => limitKeys.includes(key as keyof ExplorationLimits))
    ) {
      errors.push('$.termination.reachedLimits is invalid')
    }
    if (!isString(value.termination.message, 0, 1000)) errors.push('$.termination.message is invalid')
    if (
      value.termination.lastCompletedStepId !== null &&
      (!isString(value.termination.lastCompletedStepId) ||
        !stepIdPattern.test(value.termination.lastCompletedStepId))
    ) {
      errors.push('$.termination.lastCompletedStepId is invalid')
    }
  }

  const evidenceIds = new Set<string>()
  if (!Array.isArray(value.evidence) || value.evidence.length > 1000) {
    errors.push('$.evidence is invalid')
  } else {
    value.evidence.forEach((item, index) => {
      validateEvidence(item, index, errors)
      if (isObject(item) && typeof item.id === 'string') evidenceIds.add(item.id)
    })
  }

  if (!Array.isArray(value.steps) || value.steps.length > 25) {
    errors.push('$.steps is invalid')
  } else {
    value.steps.forEach((step, index) => {
      const path = `$.steps[${index}]`
      const stepKeys = [
        'id',
        'index',
        'workflowId',
        'goal',
        'observationEvidenceIds',
        'decision',
        'actionResults',
        'usage',
        'startedAt',
        'endedAt',
      ]
      if (!exactKeys(step, stepKeys, path, errors)) return
      if (!isString(step.id) || !stepIdPattern.test(step.id)) errors.push(`${path}.id is invalid`)
      if (!isInteger(step.index, 1, 25)) errors.push(`${path}.index is invalid`)
      if (!isEnum(step.workflowId, WORKFLOW_IDS)) errors.push(`${path}.workflowId is invalid`)
      if (!isString(step.goal, 1, 1000)) errors.push(`${path}.goal is invalid`)
      if (
        !isUniqueStrings(step.observationEvidenceIds) ||
        !step.observationEvidenceIds.every((id) => evidenceIds.has(id))
      ) {
        errors.push(`${path}.observationEvidenceIds references missing evidence`)
      }
      if (step.decision !== null) {
        if (exactKeys(step.decision, ['action', 'rationale'], `${path}.decision`, errors)) {
          if (!isEnum(step.decision.action, actionNames)) errors.push(`${path}.decision.action is invalid`)
          if (!isString(step.decision.rationale, 0, 1000)) errors.push(`${path}.decision.rationale is invalid`)
        }
      }
      if (!Array.isArray(step.actionResults) || step.actionResults.length > 100) {
        errors.push(`${path}.actionResults is invalid`)
      } else {
        step.actionResults.forEach((result, resultIndex) => {
          const resultPath = `${path}.actionResults[${resultIndex}]`
          if (exactKeys(result, ['action', 'status', 'summary'], resultPath, errors)) {
            if (!isString(result.action, 0, 40)) errors.push(`${resultPath}.action is invalid`)
            if (!isEnum(result.status, ['completed', 'failed', 'refused', 'cancelled'])) {
              errors.push(`${resultPath}.status is invalid`)
            }
            if (!isString(result.summary, 0, 1000)) errors.push(`${resultPath}.summary is invalid`)
          }
        })
      }
      if (step.usage !== null) {
        if (exactKeys(step.usage, ['inputTokens', 'outputTokens'], `${path}.usage`, errors)) {
          if (!isInteger(step.usage.inputTokens, 0, Number.MAX_SAFE_INTEGER)) {
            errors.push(`${path}.usage.inputTokens is invalid`)
          }
          if (!isInteger(step.usage.outputTokens, 0, Number.MAX_SAFE_INTEGER)) {
            errors.push(`${path}.usage.outputTokens is invalid`)
          }
        }
      }
      const stepStartedAt = step.startedAt
      const stepEndedAt = step.endedAt
      const stepStart = validateTimestamp(stepStartedAt, `${path}.startedAt`, errors)
      const stepEnd = validateTimestamp(stepEndedAt, `${path}.endedAt`, errors)
      if (stepStart && stepEnd && Date.parse(stepEndedAt) < Date.parse(stepStartedAt)) {
        errors.push(`${path}.endedAt must not precede startedAt`)
      }
    })
  }

  if (!Array.isArray(value.routeHistory) || value.routeHistory.length > 200) {
    errors.push('$.routeHistory is invalid')
  } else {
    value.routeHistory.forEach((visit, index) => {
      const path = `$.routeHistory[${index}]`
      if (exactKeys(visit, ['capturedAt', 'workflowId', 'route', 'reason'], path, errors)) {
        validateTimestamp(visit.capturedAt, `${path}.capturedAt`, errors)
        if (!isEnum(visit.workflowId, WORKFLOW_IDS)) errors.push(`${path}.workflowId is invalid`)
        if (!isString(visit.route, 1, 200) || !visit.route.startsWith('/')) errors.push(`${path}.route is invalid`)
        if (!isString(visit.reason, 0, 300)) errors.push(`${path}.reason is invalid`)
      }
    })
  }

  if (!Array.isArray(value.findings) || value.findings.length > 200) {
    errors.push('$.findings is invalid')
  } else {
    value.findings.forEach((finding, index) => validateFinding(finding, index, evidenceIds, errors))
  }

  if (
    exactKeys(
      value.privacyCheck,
      ['status', 'checkedChannels', 'canaryCount', 'violations', 'checkedAt'],
      '$.privacyCheck',
      errors,
    )
  ) {
    if (!isEnum(value.privacyCheck.status, ['passed', 'failed'])) errors.push('$.privacyCheck.status is invalid')
    const requiredChannels = ['prompts', 'screenshots', 'report', 'traces', 'logs', 'proposedIssues']
    const checkedChannels = value.privacyCheck.checkedChannels
    if (!isUniqueStrings(checkedChannels) || !requiredChannels.every((channel) => checkedChannels.includes(channel))) {
      errors.push('$.privacyCheck.checkedChannels is incomplete')
    }
    if (!isInteger(value.privacyCheck.canaryCount, 1, Number.MAX_SAFE_INTEGER)) {
      errors.push('$.privacyCheck.canaryCount is invalid')
    }
    if (!Array.isArray(value.privacyCheck.violations)) {
      errors.push('$.privacyCheck.violations is invalid')
    } else {
      value.privacyCheck.violations.forEach((violation, index) => {
        const path = `$.privacyCheck.violations[${index}]`
        if (exactKeys(violation, ['channel', 'path', 'rule'], path, errors)) {
          if (!isString(violation.channel, 0, 40)) errors.push(`${path}.channel is invalid`)
          if (!isString(violation.path, 0, 500)) errors.push(`${path}.path is invalid`)
          if (!isString(violation.rule, 0, 200)) errors.push(`${path}.rule is invalid`)
        }
      })
    }
    validateTimestamp(value.privacyCheck.checkedAt, '$.privacyCheck.checkedAt', errors)
  }

  if (
    exactKeys(
      value.publication,
      ['enabled', 'attempted', 'created', 'duplicates', 'failed'],
      '$.publication',
      errors,
    )
  ) {
    if (typeof value.publication.enabled !== 'boolean') errors.push('$.publication.enabled is invalid')
    for (const key of ['attempted', 'created', 'failed']) {
      if (!isInteger(value.publication[key], 0, 3)) errors.push(`$.publication.${key} is invalid`)
    }
    if (!isInteger(value.publication.duplicates, 0, Number.MAX_SAFE_INTEGER)) {
      errors.push('$.publication.duplicates is invalid')
    }
  }

  if (hasSecret(value)) errors.push('$ must not contain secret-shaped fields or values')
  return { ok: errors.length === 0, errors }
}

export function validateIssuePublicationRequest(value: unknown): ContractValidation {
  const errors: string[] = []
  const keys = ['schemaVersion', 'runId', 'findingId', 'fingerprint', 'repository', 'title', 'body', 'labels']
  if (!exactKeys(value, keys, '$', errors)) return { ok: false, errors }
  if (value.schemaVersion !== ISSUE_SCHEMA_VERSION) errors.push('$.schemaVersion is invalid')
  if (!isString(value.runId) || !runIdPattern.test(value.runId)) errors.push('$.runId is invalid')
  if (!isString(value.findingId) || !findingIdPattern.test(value.findingId)) errors.push('$.findingId is invalid')
  if (!isString(value.fingerprint) || !sha256Pattern.test(value.fingerprint)) {
    errors.push('$.fingerprint is invalid')
  }
  if (
    !isString(value.repository, 1, 200) ||
    !/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(value.repository)
  ) {
    errors.push('$.repository is invalid')
  }
  if (!isString(value.title, 1, 160)) errors.push('$.title is invalid')
  if (!isString(value.body, 1, 12_000)) {
    errors.push('$.body is invalid')
  } else if (
    typeof value.fingerprint === 'string' &&
    !value.body.includes(`<!-- aurearia-ai-finding:v1:${value.fingerprint} -->`)
  ) {
    errors.push('$.body must contain the exact fingerprint marker')
  }
  const allowedLabels = ['ai-browser-finding', 'needs-triage', 'accessibility'] as const
  if (
    !isUniqueStrings(value.labels) ||
    value.labels.length > 3 ||
    !value.labels.every((label) => isEnum(label, allowedLabels))
  ) {
    errors.push('$.labels is invalid')
  }
  if (hasSecret(value)) errors.push('$ must not contain secret-shaped fields or values')
  return { ok: errors.length === 0, errors }
}

export const isRunConfiguration = (value: unknown): value is RunConfiguration =>
  validateRunConfiguration(value).ok
export const isExplorationReport = (value: unknown): value is ExplorationReport =>
  validateExplorationReport(value).ok
export const isIssuePublicationRequest = (value: unknown): value is IssuePublicationRequest =>
  validateIssuePublicationRequest(value).ok
