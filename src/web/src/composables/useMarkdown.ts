import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'

// The single MarkdownIt instance for the whole app. `html: false` makes
// markdown-it escape raw HTML in the source rather than pass it through, so
// DOMPurify below is the second of two independent barriers rather than the
// only one. Do not construct MarkdownIt anywhere else — a second instance is
// a second place for these options to drift.
const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

/**
 * Renders Markdown the user owns or that has already been reviewed — notes,
 * quick-capture drafts, saved AI analysis — using DOMPurify's default
 * allowlist.
 */
export function renderSafeMarkdown(source: string | null | undefined): string {
  if (!source) return ''
  return DOMPurify.sanitize(md.render(source))
}

// Live agent output can contain anything the model emits, including text
// echoed from fetched web pages. It gets a hand-picked tag allowlist rather
// than DOMPurify's default so that a sanitizer-default change can never widen
// what a chat reply is allowed to render.
const CHAT_ALLOWED_TAGS = [
  'strong', 'em', 'br', 'p', 'ul', 'ol', 'li', 'a',
  'h1', 'h2', 'h3', 'h4', 'code', 'pre', 'blockquote', 'hr',
]
const CHAT_ALLOWED_ATTR = ['href', 'target', 'rel']

/**
 * Renders untrusted, un-reviewed Markdown streamed back from the agent. Use
 * this for anything shown before a human has approved it; use
 * `renderSafeMarkdown` once it has been saved.
 */
export function renderSafeChatMarkdown(source: string | null | undefined): string {
  if (!source) return ''
  return DOMPurify.sanitize(md.render(source), {
    ALLOWED_TAGS: CHAT_ALLOWED_TAGS,
    ALLOWED_ATTR: CHAT_ALLOWED_ATTR,
  })
}
