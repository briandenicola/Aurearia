import type {
  Coin,
  NumistaCandidate,
  NumistaEvidence,
  NumistaLookupStatus,
} from '@/types'

const QUERY_LIMIT = 500

type DirectCoinEvidence = Pick<
  Coin,
  'name' | 'ruler' | 'denomination' | 'mint' | 'dateRange' | 'material' | 'obverseInscription' | 'reverseInscription'
>

export interface NumistaStatusGuidance {
  title: string
  message: string
  canRetry: boolean
  settingsHref?: string
}

function bounded(value: string | null | undefined, limit: number): string | undefined {
  const trimmed = value?.trim()
  return trimmed ? trimmed.slice(0, limit) : undefined
}

export function buildDirectNumistaEvidence(coin: DirectCoinEvidence): NumistaEvidence {
  return {
    title: bounded(coin.name, 200),
    issuer: bounded(coin.ruler, 200),
    denomination: bounded(coin.denomination, 100),
    mint: bounded(coin.mint, 200),
    dateText: bounded(coin.dateRange, 100),
    material: bounded(coin.material, 100),
    obverseInscription: bounded(coin.obverseInscription, 500),
    reverseInscription: bounded(coin.reverseInscription, 500),
  }
}

export function buildNumistaQuery(evidence: NumistaEvidence): string {
  return [
    evidence.title,
    evidence.issuer,
    evidence.denomination,
    evidence.mint,
    evidence.dateText,
    evidence.material,
    evidence.obverseInscription,
    evidence.reverseInscription,
    evidence.visibleText,
  ].filter((value): value is string => Boolean(value?.trim()))
    .join(' ')
    .slice(0, QUERY_LIMIT)
}

export function buildDirectNumistaQuery(coin: DirectCoinEvidence): string {
  return buildNumistaQuery(buildDirectNumistaEvidence(coin))
}

export function numistaCandidateIdentity(candidate: Pick<NumistaCandidate, 'id'>): string {
  return `numista:${candidate.id}`
}

export function retainNumistaSelection(
  selected: NumistaCandidate | null,
  candidates: NumistaCandidate[],
): NumistaCandidate | null {
  if (!selected) return null
  return candidates.find(candidate => candidate.id === selected.id) ?? selected
}

export function isSelectionOutsideResults(
  selected: NumistaCandidate | null,
  candidates: NumistaCandidate[],
): boolean {
  return selected !== null && !candidates.some(candidate => candidate.id === selected.id)
}

export function getNumistaStatusGuidance(
  status: NumistaLookupStatus,
  isAdmin: boolean,
  retryAfterSeconds?: number,
): NumistaStatusGuidance | null {
  switch (status) {
    case 'success':
      return null
    case 'empty':
      return {
        title: 'No matches found',
        message: 'Revise the query and search again.',
        canRetry: true,
      }
    case 'unconfigured':
      return {
        title: 'Numista lookup is not configured',
        message: isAdmin
          ? 'Configure the Numista integration in Settings, then retry.'
          : 'Numista lookup is not available on this instance. Contact an administrator.',
        canRetry: false,
        settingsHref: isAdmin ? '/settings?tab=connections' : undefined,
      }
    case 'quota-limited':
      return {
        title: 'Numista lookup limit reached',
        message: retryAfterSeconds
          ? `Try again in about ${retryAfterSeconds} seconds.`
          : 'Wait before trying again.',
        canRetry: true,
      }
    case 'timeout':
      return {
        title: 'Numista lookup timed out',
        message: 'The query is still available. Try again.',
        canRetry: true,
      }
    case 'unavailable':
      return {
        title: 'Numista lookup is unavailable',
        message: 'The service could not complete the lookup. Try again later.',
        canRetry: true,
      }
  }
}
