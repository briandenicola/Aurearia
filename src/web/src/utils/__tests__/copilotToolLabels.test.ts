import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'
import { COPILOT_TOOL_LABELS, copilotToolLabel } from '../copilotToolLabels'

function agentToolLabels(): Record<string, string> {
  const source = readFileSync(resolve(__dirname, '../../../../agent/app/teams/coin_copilot.py'), 'utf-8')
  const block = source.match(/_TOOL_LABELS = \{([\s\S]*?)\n\}/)
  if (!block) throw new Error('_TOOL_LABELS not found in coin_copilot.py')
  return Object.fromEntries(
    [...block[1].matchAll(/"([a-z_]+)":\s*"([^"]+)"/g)].map(([, key, label]) => [key, label]),
  )
}

function goAllowedTools(): string[] {
  const source = readFileSync(resolve(__dirname, '../../../../api/services/coin_copilot_contract.go'), 'utf-8')
  const block = source.match(/CoinCopilotAllowedTools = \[\]string\{([\s\S]*?)\}/)
  if (!block) throw new Error('CoinCopilotAllowedTools not found')
  return [...block[1].matchAll(/"([a-z_]+)"/g)].map(match => match[1])
}

describe('Copilot tool labels', () => {
  it('match the agent checklist labels exactly', () => {
    expect(COPILOT_TOOL_LABELS).toEqual(agentToolLabels())
  })

  it('cover every tool the API allows', () => {
    for (const tool of goAllowedTools()) {
      expect(COPILOT_TOOL_LABELS, tool).toHaveProperty(tool)
    }
  })

  it('never falls back to an internal tool name', () => {
    expect(copilotToolLabel('market_search')).toBe('Search dealer listings')
    expect(copilotToolLabel('future_tool')).toBe('Copilot helper')
  })
})
