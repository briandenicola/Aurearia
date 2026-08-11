import type {
  Coin,
  NumistaCandidate,
  NumistaCacheMetadata,
  NumistaEvidence,
  NumistaLookupStatus,
  SelectedNumistaReference,
} from '@/types'

const QUERY_LIMIT = 500

export type NumistaPanelState = 'idle' | 'loading' | NumistaLookupStatus

type DirectCoinEvidence = Pick<
  Coin,
  'name' | 'ruler' | 'denomination' | 'mint' | 'dateRange' | 'material' | 'obverseInscription' | 'reverseInscription'
>

export interface NumistaStatusGuidance {
  state: NumistaPanelState
  label: string
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

export function selectedNumistaReferenceFromCandidate(
  candidate: NumistaCandidate,
): SelectedNumistaReference {
  return {
    catalog: 'Numista',
    number: String(candidate.id),
    uri: candidate.canonicalUrl,
  }
}

export function numistaCandidateFromReference(
  reference: SelectedNumistaReference | null | undefined,
): NumistaCandidate | null {
  if (!reference) return null
  const id = Number(reference.number)
  if (!Number.isSafeInteger(id) || id <= 0) return null

  return {
    id,
    canonicalUrl: reference.uri,
    title: `Numista #${reference.number}`,
    providerPosition: 0,
    enrichmentState: 'not_requested',
    assessment: {
      scoringVersion: 'numista-v1',
      score: 50,
      band: 'weak',
      reasons: [],
    },
  }
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
  state: NumistaPanelState,
  isAdmin: boolean,
  retryAfterSeconds?: number,
): NumistaStatusGuidance {
  switch (state) {
    case 'idle':
      return {
        state,
        label: 'Ready',
        title: 'Ready to search',
        message: 'Review the query, then search when you are ready.',
        canRetry: false,
      }
    case 'loading':
      return {
        state,
        label: 'Searching',
        title: 'Searching Numista',
        message: 'Looking for ranked catalog candidates.',
        canRetry: false,
      }
    case 'success':
      return {
        state,
        label: 'Results ready',
        title: 'Matches ready for review',
        message: 'Review the evidence and explicitly select a catalog reference.',
        canRetry: false,
      }
    case 'empty':
      return {
        state,
        label: 'No matches',
        title: 'No matches found',
        message: 'Revise the query and search again.',
        canRetry: true,
      }
    case 'unconfigured':
      return {
        state,
        label: 'Setup needed',
        title: 'Numista lookup is not configured',
        message: isAdmin
          ? 'Add the Numista API key in Admin System settings, then retry the lookup.'
          : 'Numista lookup is not available on this instance. Ask an administrator for help.',
        canRetry: isAdmin,
        settingsHref: isAdmin ? '/admin?tab=system' : undefined,
      }
    case 'quota-limited':
      return {
        state,
        label: 'Temporarily limited',
        title: 'Numista lookup limit reached',
        message: retryAfterSeconds
          ? `Numista asked the app to wait ${formatDuration(retryAfterSeconds)} before retrying.`
          : 'Wait before trying again.',
        canRetry: true,
      }
    case 'timeout':
      return {
        state,
        label: 'Timed out',
        title: 'Numista lookup timed out',
        message: 'The query and selection are unchanged. Retry when ready.',
        canRetry: true,
      }
    case 'unavailable':
      return {
        state,
        label: 'Unavailable',
        title: 'Numista lookup is unavailable',
        message: 'The service could not complete the lookup. Your query and selection are unchanged.',
        canRetry: true,
      }
  }
}

export function getNumistaCacheFreshnessText(
  status: NumistaLookupStatus | null,
  cache: NumistaCacheMetadata | null | undefined,
): string | null {
  if (!cache || (status !== 'success' && status !== 'empty')) return null
  if (!cache.hit) return 'Fresh results'
  return `Cached results · ${formatAge(cache.ageSeconds)} old`
}

function formatAge(seconds: number): string {
  if (seconds < 60) return `${seconds} sec`
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} hr`
  return `${Math.floor(seconds / 86400)} day${seconds < 172800 ? '' : 's'}`
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds} second${seconds === 1 ? '' : 's'}`
  if (seconds % 60 === 0 && seconds < 3600) {
    const minutes = seconds / 60
    return `${minutes} minute${minutes === 1 ? '' : 's'}`
  }
  return `${seconds} seconds`
}
