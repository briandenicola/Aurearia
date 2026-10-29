import { spawnSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path, { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, describe, expect, it } from 'vitest'

// The Go API embeds a generated copy of this viewer. Rebuild it into a temp
// directory and require the committed file to match, so a source change
// without `npm run build:mcp-app` fails the web gate instead of shipping a
// stale view.
const testDir = path.dirname(fileURLToPath(import.meta.url))
const webRoot = path.resolve(testDir, '../../../..')
const committed = path.resolve(webRoot, '../api/handlers/mcpapps/coin-viewer.html')
const outDir = mkdtempSync(join(tmpdir(), 'coin-viewer-'))

afterAll(() => rmSync(outDir, { recursive: true, force: true }))

describe('embedded coin viewer', () => {
  it('matches a fresh build of the Vue source', () => {
    const result = spawnSync(
      process.execPath,
      [join(webRoot, 'node_modules/vite/bin/vite.js'), 'build', '--config', 'vite.config.mcp-app.ts', '--outDir', outDir, '--logLevel', 'error'],
      // Vitest exports NODE_ENV=test, which would produce a development build.
      { cwd: webRoot, encoding: 'utf8', timeout: 60_000, env: { ...process.env, NODE_ENV: 'production' } },
    )
    expect(result.status, result.stderr || result.stdout).toBe(0)

    const normalize = (text: string) => text.replace(/\r\n/g, '\n')
    const fresh = normalize(readFileSync(join(outDir, 'coin-viewer.html'), 'utf8'))
    const embedded = normalize(readFileSync(committed, 'utf8'))
    expect(embedded === fresh, 'src/api/handlers/mcpapps/coin-viewer.html is stale: run `npm run build:mcp-app` in src/web').toBe(true)
  }, 60_000)
})
