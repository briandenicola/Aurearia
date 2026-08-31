import { Buffer } from 'node:buffer'
import { createHash, randomUUID } from 'node:crypto'
import { readFile, mkdir, rename, rm, writeFile } from 'node:fs/promises'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

export const PACKAGE_NAME = '@imgly/background-removal'
export const EXPECTED_BASE_URL =
  'https://staticimgly.com/@imgly/background-removal-data/1.7.0/dist/'
export const REQUIRED_RESOURCES = [
  '/onnxruntime-web/ort-wasm-simd-threaded.wasm',
  '/onnxruntime-web/ort-wasm-simd-threaded.mjs',
  '/models/isnet_quint8',
]

const SHA256_PATTERN = /^[a-f0-9]{64}$/

function assertPlainObject(value, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} must be an object`)
  }
}

function assertChunkName(name) {
  if (typeof name !== 'string' || !SHA256_PATTERN.test(name)) {
    throw new Error(`Invalid background-removal chunk name: ${String(name)}`)
  }
}

export function validateLockfile(lockfile, packageVersion) {
  assertPlainObject(lockfile, 'Background-removal asset lockfile')

  if (lockfile.package !== PACKAGE_NAME) {
    throw new Error(`Asset lockfile package must be ${PACKAGE_NAME}`)
  }
  if (lockfile.version !== packageVersion) {
    throw new Error(
      `Asset lockfile version ${String(lockfile.version)} does not match ${PACKAGE_NAME} ${packageVersion}`,
    )
  }
  if (lockfile.baseUrl !== EXPECTED_BASE_URL) {
    throw new Error(`Asset lockfile base URL must be ${EXPECTED_BASE_URL}`)
  }

  assertPlainObject(lockfile.resources, 'Asset lockfile resources')
  const resourceNames = Object.keys(lockfile.resources).sort()
  const expectedResourceNames = [...REQUIRED_RESOURCES].sort()
  if (
    resourceNames.length !== expectedResourceNames.length ||
    resourceNames.some((name, index) => name !== expectedResourceNames[index])
  ) {
    throw new Error(
      `Asset lockfile must contain exactly these resources: ${REQUIRED_RESOURCES.join(', ')}`,
    )
  }

  const chunkNames = new Set()
  for (const resourceName of REQUIRED_RESOURCES) {
    const resource = lockfile.resources[resourceName]
    assertPlainObject(resource, `Resource ${resourceName}`)
    if (!Number.isSafeInteger(resource.size) || resource.size <= 0) {
      throw new Error(`Resource ${resourceName} has an invalid size`)
    }
    if (typeof resource.mime !== 'string' || resource.mime.length === 0) {
      throw new Error(`Resource ${resourceName} has an invalid MIME type`)
    }
    if (!Array.isArray(resource.chunks) || resource.chunks.length === 0) {
      throw new Error(`Resource ${resourceName} must contain chunks`)
    }

    let expectedOffset = 0
    for (const chunk of resource.chunks) {
      assertPlainObject(chunk, `Chunk in ${resourceName}`)
      assertChunkName(chunk.name)
      if (chunk.hash !== chunk.name) {
        throw new Error(`Chunk ${chunk.name} hash must match its content-addressed name`)
      }
      if (chunkNames.has(chunk.name)) {
        throw new Error(`Duplicate background-removal chunk: ${chunk.name}`)
      }
      chunkNames.add(chunk.name)

      if (
        !Array.isArray(chunk.offsets) ||
        chunk.offsets.length !== 2 ||
        !chunk.offsets.every(Number.isSafeInteger) ||
        chunk.offsets[0] !== expectedOffset ||
        chunk.offsets[1] <= chunk.offsets[0]
      ) {
        throw new Error(`Chunk ${chunk.name} has invalid offsets`)
      }
      expectedOffset = chunk.offsets[1]
    }

    if (expectedOffset !== resource.size) {
      throw new Error(`Resource ${resourceName} chunks do not match its declared size`)
    }
  }

  return lockfile
}

export function resolveAssetLocation(outPath, chunkName, baseUrl = EXPECTED_BASE_URL) {
  assertChunkName(chunkName)
  if (baseUrl !== EXPECTED_BASE_URL) {
    throw new Error(`Background-removal asset base URL must be ${EXPECTED_BASE_URL}`)
  }

  const url = new URL(chunkName, baseUrl)
  if (url.origin !== new URL(EXPECTED_BASE_URL).origin || !url.href.startsWith(EXPECTED_BASE_URL)) {
    throw new Error(`Background-removal asset URL escaped the trusted origin: ${url.href}`)
  }

  const root = resolve(outPath)
  const destination = resolve(root, chunkName)
  const relativeDestination = relative(root, destination)
  if (
    relativeDestination.length === 0 ||
    relativeDestination.startsWith('..') ||
    resolve(dirname(destination)) !== root
  ) {
    throw new Error(`Background-removal asset path escaped the output directory: ${chunkName}`)
  }

  return { destination, url }
}

export async function downloadVerifiedAsset({
  fetchImpl = fetch,
  baseUrl = EXPECTED_BASE_URL,
  outPath,
  chunk,
}) {
  const { destination, url } = resolveAssetLocation(outPath, chunk.name, baseUrl)
  const expectedSize = chunk.offsets[1] - chunk.offsets[0]
  const response = await fetchImpl(url, { redirect: 'error' })
  if (!response.ok) {
    throw new Error(`Failed to download ${url.href}: HTTP ${response.status}`)
  }
  if (response.url && response.url !== url.href) {
    throw new Error(`Background-removal asset response changed URL to ${response.url}`)
  }

  const contentLength = response.headers.get('content-length')
  if (contentLength !== null && Number(contentLength) !== expectedSize) {
    throw new Error(
      `Background-removal asset ${chunk.name} declared ${contentLength} bytes; expected ${expectedSize}`,
    )
  }

  if (!response.body) {
    throw new Error(`Background-removal asset ${chunk.name} returned an empty response body`)
  }

  const parts = []
  let receivedSize = 0
  for await (const part of response.body) {
    receivedSize += part.byteLength
    if (receivedSize > expectedSize) {
      throw new Error(
        `Background-removal asset ${chunk.name} exceeded its locked size of ${expectedSize} bytes`,
      )
    }
    parts.push(Buffer.from(part))
  }
  const bytes = Buffer.concat(parts, receivedSize)
  if (bytes.byteLength !== expectedSize) {
    throw new Error(
      `Background-removal asset ${chunk.name} contained ${bytes.byteLength} bytes; expected ${expectedSize}`,
    )
  }

  const digest = createHash('sha256').update(bytes).digest('hex')
  if (digest !== chunk.hash) {
    throw new Error(
      `Background-removal asset ${chunk.name} failed SHA-256 verification: received ${digest}`,
    )
  }

  await mkdir(dirname(destination), { recursive: true })
  const temporaryPath = `${destination}.${randomUUID()}.tmp`
  try {
    await writeFile(temporaryPath, bytes, { flag: 'wx' })
    await rename(temporaryPath, destination)
  } finally {
    await rm(temporaryPath, { force: true })
  }
}

export async function prepareBackgroundRemovalAssets({
  fetchImpl = fetch,
  packageJsonUrl = new URL('../package.json', import.meta.url),
  lockfileUrl = new URL('./background-removal-assets.lock.json', import.meta.url),
  outDir = new URL('../public/imgly-background-removal/', import.meta.url),
} = {}) {
  const packageJson = JSON.parse(await readFile(packageJsonUrl, 'utf8'))
  const packageVersion = packageJson.dependencies?.[PACKAGE_NAME]
  if (typeof packageVersion !== 'string' || !/^\d+\.\d+\.\d+$/.test(packageVersion)) {
    throw new Error(`${PACKAGE_NAME} must be pinned to an exact semantic version`)
  }

  const lockfile = validateLockfile(
    JSON.parse(await readFile(lockfileUrl, 'utf8')),
    packageVersion,
  )
  const outPath = resolve(fileURLToPath(outDir))
  const stagingPath = `${outPath}.${randomUUID()}.tmp`

  await mkdir(stagingPath, { recursive: true })
  try {
    for (const resource of Object.values(lockfile.resources)) {
      for (const chunk of resource.chunks) {
        await downloadVerifiedAsset({
          fetchImpl,
          baseUrl: lockfile.baseUrl,
          outPath: stagingPath,
          chunk,
        })
      }
    }

    await writeFile(
      resolve(stagingPath, 'resources.json'),
      `${JSON.stringify(lockfile.resources, null, 2)}\n`,
    )
    await rm(outPath, { recursive: true, force: true })
    await rename(stagingPath, outPath)
  } finally {
    await rm(stagingPath, { recursive: true, force: true })
  }

  const chunkCount = Object.values(lockfile.resources).reduce(
    (count, resource) => count + resource.chunks.length,
    0,
  )
  console.log(
    `Downloaded and verified ${chunkCount} background-removal asset chunks for ${PACKAGE_NAME} ${packageVersion}`,
  )
}

const invokedPath = process.argv[1] ? pathToFileURL(resolve(process.argv[1])).href : ''
if (invokedPath === import.meta.url) {
  await prepareBackgroundRemovalAssets()
}
