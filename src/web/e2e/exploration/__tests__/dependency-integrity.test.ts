import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const webRoot = process.cwd()
const repositoryRoot = resolve(webRoot, '..', '..')

const readJson = (path: string): Record<string, unknown> =>
  JSON.parse(readFileSync(path, 'utf8')) as Record<string, unknown>

describe('Feature 360 dependency integrity', () => {
  it('pins the direct Playwright dependency and its complete lock family', () => {
    const packageJson = readJson(resolve(webRoot, 'package.json'))
    const packageLock = readJson(resolve(webRoot, 'package-lock.json'))
    const devDependencies = packageJson.devDependencies as Record<string, string>
    const packages = packageLock.packages as Record<string, Record<string, unknown>>
    const lockedRoot = packages[''].devDependencies as Record<string, string>

    expect(devDependencies['@playwright/test']).toBe('1.63.0')
    expect((packageJson.engines as Record<string, string>).node).toBe('^22.22.2 || >=24.15.0')
    expect(lockedRoot['@playwright/test']).toBe('1.63.0')
    expect(packages['node_modules/@playwright/test'].version).toBe('1.63.0')
    expect(packages['node_modules/@playwright/test'].resolved).toBe(
      'https://registry.npmjs.org/@playwright/test/-/test-1.63.0.tgz',
    )
    expect(packages['node_modules/@playwright/test'].integrity).toBe(
      'sha512-oxMK4vllB9RK5NQ2l1pq1IfOf2AvnEuj/vYGDj0H2nMtmtZpKtCwt/l00GEO6xjGfpBNAvjovvYdCm50dRQkpQ==',
    )
    expect(packages['node_modules/playwright'].version).toBe('1.63.0')
    expect(packages['node_modules/playwright'].resolved).toBe(
      'https://registry.npmjs.org/playwright/-/playwright-1.63.0.tgz',
    )
    expect(packages['node_modules/playwright'].integrity).toBe(
      'sha512-+7ziBLidS4NaNCdt57SUDT+wYmmd5fmiQejUic/kb+YsYSCPyOOE9sebzMjNmQrsnNpDJqd4WHvV/8lfKfUDUg==',
    )
    expect(packages['node_modules/playwright-core'].version).toBe('1.63.0')
    expect(packages['node_modules/playwright-core'].resolved).toBe(
      'https://registry.npmjs.org/playwright-core/-/playwright-core-1.63.0.tgz',
    )
    expect(packages['node_modules/playwright-core'].integrity).toBe(
      'sha512-rYCsBF/M5HjUch52bbtVONEFjv6Xu8sm8h72dNlR5bzIE1fvC/bxgspzkjSfU+MweEMmPM8KJebG6nnyxo5mCg==',
    )
  })

  it('does not add a browser-agent or model SDK to npm dependencies', () => {
    const packageJson = readJson(resolve(webRoot, 'package.json'))
    const dependencies = {
      ...(packageJson.dependencies as Record<string, string>),
      ...(packageJson.devDependencies as Record<string, string>),
    }
    const forbidden = [
      '@anthropic-ai/sdk',
      '@browserbasehq/stagehand',
      '@langchain/core',
      'browser-use',
      'langchain',
      'openai',
      'playwright-ai',
    ]

    const lockedPackages = Object.keys(
      (readJson(resolve(webRoot, 'package-lock.json')).packages ?? {}) as Record<string, unknown>,
    ).map((name) => name.replace(/^node_modules\//, ''))
    expect(Object.keys(dependencies).filter((name) => forbidden.includes(name))).toEqual([])
    expect(lockedPackages.filter((name) => forbidden.includes(name))).toEqual([])
  })

  it('preserves the approved Python model-stack lock resolutions', () => {
    const uvLock = readFileSync(resolve(repositoryRoot, 'src', 'agent', 'uv.lock'), 'utf8').replaceAll('\r', '')
    const lockedVersion = (name: string): string | undefined => {
      const block = uvLock
        .split('[[package]]')
        .find((candidate) => candidate.includes(`\nname = "${name}"\n`))
      return block?.match(/\nversion = "([^"]+)"/)?.[1]
    }

    expect(lockedVersion('langchain')).toBe('1.4.0')
    expect(lockedVersion('langchain-anthropic')).toBe('1.7.2')
    expect(lockedVersion('langchain-ollama')).toBe('1.1.0')
    expect(lockedVersion('langgraph')).toBe('1.2.11')
  })
})
