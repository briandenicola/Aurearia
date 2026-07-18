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
    expect(source).toContain('{{ relativeTime }}')
    expect(source).toContain('relativeTime')
    expect(source).toContain('No image')
    expect(source).not.toContain('<img')
  })

  it('constrains long draft names and metadata inside the PWA viewport', () => {
    expect(source).toContain('grid-cols-[64px_minmax(0,1fr)]')
    expect(source).toContain('min-[601px]:grid-cols-[76px_minmax(0,1fr)]')
    expect(source).toContain('min-w-0 overflow-hidden')
    expect(source).toContain('break-words')
    expect(source).toContain('chip-sm inline-block max-w-full truncate')
    expect(source).toContain('min-[601px]:')
  })
})
