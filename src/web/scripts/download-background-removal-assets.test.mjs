import assert from 'node:assert/strict'
import { Buffer } from 'node:buffer'
import { createHash } from 'node:crypto'
import { mkdtemp, readFile, rm, stat, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, sep } from 'node:path'
import test from 'node:test'
import { pathToFileURL } from 'node:url'

import {
  EXPECTED_BASE_URL,
  PACKAGE_NAME,
  REQUIRED_RESOURCES,
  downloadVerifiedAsset,
  prepareBackgroundRemovalAssets,
  resolveAssetLocation,
  validateLockfile,
} from './download-background-removal-assets.mjs'

const lockfile = JSON.parse(
  await readFile(new URL('./background-removal-assets.lock.json', import.meta.url), 'utf8'),
)

function cloneLockfile() {
  return JSON.parse(JSON.stringify(lockfile))
}

test('accepts the committed asset lockfile', () => {
  assert.equal(validateLockfile(lockfile, '1.7.0'), lockfile)
})

test('rejects missing, unexpected, and duplicate locked assets', () => {
  const missing = cloneLockfile()
  delete missing.resources[REQUIRED_RESOURCES[0]]
  assert.throws(() => validateLockfile(missing, '1.7.0'), /exactly these resources/)

  const unexpected = cloneLockfile()
  unexpected.resources['/models/unreviewed'] = unexpected.resources[REQUIRED_RESOURCES[0]]
  assert.throws(() => validateLockfile(unexpected, '1.7.0'), /exactly these resources/)

  const duplicate = cloneLockfile()
  duplicate.resources[REQUIRED_RESOURCES[1]].chunks[0] =
    duplicate.resources[REQUIRED_RESOURCES[0]].chunks[0]
  duplicate.resources[REQUIRED_RESOURCES[1]].size =
    duplicate.resources[REQUIRED_RESOURCES[1]].chunks[0].offsets[1]
  assert.throws(() => validateLockfile(duplicate, '1.7.0'), /Duplicate/)
})

test('rejects version and trusted-origin drift', () => {
  assert.throws(() => validateLockfile(lockfile, '1.7.1'), /does not match/)

  const offOrigin = cloneLockfile()
  offOrigin.baseUrl = 'https://example.com/assets/'
  assert.throws(() => validateLockfile(offOrigin, '1.7.0'), /base URL/)
})

test('rejects absolute URLs, traversal, encoded traversal, and nested paths', () => {
  const unsafeNames = [
    'https://example.com/payload',
    '../payload',
    '%2e%2e%2fpayload',
    '/payload',
    'nested/payload',
    String.raw`..\payload`,
  ]

  for (const name of unsafeNames) {
    assert.throws(() => resolveAssetLocation(tmpdir(), name), /Invalid/)
  }
})

test('downloads a locked asset only when size and SHA-256 match', async (context) => {
  const outPath = await mkdtemp(join(tmpdir(), 'aurearia-assets-'))
  context.after(() => rm(outPath, { recursive: true, force: true }))

  const bytes = Buffer.from('reviewed asset')
  const hash = createHash('sha256').update(bytes).digest('hex')
  const chunk = { name: hash, hash, offsets: [0, bytes.byteLength] }
  let observedUrl
  let observedOptions
  const fetchImpl = async (url, options) => {
    observedUrl = url
    observedOptions = options
    return new Response(bytes, {
      headers: { 'content-length': String(bytes.byteLength) },
    })
  }

  await downloadVerifiedAsset({ fetchImpl, outPath, chunk })

  assert.equal(observedUrl.href, `${EXPECTED_BASE_URL}${hash}`)
  assert.deepEqual(observedOptions, { redirect: 'error' })
  assert.deepEqual(await readFile(join(outPath, hash)), bytes)
})

