import { readFile } from 'node:fs/promises'
import {
  type ExplorationLimits,
  type RunConfiguration,
  validateRunConfiguration,
} from './contracts'

export const HARD_MAXIMUMS: Readonly<ExplorationLimits> = Object.freeze({
  steps: 25,
  wallTimeSeconds: 900,
  modelCalls: 20,
  modelTokens: 60_000,
  browserActions: 100,
  issueAttempts: 3,
})

function immutableConfiguration(value: RunConfiguration): Readonly<RunConfiguration> {
  const limits = Object.freeze({ ...value.limits })
  const workflowScope = Object.freeze([...value.workflowScope])
  return Object.freeze({ ...value, limits, workflowScope }) as Readonly<RunConfiguration>
}

export function parseRunConfiguration(value: unknown): Readonly<RunConfiguration> {
  const candidate =
    typeof value === 'object' && value !== null && !Array.isArray(value) && !Object.hasOwn(value, 'createIssues')
      ? { ...value, createIssues: false }
      : value
  const result = validateRunConfiguration(candidate)
  if (!result.ok) {
    throw new Error(`Invalid exploration configuration: ${result.errors.join('; ')}`)
  }
  return immutableConfiguration(candidate as RunConfiguration)
}

export async function loadRunConfiguration(path: string): Promise<Readonly<RunConfiguration>> {
  const raw = await readFile(path, 'utf8')
  return parseRunConfiguration(JSON.parse(raw) as unknown)
}
