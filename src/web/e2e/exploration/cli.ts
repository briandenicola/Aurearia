import { randomBytes } from 'node:crypto'
import { mkdir, writeFile } from 'node:fs/promises'
import { spawn } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from '@playwright/test'
import type { Page } from '@playwright/test'
import { BudgetExceededError, BudgetGuard } from './budget'
import {
  executeBrowserAction,
  type BrowserAction,
  type BrowserDriverContext,
  validateBrowserAction,
  validateNavigationDestination,
} from './browser-driver'
import { loadRunConfiguration } from './config'
import { selectExplorationWorkflows, type ExplorationWorkflow } from './workflows'

type ComposeObject = {
  name?: string
  services?: Record<string, {
    build?: unknown
    image?: string
    ports?: Array<string | { host_ip?: string; published?: number | string; target?: number }>
    expose?: Array<string | number>
    volumes?: Array<string | { type?: string; source?: string; target?: string }>
    restart?: string
    env_file?: unknown
    networks?: string[] | Record<string, unknown>
  }>
  volumes?: Record<string, { external?: boolean | { name?: string }; name?: string }>
  networks?: Record<string, { internal?: boolean }>
}
type ComposeServiceNetworks = NonNullable<ComposeObject['services']>[string]['networks']

export interface Lifecycle {
  provision: () => Promise<unknown>
  seed: () => Promise<unknown>
  waitReady: () => Promise<unknown>
  explore: () => Promise<unknown>
  finalize: () => Promise<unknown>
  teardown: () => Promise<unknown>
}

export async function runExplorationLifecycle(lifecycle: Lifecycle): Promise<void> {
  let primaryError: unknown
  try {
    await lifecycle.provision()
    await lifecycle.seed()
    await lifecycle.waitReady()
    await lifecycle.explore()
  } catch (error) {
    primaryError = error
  } finally {
    try {
      await lifecycle.finalize()
    } catch (error) {
      primaryError ??= error
    }
    try {
      await lifecycle.teardown()
    } catch (error) {
      primaryError ??= error
    }
  }
  if (primaryError) throw primaryError
}

export function cleanupCommand(project: string): string[] {
  return [
    'docker', 'compose', '-f', 'docker-compose.exploration.yml', '-p', project,
    'down', '-v', '--remove-orphans',
  ]
}

function isLoopback(origin: string): boolean {
  try {
    const url = new URL(origin)
    return url.protocol === 'http:' && (url.hostname === '127.0.0.1' || url.hostname === 'localhost') &&
      url.port !== '' && url.port !== '8080'
  } catch {
    return false
  }
}

export function assertComposeIsolation(value: unknown, origin: string, project: string): void {
  if (!/^ai-browser-[a-z0-9-]{6,}$/.test(project)) throw new Error('A unique ai-browser Compose project is required')
  if (!isLoopback(origin)) throw new Error('Browser origin must use a random loopback port')
  if (typeof value !== 'object' || value === null) throw new Error('Resolved Compose configuration is invalid')
  const compose = value as ComposeObject
  const services = compose.services ?? {}
  const networkNames = (networks: ComposeServiceNetworks): string[] =>
    Array.isArray(networks) ? networks : Object.keys(networks ?? {})
  if (compose.networks?.internal?.internal !== true) {
    throw new Error('Internal application network must disable external connectivity')
  }
  for (const [name, service] of Object.entries(services)) {
    if (!service.build || service.image) throw new Error(`${name} must build current source without a remote image`)
    if (service.restart && service.restart !== 'no') throw new Error(`${name} must not restart`)
    if (service.env_file) throw new Error(`${name} must not load deployment env files`)
    if (name === 'agent' && service.ports?.length) throw new Error('Agent must not expose a host port')
    const memberNetworks = networkNames(service.networks)
    if (name === 'agent') {
      if (
        memberNetworks.length !== 2 ||
        !memberNetworks.includes('internal') ||
        !memberNetworks.includes('provider-egress')
      ) {
        throw new Error('Agent must use only internal and provider-egress networks')
      }
    }
    if (name === 'app' || name === 'seed') {
      if (memberNetworks.length !== 1 || memberNetworks[0] !== 'internal') {
        throw new Error(`${name} must use only the internal network`)
      }
    }
    for (const volume of service.volumes ?? []) {
      const source = typeof volume === 'string' ? volume.split(':')[0] : (volume.source ?? '')
      const isBind = typeof volume === 'object' && volume.type === 'bind'
      const isProjectVolume = source in (compose.volumes ?? {}) || source.startsWith(`${project}_`)
      if (
        isBind ||
        /^[A-Za-z]:[\\/]/.test(source) ||
        source.startsWith('/') ||
        source.includes('..') ||
        !isProjectVolume
      ) {
        throw new Error(`${name} uses an external filesystem path`)
      }
    }
  }
  const appPort = services.app?.ports?.[0]
  if (!appPort || typeof appPort === 'string') throw new Error('App must declare structured loopback random port')
  if (appPort.host_ip !== '127.0.0.1' || ![0, '0', undefined].includes(appPort.published)) {
    throw new Error('App port must be loopback-only and OS-assigned')
  }
  for (const [name, volume] of Object.entries(compose.volumes ?? {})) {
    if (volume.external || (volume.name && volume.name !== `${project}_${name}`)) {
      throw new Error(`Volume ${name} must be project scoped`)
    }
  }
}

