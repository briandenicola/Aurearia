import { Blob as NodeBlob, File as NodeFile } from 'node:buffer'
import { beforeEach } from 'vitest'

// jsdom's Blob/File implementation lacks .stream(), which Node's native
// Response (used by fetch mocks in tests) requires. Use Node's native
// Blob/File in the test environment so they interop correctly — this
// matches real browsers, where Blob and Response come from the same platform.
globalThis.Blob = NodeBlob as unknown as typeof Blob
globalThis.File = NodeFile as unknown as typeof File

beforeEach(() => {
  for (const id of ['desktop-page-title', 'desktop-page-actions']) {
    const existing = document.getElementById(id)
    if (existing) {
      existing.replaceChildren()
      continue
    }

    const target = document.createElement('div')
    target.id = id
    document.body.appendChild(target)
  }
})
