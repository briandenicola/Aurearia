import { describe, expect, it } from 'vitest'
import { assertComposeIsolation, cleanupCommand } from '../cli'

const safe = {
  name: 'ai-browser-test-abc123',
  services: {
    app: {
      build: '.',
      ports: ['127.0.0.1:49152:8080'],
      volumes: ['db:/app/data', 'uploads:/app/uploads'],
      networks: ['internal'],
    },
    agent: {
      build: { context: 'src/agent' },
      expose: [8081],
      networks: ['internal', 'provider-egress'],
    },
    seed: { build: '.', volumes: ['db:/app/data'], restart: 'no', networks: ['internal'] },
  },
  volumes: { db: {}, uploads: {} },
  networks: { internal: { internal: true }, 'provider-egress': {} },
}

describe('Compose isolation guards', () => {
  it('accepts current-source, allocated-port, project-scoped isolation', () => {
    expect(() => assertComposeIsolation(safe, 'http://127.0.0.1:49152', 'ai-browser-test-abc123')).not.toThrow()
  })

  type UnsafeChange = {
    origin?: string
    project?: string
    mutate?: (value: typeof safe) => void
  }
  it.each([
    ['non-loopback origin', { origin: 'https://beta.example.com' }],
    ['fixed public port', { mutate: (v) => { v.services.app.ports[0] = '0.0.0.0:8080:8080' } }],
    ['remote image', { mutate: (v) => { Object.assign(v.services.app, { image: 'registry/app:latest', build: undefined }) } }],
    ['agent host exposure', { mutate: (v) => { Object.assign(v.services.agent, { ports: ['8081:8081'] }) } }],
    ['external database path', { mutate: (v) => { v.services.app.volumes = ['C:/prod/data:/app/data'] } }],
    ['production volume', { mutate: (v) => { Object.assign(v.volumes.db, { external: true }) } }],
    ['named production volume', { mutate: (v) => { Object.assign(v.volumes.db, { name: 'production-db' }) } }],
    ['restart policy', { mutate: (v) => { Object.assign(v.services.app, { restart: 'unless-stopped' }) } }],
    ['deployment env file', { mutate: (v) => { Object.assign(v.services.app, { env_file: ['.env'] }) } }],
    ['external-capable internal network', { mutate: (v) => { v.networks.internal.internal = false } }],
    ['app provider egress', { mutate: (v) => { v.services.app.networks.push('provider-egress') } }],
    ['seed provider egress', { mutate: (v) => { v.services.seed.networks.push('provider-egress') } }],
    ['agent missing internal network', { mutate: (v) => { v.services.agent.networks = ['provider-egress'] } }],
    ['agent extra network', { mutate: (v) => { v.services.agent.networks.push('unexpected') } }],
    ['missing unique project', { project: 'ancientcoins' }],
  ] satisfies Array<[string, UnsafeChange]>)('rejects %s', (_name, change) => {
    const value = structuredClone(safe)
    change.mutate?.(value)
    expect(() => assertComposeIsolation(value, change.origin ?? 'http://localhost:49152', change.project ?? safe.name)).toThrow()
  })

  it('always uses volume/orphan teardown', () => {
    expect(cleanupCommand('ai-browser-test-abc123')).toEqual([
      'docker', 'compose', '-f', 'docker-compose.exploration.yml', '-p', 'ai-browser-test-abc123',
      'down', '-v', '--remove-orphans',
    ])
  })
})
