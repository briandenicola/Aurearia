import { execFile } from 'node:child_process'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { promisify } from 'node:util'
import { describe, expect, it } from 'vitest'
import { main } from '../cli'

const execFileAsync = promisify(execFile)
const enabled = process.env.AI_BROWSER_CI_ACCEPTANCE === 'true'

async function dockerLines(args: string[]): Promise<string[]> {
  const { stdout } = await execFileAsync('docker', args, { windowsHide: true })
  return stdout.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
}

describe.skipIf(!enabled)('Docker exploration acceptance', () => {
  it('runs the isolated fake-model stack and leaves no Compose resources', async () => {
    const project = process.env.AI_BROWSER_COMPOSE_PROJECT
    const configPath = process.env.AI_BROWSER_RUN_CONFIG
    if (!project || !configPath) throw new Error('CI acceptance project and configuration are required')

    await main(['--config', configPath])

    const repoRoot = resolve(process.cwd(), '../..')
    const outcome = JSON.parse(await readFile(
      resolve(repoRoot, '.artifacts', 'ai-browser', project, 'phase3-run.json'),
      'utf8',
    )) as { project?: string; status?: string }
    expect(outcome).toMatchObject({ project, status: 'completed' })

    expect(await dockerLines([
      'ps', '-aq', '--filter', `label=com.docker.compose.project=${project}`,
    ])).toEqual([])
    expect(await dockerLines([
      'volume', 'ls', '-q', '--filter', `label=com.docker.compose.project=${project}`,
    ])).toEqual([])
    expect(await dockerLines([
      'network', 'ls', '-q', '--filter', `label=com.docker.compose.project=${project}`,
    ])).toEqual([])
  }, 20 * 60 * 1000)
})
