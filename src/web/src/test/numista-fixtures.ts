import type {
  NumistaCandidate,
  NumistaEnrichmentRequest,
  NumistaEvidence,
  NumistaLookupOutcome,
  NumistaLookupRequest,
  SelectedNumistaReference,
} from '@/types'

export function makeNumistaEvidence(overrides: Partial<NumistaEvidence> = {}): NumistaEvidence {
  return {
    title: 'Antoninus Pius denarius',
    issuer: 'Antoninus Pius',
    denomination: 'Denarius',
    mint: 'Rome',
    dateText: '138–161',
    material: 'Silver',
    obverseInscription: 'ANTONINVS AVG PIVS',
    reverseInscription: 'TR POT COS III',
    ...overrides,
  }
}

export function makeNumistaLookupRequest(overrides: Partial<NumistaLookupRequest> = {}): NumistaLookupRequest {
  return {
    query: 'Antoninus Pius denarius Rome 138–161 Silver',
    path: 'direct',
    evidence: makeNumistaEvidence(),
    ...overrides,
  }
}

export function makeNumistaEnrichmentRequest(
  overrides: Partial<NumistaEnrichmentRequest> = {},
): NumistaEnrichmentRequest {
  return {
    ...makeNumistaLookupRequest(),
    candidates: [makeNumistaCandidate({ enrichmentState: 'not_requested' })],
    ...overrides,
  }
}

export function makeNumistaCandidate(overrides: Partial<NumistaCandidate> = {}): NumistaCandidate {
  return {
    id: 12345,
    canonicalUrl: 'https://en.numista.com/catalogue/pieces12345.html',
    title: 'Denarius - Antoninus Pius',
    issuer: 'Roman Empire',
    denomination: 'Denarius',
    mint: 'Rome',
    yearDisplay: '138–161',
    material: 'Silver',
    providerPosition: 0,
    enrichmentState: 'enriched',
    assessment: {
      scoringVersion: 'numista-v1',
      score: 91,
      band: 'strong',
      reasons: [
        { field: 'denomination', kind: 'match', code: 'denomination_match', label: 'Denomination matches' },
      ],
    },
    ...overrides,
  }
}

export function makeNumistaLookupOutcome(overrides: Partial<NumistaLookupOutcome> = {}): NumistaLookupOutcome {
  return {
    status: 'success',
    effectiveQuery: 'Antoninus Pius denarius Rome 138–161 Silver',
    candidates: [makeNumistaCandidate()],
    cache: {
      hit: false,
      coalesced: false,
      createdAt: '2026-08-11T12:00:00Z',
      expiresAt: '2026-08-12T12:00:00Z',
      ageSeconds: 0,
    },
    stage: 'broad',
    ...overrides,
  }
}

export function makeSelectedNumistaReference(
  overrides: Partial<SelectedNumistaReference> = {},
): SelectedNumistaReference {
  return {
    catalog: 'Numista',
    number: '12345',
    uri: 'https://en.numista.com/catalogue/pieces12345.html',
    ...overrides,
  }
}
