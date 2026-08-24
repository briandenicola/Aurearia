import { removeBackground } from '@imgly/background-removal'
import { backgroundRemovalConfig } from './backgroundRemovalConfig'

// Message protocol between src/utils/backgroundRemoval.ts (the client, running
// on the main thread) and this dedicated module worker. Kept as plain,
// structured-clone-safe shapes (no class instances) so they survive
// postMessage in both directions.
export interface BackgroundRemovalWorkerRequest {
  id: string
  image: Blob
}

export type BackgroundRemovalWorkerResponse =
  | { id: string; ok: true; result: Blob }
  | { id: string; ok: false; error: { name: string; message: string } }

function toWireError(err: unknown): { name: string; message: string } {
  if (err instanceof Error) {
    return { name: err.name, message: err.message }
  }
  return { name: 'Error', message: String(err) }
}

self.addEventListener('message', (event: MessageEvent<BackgroundRemovalWorkerRequest>) => {
  const { id, image } = event.data
  removeBackground(image, backgroundRemovalConfig).then(
    (result) => {
      const response: BackgroundRemovalWorkerResponse = { id, ok: true, result }
      self.postMessage(response)
    },
    (err: unknown) => {
      const response: BackgroundRemovalWorkerResponse = { id, ok: false, error: toWireError(err) }
      self.postMessage(response)
    },
  )
})
