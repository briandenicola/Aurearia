import { describe, it, expect } from 'vitest'
import { renderSafeMarkdown, renderSafeChatMarkdown } from '@/composables/useMarkdown'


/**
 * Parses rendered HTML and returns a list of anything executable that survived
 * sanitization. An empty array means the markup is inert.
 */
function assertInert(html: string): string[] {
  const doc = new DOMParser().parseFromString(html, 'text/html')
  const problems: string[] = []

  for (const selector of ['script', 'iframe', 'object', 'embed', 'form']) {
    if (doc.querySelector(selector)) problems.push(`<${selector}> element present`)
  }
  for (const el of Array.from(doc.querySelectorAll('*'))) {
    for (const attr of Array.from(el.attributes)) {
      if (attr.name.toLowerCase().startsWith('on')) {
        problems.push(`event handler ${attr.name} on <${el.tagName.toLowerCase()}>`)
      }
      if (['href', 'src', 'xlink:href'].includes(attr.name.toLowerCase())) {
        const value = attr.value.trim().toLowerCase().replace(/\s+/g, '')
        if (value.startsWith('javascript:') || value.startsWith('data:text/html')) {
          problems.push(`dangerous URL in ${attr.name}: ${attr.value}`)
        }
      }
    }
  }
  return problems
}

describe('renderSafeMarkdown', () => {
  it('renders ordinary markdown', () => {
    expect(renderSafeMarkdown('**Trajan** denarius')).toContain('<strong>Trajan</strong>')
  })

  it('returns an empty string for nullish input', () => {
    expect(renderSafeMarkdown(null)).toBe('')
    expect(renderSafeMarkdown(undefined)).toBe('')
    expect(renderSafeMarkdown('')).toBe('')
  })

  // These assert the real property — that nothing *live* reaches the DOM.
  // Checking for substrings would be misleading: `html: false` escapes raw
  // HTML rather than deleting it, so "<script>" survives as inert
  // "&lt;script&gt;" text and a naive substring check would fail on safe output.
  it.each([
    ['script tag', 'hello <script>alert(1)</script> world'],
    ['img with onerror', '<img src=x onerror="alert(1)">'],
    ['iframe', '<iframe src="https://evil.example"></iframe>'],
    ['svg onload', '<svg onload="alert(1)"></svg>'],
    ['javascript: link', '[click](javascript:alert(1))'],
    ['data: link', '[click](data:text/html;base64,PHNjcmlwdD4=)'],
  ])('yields no executable content for %s', (_label, source) => {
    expect(assertInert(renderSafeMarkdown(source))).toEqual([])
  })
})

describe('renderSafeChatMarkdown', () => {
  it('allows the formatting tags agent replies need', () => {
    const html = renderSafeChatMarkdown('**bold** and `code`')
    expect(html).toContain('<strong>bold</strong>')
    expect(html).toContain('<code>code</code>')
  })

  // The chat allowlist is deliberately narrower than DOMPurify's default:
  // agent output can echo text from fetched pages, so it must not be able to
  // introduce embedded content even if the sanitizer's defaults widen later.
  it('drops tags outside its allowlist that the default renderer would keep', () => {
    const source = '<img src="https://example.com/x.png"> <table><tr><td>c</td></tr></table>'
    const chat = renderSafeChatMarkdown(source)
    expect(chat).not.toContain('<img')
    expect(chat).not.toContain('<table')
  })

  it.each([
    ['script tag', 'hi <script>alert(1)</script>'],
    ['img with onerror', '<img src=x onerror="alert(1)">'],
    ['javascript: link', '[click](javascript:alert(1))'],
  ])('yields no executable content for %s', (_label, source) => {
    expect(assertInert(renderSafeChatMarkdown(source))).toEqual([])
  })

  it('returns an empty string for nullish input', () => {
    expect(renderSafeChatMarkdown(null)).toBe('')
    expect(renderSafeChatMarkdown(undefined)).toBe('')
  })
})
