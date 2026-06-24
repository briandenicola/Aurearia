import { createWriteStream } from 'node:fs'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { pipeline } from 'node:stream/promises'
import { fileURLToPath } from 'node:url'

const packageJson = JSON.parse(await readFile(new URL('../package.json', import.meta.url), 'utf8'))
const version = packageJson.dependencies['@imgly/background-removal'].replace(/^[^\d]*/, '')
const baseUrl = `https://staticimgly.com/@imgly/background-removal-data/${version}/dist/`
const outDir = new URL('../public/imgly-background-removal/', import.meta.url)
const outPath = fileURLToPath(outDir)
const requiredResources = [
  '/onnxruntime-web/ort-wasm-simd-threaded.wasm',
  '/onnxruntime-web/ort-wasm-simd-threaded.mjs',
  '/models/isnet_quint8',
]

async function download(url, destination) {
  const response = await fetch(url)
  if (!response.ok || !response.body) {
    throw new Error(`Failed to download ${url}: HTTP ${response.status}`)
  }
  await mkdir(dirname(destination), { recursive: true })
  await pipeline(response.body, createWriteStream(destination))
}

const resourcesResponse = await fetch(new URL('resources.json', baseUrl))
if (!resourcesResponse.ok) {
  throw new Error(`Failed to download resources.json: HTTP ${resourcesResponse.status}`)
}

const resources = await resourcesResponse.json()
await mkdir(outDir, { recursive: true })
await writeFile(new URL('resources.json', outDir), `${JSON.stringify(resources, null, 2)}\n`)

const chunkNames = new Set()
for (const key of requiredResources) {
  const resource = resources[key]
  if (!resource) {
    throw new Error(`Required background-removal resource is missing: ${key}`)
  }
  for (const chunk of resource.chunks) {
    chunkNames.add(chunk.name)
  }
}

for (const chunkName of chunkNames) {
  await download(new URL(chunkName, baseUrl), join(outPath, chunkName))
}

console.log(`Downloaded ${chunkNames.size} background-removal asset chunks for @imgly/background-removal ${version}`)
