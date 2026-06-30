import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../QuickCaptureDraftCard.vue'), 'utf8')

describe('QuickCaptureDraftCard', () => {
  it('renders preview media through authenticated owner-safe URLs and links to resume', () => {
    expect(source).toContain('AuthenticatedImage')
    expect(source).toContain(':media-path="previewImage.filePath"')
    expect(source).toContain('/quick-capture/drafts/')
    expect(source).toContain('RouterLink')
  })

  it('shows incomplete context, updated time, and empty-image fallback without leaking raw img URLs', () => {
    expect(source).toContain('Incomplete Quick Capture draft')
    expect(source).toContain('renderSafeMarkdown')
    expect(source).toContain('v-html="renderedNotes"')
    expect(source).toContain('updated-at')
    expect(source).toContain('relativeTime')
    expect(source).toContain('No image')
    expect(source).not.toContain('<img')
  })

  it('constrains long draft names and metadata inside the PWA viewport', () => {
    expect(source).toContain('grid-template-columns: 76px minmax(0, 1fr)')
    expect(source).toContain('.draft-info {')
    expect(source).toContain('overflow-wrap: anywhere')
    expect(source).toContain('.draft-meta .chip-sm')
    expect(source).toContain('text-overflow: ellipsis')
    expect(source).toContain('@media (max-width: 600px)')
  })
})