test('does not publish tampered or incorrectly sized assets', async (context) => {
  const outPath = await mkdtemp(join(tmpdir(), 'aurearia-assets-'))
  context.after(() => rm(outPath, { recursive: true, force: true }))

  const expectedBytes = Buffer.from('reviewed asset')
  const tamperedBytes = Buffer.from('tampered asset')
  const hash = createHash('sha256').update(expectedBytes).digest('hex')
  const chunk = { name: hash, hash, offsets: [0, expectedBytes.byteLength] }

  await assert.rejects(
    downloadVerifiedAsset({
      fetchImpl: async () =>
        new Response(tamperedBytes, {
          headers: { 'content-length': String(expectedBytes.byteLength) },
        }),
      outPath,
      chunk,
    }),
    /failed SHA-256 verification/,
  )
  await assert.rejects(stat(join(outPath, hash)), { code: 'ENOENT' })

  await assert.rejects(
    downloadVerifiedAsset({
      fetchImpl: async () =>
        new Response(expectedBytes, {
          headers: { 'content-length': String(expectedBytes.byteLength + 1) },
        }),
      outPath,
      chunk,
    }),
    /declared .* expected/,
  )
  await assert.rejects(stat(join(outPath, hash)), { code: 'ENOENT' })

  await assert.rejects(
    downloadVerifiedAsset({
      fetchImpl: async () => new Response(Buffer.concat([expectedBytes, Buffer.from('extra')])),
      outPath,
      chunk,
    }),
    /exceeded its locked size/,
  )
  await assert.rejects(stat(join(outPath, hash)), { code: 'ENOENT' })
})

test('rejects redirected and failed responses', async (context) => {
  const outPath = await mkdtemp(join(tmpdir(), 'aurearia-assets-'))
  context.after(() => rm(outPath, { recursive: true, force: true }))

  const bytes = Buffer.from('reviewed asset')
  const hash = createHash('sha256').update(bytes).digest('hex')
  const chunk = { name: hash, hash, offsets: [0, bytes.byteLength] }

  await assert.rejects(
    downloadVerifiedAsset({
      fetchImpl: async () => new Response(null, { status: 302 }),
      outPath,
      chunk,
    }),
    /HTTP 302/,
  )
})

test('publishes a complete verified asset directory from a trailing-slash URL', async (context) => {
  const root = await mkdtemp(join(tmpdir(), 'aurearia-prepare-assets-'))
  context.after(() => rm(root, { recursive: true, force: true }))

  const payloads = REQUIRED_RESOURCES.map((_, index) => Buffer.from(`asset-${index}`))
  const resources = Object.fromEntries(
    REQUIRED_RESOURCES.map((resourceName, index) => {
      const bytes = payloads[index]
      const hash = createHash('sha256').update(bytes).digest('hex')
      return [
        resourceName,
        {
          chunks: [{ hash, name: hash, offsets: [0, bytes.byteLength] }],
          size: bytes.byteLength,
          mime: 'application/octet-stream',
        },
      ]
    }),
  )
  const packageJsonPath = join(root, 'package.json')
  const lockfilePath = join(root, 'assets.lock.json')
  const outPath = join(root, 'public', 'assets')
  await writeFile(
    packageJsonPath,
    JSON.stringify({ dependencies: { [PACKAGE_NAME]: '1.7.0' } }),
  )
  await writeFile(
    lockfilePath,
    JSON.stringify({
      package: PACKAGE_NAME,
      version: '1.7.0',
      baseUrl: EXPECTED_BASE_URL,
      resources,
    }),
  )

  await prepareBackgroundRemovalAssets({
    packageJsonUrl: pathToFileURL(packageJsonPath),
    lockfileUrl: pathToFileURL(lockfilePath),
    outDir: pathToFileURL(`${outPath}${sep}`),
    fetchImpl: async (url) => {
      const hash = url.pathname.split('/').at(-1)
      const resourceIndex = Object.values(resources).findIndex(
        (resource) => resource.chunks[0].hash === hash,
      )
      return new Response(payloads[resourceIndex], {
        headers: { 'content-length': String(payloads[resourceIndex].byteLength) },
      })
    },
  })

  assert.deepEqual(JSON.parse(await readFile(join(outPath, 'resources.json'), 'utf8')), resources)
  for (const [index, resource] of Object.values(resources).entries()) {
    assert.deepEqual(await readFile(join(outPath, resource.chunks[0].name)), payloads[index])
  }
})
