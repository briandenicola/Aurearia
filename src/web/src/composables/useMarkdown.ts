import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

export function renderSafeMarkdown(source: string | null | undefined): string {
  if (!source) return ''
  return DOMPurify.sanitize(md.render(source))
}
