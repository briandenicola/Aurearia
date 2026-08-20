import { describe, it, expect } from 'vitest'
import { renderSafeMarkdown, renderSafeChatMarkdown } from '@/composables/useMarkdown'


/**
 * Safe URL schemes for a rendered link or image.
 *
 * This is an allowlist on purpose. An earlier version denylisted `javascript:`
 * and `data:text/html`, which CodeQL correctly flagged as incomplete
 * (js/incomplete-url-scheme-check): it silently accepted `vbscript:`, other
 * `data:` payloads, and anything else novel. Since this helper is the oracle
 * that proves sanitized output is inert, a gap here means a regression could
 * pass the suite unnoticed — so it enumerates what is allowed rather than
 * guessing at what is not.
 */
const SAFE_URL_SCHEMES = ['http:', 'https:', 'mailto:']

/** Matches a leading URI scheme per RFC 3986, tolerating obfuscation whitespace. */
const SCHEME_PATTERN = /^[a-z][a-z0-9+.-]*:/i

function hasSafeScheme(rawValue: string): boolean {
  // Browsers ignore whitespace and control characters inside a scheme, so a
  // value like "java\tscript:alert(1)" still executes. Strip those the way a
  // browser would before testing, or this check is trivially bypassed.
  const value = rawValue.replace(/[\s\u0000-\u001F]/g, '').toLowerCase()
  const match = SCHEME_PATTERN.exec(value)
  // No scheme at all: a relative path, anchor, or protocol-relative URL.
  if (!match) return true
  return SAFE_URL_SCHEMES.includes(match[0])
}


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
        if (!hasSafeScheme(attr.value)) {
          problems.push(`unsafe URL scheme in ${attr.name}: ${attr.value}`)
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
    ['vbscript: link', '[click](vbscript:msgbox(1))'],
    ['tab-obfuscated javascript:', '<a href="java\tscript:alert(1)">x</a>'],
  ])('yields no executable content for %s', (_label, source) => {
    expect(assertInert(renderSafeMarkdown(source))).toEqual([])
  })

  it('permits the URL schemes real content uses', () => {
    for (const source of [
      '[site](https://example.com)',
      '[site](http://example.com)',
      '[mail](mailto:someone@example.com)',
      '[anchor](#provenance)',
      '[relative](/coin/42)',
    ]) {
      expect(assertInert(renderSafeMarkdown(source))).toEqual([])
    }
  })
})

// The oracle above is what proves the renderers are safe, so it needs its own
// proof. CodeQL flagged an earlier denylist version of hasSafeScheme as
// incomplete (js/incomplete-url-scheme-check) — it would have accepted
// vbscript:. These cases fail if the allowlist ever regresses to a denylist.
describe('assertInert (the test oracle itself)', () => {
  it.each([
    ['javascript:', '<a href="javascript:alert(1)">x</a>'],
    ['vbscript:', '<a href="vbscript:msgbox(1)">x</a>'],
    ['data:text/html', '<a href="data:text/html,<script>alert(1)</script>">x</a>'],
    ['data:image/svg+xml', '<a href="data:image/svg+xml,<svg onload=alert(1)>">x</a>'],
    ['file:', '<a href="file:///etc/passwd">x</a>'],
    ['whitespace-obfuscated', '<a href="java\tscript:alert(1)">x</a>'],
    ['newline-obfuscated', '<a href="java\nscript:alert(1)">x</a>'],
    ['uppercase', '<a href="JaVaScRiPt:alert(1)">x</a>'],
  ])('flags a raw %s URL that survived to the DOM', (_label, html) => {
    expect(assertInert(html).length).toBeGreaterThan(0)
  })

  it('flags script elements and event handlers', () => {
    expect(assertInert('<script>alert(1)</script>').length).toBeGreaterThan(0)
    expect(assertInert('<div onclick="alert(1)">x</div>').length).toBeGreaterThan(0)
  })

  it('passes genuinely inert markup', () => {
    expect(assertInert('<p><strong>Trajan</strong> <a href="https://example.com">ref</a></p>')).toEqual([])
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
