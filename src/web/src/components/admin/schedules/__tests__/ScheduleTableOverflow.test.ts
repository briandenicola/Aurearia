import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const schedulesDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')

// Admin run-history tables carry 6-8 columns. Without an overflow-x-auto wrapper they burst
// their card on a narrow PWA viewport and the right-hand columns (Duration, Errors) are
// simply unreachable — there is nothing to scroll. This scans the whole directory rather
// than naming components, so a new schedule section is covered the day it lands.
describe('admin schedule run-history tables', () => {
  const files = fs.readdirSync(schedulesDir).filter(name => name.endsWith('.vue'))

  it('has schedule components to check', () => {
    expect(files.length).toBeGreaterThan(5)
  })

  it.each(files)('%s wraps every table in an overflow-x-auto container', file => {
    const lines = fs.readFileSync(path.join(schedulesDir, file), 'utf8').split('\n')

    lines.forEach((line, index) => {
      if (!/^\s*<table(\s|>)/.test(line)) return
      const wrapper = lines[index - 1] ?? ''
      expect(
        wrapper.includes('overflow-x-auto'),
        `${file}:${index + 1} opens a table that is not inside an overflow-x-auto wrapper`,
      ).toBe(true)
    })
  })
})
