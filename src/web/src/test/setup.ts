import { Blob as NodeBlob, File as NodeFile } from 'node:buffer'

// jsdom's Blob/File implementation lacks .stream(), which Node's native
// Response (used by fetch mocks in tests) requires. Use Node's native
// Blob/File in the test environment so they interop correctly — this
// matches real browsers, where Blob and Response come from the same platform.
globalThis.Blob = NodeBlob as unknown as typeof Blob
globalThis.File = NodeFile as unknown as typeof File