async function run(
  command: string,
  args: string[],
  options: { cwd: string; env?: NodeJS.ProcessEnv; signal?: AbortSignal },
):
Promise<{ stdout: string; stderr: string }> {
  return await new Promise((resolvePromise, reject) => {
    const child = spawn(command, args, {
      cwd: options.cwd,
      env: options.env,
      signal: options.signal,
      windowsHide: true,
    })
    let stdout = ''
    let stderr = ''
    child.stdout.on('data', (chunk) => { stdout += String(chunk) })
    child.stderr.on('data', (chunk) => { stderr += String(chunk) })
    child.once('error', reject)
    child.once('exit', (code) => {
      if (code === 0) resolvePromise({ stdout, stderr })
      else reject(new Error(`${command} ${args.join(' ')} failed (${code}): ${stderr || stdout}`))
    })
  })
}

const ALLOWED_ACTIONS = [
  'navigate', 'click', 'fill', 'select', 'upload_fixture',
  'set_viewport', 'back', 'wait_for_ui', 'checkpoint', 'finish',
] as const

type DecisionResponse = BrowserAction & {
  schemaVersion: 'aurearia.browser-exploration-decision/v1'
  rationale: string
  suspectedFindings: unknown[]
  usage: { inputTokens: number; outputTokens: number }
}

type DecisionRequester = (body: Record<string, unknown>, signal: AbortSignal) => Promise<unknown>

function parseDecision(value: unknown, context: BrowserDriverContext): DecisionResponse {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error('Decision service returned an invalid object')
  }
  const candidate = value as Partial<DecisionResponse>
  if (
    candidate.schemaVersion !== 'aurearia.browser-exploration-decision/v1' ||
    typeof candidate.rationale !== 'string' ||
    candidate.rationale.length > 1000 ||
    !Array.isArray(candidate.suspectedFindings) ||
    typeof candidate.usage !== 'object' ||
    candidate.usage === null ||
    !Number.isInteger(candidate.usage.inputTokens) ||
    candidate.usage.inputTokens < 0 ||
    !Number.isInteger(candidate.usage.outputTokens) ||
    candidate.usage.outputTokens < 0
  ) {
    throw new Error('Decision service returned an invalid contract')
  }
  validateBrowserAction(candidate, context)
  return candidate as DecisionResponse
}

async function observePage(page: Page, budget: BudgetGuard, workflow: ExplorationWorkflow): Promise<string> {
  const snapshot = await budget.perform('browserActions', async () =>
    page.locator('body').ariaSnapshot({ timeout: 3000 }))
  const route = new URL(page.url()).pathname
  if (!workflow.routes.includes(route)) throw new Error('Observation route escaped the selected workflow')
  return `Route: ${route}\n${snapshot.slice(0, 1800)}`
}

