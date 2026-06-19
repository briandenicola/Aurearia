const ALLOWED_PROTOCOLS = new Set(['http:', 'https:'])

// Use for user/API-provided external anchors; internal navigation stays on router-link.
export function sanitizeExternalUrl(url: string | null | undefined): string | null {
  if (!url) return null

  const trimmedUrl = url.trim()
  if (!trimmedUrl) return null

  try {
    const parsed = new URL(trimmedUrl)
    if (!ALLOWED_PROTOCOLS.has(parsed.protocol.toLowerCase())) {
      return null
    }
    return parsed.toString()
  } catch {
    return null
  }
}

export function useSafeExternalLink(url: string | null | undefined): string | null {
  return sanitizeExternalUrl(url)
}
