import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const repositoryRoot = resolve(process.cwd(), '../..')
const workflowPath = resolve(repositoryRoot, '.github', 'workflows', 'ai-browser-exploration.yml')
const workflow = readFileSync(workflowPath, 'utf8').replaceAll('\r', '')

describe('AI browser advisory workflow', () => {
  it('runs manually and nightly without changing the Quality Gate', () => {
    expect(workflow).toMatch(/\n {2}workflow_dispatch:\s*\n/)
    expect(workflow).toMatch(/\n {2}schedule:\s*\n/)
    expect(workflow).toContain("AI_BROWSER_FAKE_MODEL: 'true'")
    expect(workflow).toContain("github.event_name == 'schedule' && 'nightly' || 'manual'")
    expect(workflow).not.toContain('pull_request:')
    expect(workflow).not.toContain('push:')
  })

  it('uses least privilege and immutable action pins', () => {
    expect(workflow).toMatch(/permissions:\n {2}contents: read/)
    expect(workflow).not.toContain('issues: write')
    const actionUses = [...workflow.matchAll(/uses:\s+[^@\s]+@([^\s]+)/g)]
    expect(actionUses.length).toBeGreaterThan(0)
    for (const match of actionUses) expect(match[1]).toMatch(/^[a-f0-9]{40}$/)
  })

  it('always tears down the exact project and checks for leaked resources', () => {
    expect(workflow).toMatch(/name: Tear down exact Compose project\n {8}if: always\(\)/)
    expect(workflow).toContain('-p "$AI_BROWSER_COMPOSE_PROJECT"')
    expect(workflow).toContain('down -v --remove-orphans')
    expect(workflow).toMatch(/name: Assert no project resources remain\n {8}if: always\(\)/)
    expect(workflow).toContain('label=com.docker.compose.project=$AI_BROWSER_COMPOSE_PROJECT')
  })

  it('uploads finite-retention evidence even after failure', () => {
    expect(workflow).toMatch(/name: Upload sanitized exploration evidence\n {8}if: always\(\)/)
    expect(workflow).toContain('retention-days: 14')
    expect(workflow).toContain('if-no-files-found: warn')
  })
})