export async function executeDecisionLoop(options: {
  page: Page
  budget: BudgetGuard
  workflow: ExplorationWorkflow
  context: BrowserDriverContext
  runId: string
  provider: string
  model: string
  nextStepIndex: () => number
  requestDecision: DecisionRequester
}): Promise<'checkpoint' | 'finished'> {
  let unchangedDecisions = 0
  let previousAction = ''
  let currentObservation: string | undefined
  for (;;) {
    options.budget.reserve('steps')
    const stepIndex = options.nextStepIndex()
    const observation = currentObservation ??
      await observePage(options.page, options.budget, options.workflow)
    const stateBeforeAction = `${options.page.url()}\n${observation}`
    const decisionValue = await options.budget.perform('modelCalls', async (signal) =>
      options.requestDecision({
        schemaVersion: 'aurearia.browser-exploration-decision/v1',
        runId: options.runId,
        stepId: `step-${String(stepIndex).padStart(3, '0')}`,
        provider: options.provider,
        model: options.model,
        workflow: options.workflow.id,
        goal: options.workflow.goal,
        allowedActions: [...ALLOWED_ACTIONS],
        allowedRoutes: [...options.workflow.routes],
        observations: [{
          evidenceId: `ev_ui_${String(stepIndex).padStart(4, '0')}`,
          kind: 'ui',
          route: new URL(options.page.url()).pathname,
          summary: observation,
        }],
      }, signal))
    const decision = parseDecision(decisionValue, options.context)
    options.budget.reconcileModelUsage(decision.usage)

    const actionSignature = JSON.stringify({ action: decision.action, target: decision.target })
    if (decision.action === 'finish' || decision.action === 'checkpoint') {
      const result = await executeBrowserAction(options.page, decision, options.context)
      if (result === 'checkpoint' || result === 'finished') return result
      throw new Error('Terminal decision did not terminate the workflow')
    }
    await options.budget.perform('browserActions', async () =>
      executeBrowserAction(options.page, decision, options.context))
    currentObservation = await observePage(options.page, options.budget, options.workflow)
    const nextState = `${options.page.url()}\n${currentObservation}`
    unchangedDecisions = nextState === stateBeforeAction
      ? (actionSignature === previousAction ? unchangedDecisions + 1 : 1)
      : 0
    if (unchangedDecisions >= 3) throw new Error('Exploration stopped after repeated actions made no progress')
    previousAction = actionSignature
  }
}

async function waitFor(url: string, signal: AbortSignal): Promise<void> {
  for (let attempt = 0; attempt < 60; attempt += 1) {
    if (signal.aborted) throw signal.reason
    try {
      const response = await fetch(url, { signal })
      if (response.ok) return
    } catch {
      // Keep retrying only this loopback endpoint until the bounded deadline.
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 1000))
  }
  throw new Error(`Readiness failed: ${url}`)
}

function generatedEnvironment(project: string): NodeJS.ProcessEnv {
  const secret = () => randomBytes(32).toString('base64url')
  return {
    ...process.env,
    COMPOSE_PROJECT_NAME: project,
    JWT_SECRET: secret(),
    AGENT_INTERNAL_SERVICE_TOKEN: secret(),
    AI_BROWSER_INTERNAL_BEARER_TOKEN: secret(),
    AI_BROWSER_TEST_USERNAME: `explorer-${randomBytes(6).toString('hex')}`,
    AI_BROWSER_TEST_PASSWORD: secret(),
    AI_BROWSER_TEST_EMAIL: `explorer-${randomBytes(6).toString('hex')}@example.test`,
    AI_BROWSER_FAKE_MODEL: process.env.AI_BROWSER_FAKE_MODEL ?? 'true',
  }
}

