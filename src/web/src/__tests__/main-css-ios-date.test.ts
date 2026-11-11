import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'fs'
import { join } from 'path'

// Regression guard: iOS Safari / installed PWA renders date-like inputs with an
// intrinsic min-width, so they bleed out of their field box unless the global
// rule in main.css neutralises it.
const SRC_DIR = join(__dirname, '..')
const mainCss = readFileSync(join(SRC_DIR, 'assets', 'styles', 'main.css'), 'utf-8')

function vueFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const full = join(dir, name)
    if (name === 'node_modules' || name === '__tests__') return []
    if (statSync(full).isDirectory()) return vueFiles(full)
    return name.endsWith('.vue') ? [full] : []
  })
}

describe('iOS date input overflow guard', () => {
  const rule = mainCss.match(/input\[type='date'\],[\s\S]*?\{([\s\S]*?)\}/)

  it('keeps a global rule for date-like inputs in main.css', () => {
    expect(rule).not.toBeNull()
    for (const type of ['date', 'datetime-local', 'time', 'month']) {
      expect(mainCss).toContain(`input[type='${type}']`)
    }
    const body = rule![1]
    expect(body).toMatch(/-webkit-appearance:\s*none/)
    expect(body).toMatch(/min-width:\s*0/)
    expect(body).toMatch(/max-width:\s*100%/)
    expect(mainCss).toContain('::-webkit-date-and-time-value')
  })

  it('does not let components re-enable a native min-width on date inputs', () => {
    const offenders: string[] = []
    for (const file of vueFiles(SRC_DIR)) {
      const source = readFileSync(file, 'utf-8')
      for (const match of source.matchAll(/<input[^>]*type="(?:date|datetime-local|time|month)"[^>]*>/g)) {
        if (/\b(?:appearance-auto|min-w-(?!0\b)[\w[\]-]+|inline-block|inline-flex)\b/.test(match[0])) {
          offenders.push(`${file.replace(SRC_DIR, 'src')}: ${match[0]}`)
        }
      }
    }
    expect(offenders).toEqual([])
  })
})
