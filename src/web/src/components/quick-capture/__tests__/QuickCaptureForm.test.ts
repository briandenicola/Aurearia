import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../QuickCaptureForm.vue'), 'utf8')

describe('QuickCaptureForm', () => {
  it('requires title, note, or image before enabling save', () => {
    expect(source).toContain('canSave')
    expect(source).toContain('workingTitle.value.trim()')
    expect(source).toContain('notes.value.trim()')
    expect(source).toContain('detailImages.value.length')
  })
})