export async function main(argv = process.argv.slice(2)): Promise<void> {
  const configIndex = argv.indexOf('--config')
  if (configIndex < 0 || !argv[configIndex + 1]) throw new Error('Usage: cli.ts --config <ignored run config>')
  const config = await loadRunConfiguration(resolve(argv[configIndex + 1]))
  const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../../..')
  const composeFile = 'docker-compose.exploration.yml'
  const compactTime = new Date().toISOString().replace(/[-:]/g, '').replace(/\.\d{3}Z$/, 'Z')
  const runId = `aibr_${compactTime}_${randomBytes(6).toString('hex')}`
  const project = `ai-browser-${Date.now().toString(36)}-${randomBytes(4).toString('hex')}`
  const env = generatedEnvironment(project)
  const budget = new BudgetGuard(config.limits)
  const artifactDir = resolve(repoRoot, '.artifacts', 'ai-browser', project)
  let appOrigin = 'http://127.0.0.1:0'
  let outcome: Record<string, unknown> = { runId, project, status: 'infrastructure_failed' }

  const compose = async (signal: AbortSignal | undefined, ...args: string[]) => run(
    'docker',
    ['compose', '-f', composeFile, '-p', project, ...args],
    { cwd: repoRoot, env, signal },
  )
  try {
    await runExplorationLifecycle({
      provision: async () => budget.withinDeadline(async (signal) => {
        const resolved = JSON.parse((await compose(signal, 'config', '--format', 'json')).stdout) as unknown
        assertComposeIsolation(resolved, 'http://127.0.0.1:49152', project)
        await compose(signal, 'up', '--build', '-d', '--wait', 'agent')
      }),
      seed: async () => budget.withinDeadline(async (signal) => {
        await compose(signal, 'run', '--rm', 'seed')
      }),
      waitReady: async () => budget.withinDeadline(async (signal) => {
        await compose(signal, 'up', '--build', '-d', 'app')
        const port = (await compose(signal, 'port', 'app', '8080')).stdout.trim().split(':').pop()
        if (!port || port === '8080') throw new Error('Compose did not assign a random app port')
        appOrigin = `http://127.0.0.1:${port}`
        if (!isLoopback(appOrigin)) throw new Error('Resolved app target is not isolated loopback')
        await waitFor(`${appOrigin}/healthz`, signal)
      }),
      explore: async () => {
        const browser = await budget.withinDeadline(async () => chromium.launch())
        try {
          const page = await budget.withinDeadline(async () => browser.newPage())
          await budget.perform('browserActions', async () => page.goto(`${appOrigin}/login`))
          await budget.perform('browserActions', async () =>
            page.getByRole('textbox').first().fill(env.AI_BROWSER_TEST_USERNAME!))
          await budget.perform('browserActions', async () =>
            page.locator('input[type="password"]').fill(env.AI_BROWSER_TEST_PASSWORD!))
          await budget.perform('browserActions', async () =>
            page.getByRole('button', { name: 'Sign In' }).click())
          validateNavigationDestination(page.url(), {
            origin: appOrigin,
            allowedRoutes: ['/'],
            uploadFixtures: {},
          })

          let stepIndex = 0
          const requestDecision: DecisionRequester = async (body, signal) => {
            const response = await fetch(`${appOrigin}/api/internal/browser-exploration/decide`, {
              method: 'POST',
              signal,
              headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${env.AI_BROWSER_INTERNAL_BEARER_TOKEN}`,
              },
              body: JSON.stringify(body),
            })
            if (!response.ok) throw new Error(`Decision service returned HTTP ${response.status}`)
            return await response.json() as unknown
          }
          for (const workflow of selectExplorationWorkflows(config.workflowScope)) {
            const context: BrowserDriverContext = {
              origin: appOrigin,
              allowedRoutes: workflow.routes,
              uploadFixtures: {},
            }
            await budget.perform('browserActions', async () => page.setViewportSize(workflow.viewport))
            await budget.perform('browserActions', async () =>
              executeBrowserAction(
                page,
                { action: 'navigate', target: { route: workflow.routes[0] } },
                context,
              ))
            await executeDecisionLoop({
              page,
              budget,
              workflow,
              context,
              runId,
              provider: config.provider,
              model: config.model,
              nextStepIndex: () => {
                stepIndex += 1
                return stepIndex
              },
              requestDecision,
            })
          }
          const termination = budget.termination()
          outcome = termination.reason === 'completed'
            ? { runId, project, appOrigin, status: 'completed', termination, ...budget.snapshot() }
            : { runId, project, appOrigin, status: 'bounded', termination, ...budget.snapshot() }
        } catch (error) {
          if (error instanceof BudgetExceededError) {
            outcome = {
              runId,
              project,
              appOrigin,
              status: 'bounded',
              termination: budget.termination(),
              ...budget.snapshot(),
            }
            return
          }
          throw error
        } finally {
          await browser.close()
        }
      },
    finalize: async () => {
      await mkdir(artifactDir, { recursive: true, mode: 0o700 })
      await writeFile(resolve(artifactDir, 'phase3-run.json'), JSON.stringify(outcome, null, 2), { mode: 0o600 })
    },
    teardown: async () => {
      let inspectionError: unknown
      const cleanup = cleanupCommand(project)
      await writeFile(
        resolve(artifactDir, 'cleanup-attempt.json'),
        JSON.stringify({ attemptedAt: new Date().toISOString(), command: cleanup }, null, 2),
        { mode: 0o600 },
      )
      try {
        const inspection = await compose(undefined, 'ps', '--format', 'json')
        await writeFile(resolve(artifactDir, 'compose-targets.json'), inspection.stdout, { mode: 0o600 })
      } catch (error) {
        inspectionError = error
      } finally {
        const [, ...args] = cleanup
        await run('docker', args, { cwd: repoRoot, env })
      }
      if (inspectionError) throw inspectionError
    },
    })
  } finally {
    budget.dispose()
  }
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : ''
if (invokedPath === fileURLToPath(import.meta.url)) {
  main().catch((error) => {
    console.error(error instanceof Error ? error.message : error)
    process.exitCode = 1
  })
}
