import { MATERIALS, type CoinLookupResponse, type CoinMutationPayload, type Material } from '@/types'

type NormalizableLookupField =
  | 'name'
  | 'ruler'
  | 'denomination'
  | 'era'
  | 'mint'
  | 'material'
  | 'category'
  | 'grade'
  | 'obverseInscription'
  | 'reverseInscription'
  | 'obverseDescription'
  | 'reverseDescription'

const lookupFieldAliases: Record<NormalizableLookupField, string[]> = {
  name: ['name', 'coin name', 'title', 'attribution'],
  ruler: ['ruler', 'issuer', 'emperor', 'authority'],
  denomination: ['denomination', 'coin type', 'type'],
  era: ['era', 'period'],
  mint: ['mint', 'mint location'],
  material: ['material', 'metal', 'composition'],
  category: ['category', 'culture', 'region'],
  grade: ['grade', 'condition'],
  obverseInscription: ['obverse inscription', 'obverse legend'],
  reverseInscription: ['reverse inscription', 'reverse legend'],
  obverseDescription: ['obverse description', 'obverse'],
  reverseDescription: ['reverse description', 'reverse'],
}

function normalizeLookupKey(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9]/g, '')
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return null
  return value as Record<string, unknown>
}

function cleanLookupValue(value: unknown) {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : undefined
}

function isPlaceholderLookupValue(value: string | undefined) {
  if (!value) return true
  const normalized = value.trim().toLowerCase()
  return normalized === '' || normalized === 'unidentified coin' || normalized === 'unknown' || normalized === 'n/a'
}

function getAliasedField(source: Record<string, unknown> | null | undefined, field: NormalizableLookupField) {
  if (!source) return undefined
  const aliases = new Set(lookupFieldAliases[field].map(normalizeLookupKey))
  const entry = Object.entries(source).find(([key]) => aliases.has(normalizeLookupKey(key)))
  return entry ? cleanLookupValue(entry[1]) : undefined
}

function parseLookupLineFields(text: string | undefined) {
  const fields: Partial<Record<NormalizableLookupField, string>> = {}
  if (!text) return fields

  for (const line of text.split(/\r?\n/)) {
    const match = line.match(/^\s*([A-Za-z][A-Za-z\s/()-]{0,40})\s*:\s*(.+?)\s*$/)
    if (!match) continue

    const label = match[1] ?? ''
    const value = cleanLookupValue(match[2])
    if (!value) continue

    const normalizedLabel = normalizeLookupKey(label)
    const field = Object.entries(lookupFieldAliases).find(([, aliases]) =>
      aliases.map(normalizeLookupKey).includes(normalizedLabel)
    )?.[0] as NormalizableLookupField | undefined

    if (field && !fields[field]) {
      fields[field] = value
    }
  }

  return fields
}

function parseJsonLookupFields(text: string | undefined) {
  if (!text) return null
  try {
    return asRecord(JSON.parse(text))
  } catch {
    return null
  }
}

function normalizeMaterial(value: string): Material {
  const normalized = value.trim().toLowerCase()
  const aliases: Record<string, Material> = {
    ar: 'Silver',
    silver: 'Silver',
    ae: 'Bronze',
    bronze: 'Bronze',
    copper: 'Copper',
    au: 'Gold',
    gold: 'Gold',
    electrum: 'Electrum',
  }
  return aliases[normalized] ?? MATERIALS.find(material => material.toLowerCase() === normalized) ?? 'Other'
}

function normalizeObservationForCompare(value: string) {
  return value
    .toLowerCase()
    .replace(/[`*_>#-]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

export function appendUniqueObservation(parts: string[], value: string | undefined, heading?: string) {
  const clean = cleanLookupValue(value)
  if (!clean) return

  const normalizedClean = normalizeObservationForCompare(clean)
  const isDuplicate = parts.some(part => {
    const normalizedPart = normalizeObservationForCompare(part)
    return normalizedPart.includes(normalizedClean) || normalizedClean.includes(normalizedPart)
  })
  if (isDuplicate) return

  parts.push(heading ? `**${heading}:** ${clean}` : clean)
}

export function deriveAiObservations(lookup: CoinLookupResponse, draft: CoinMutationPayload) {
  const parts: string[] = []

  appendUniqueObservation(parts, draft.notes)
  appendUniqueObservation(parts, draft.aiAnalysis)
  if (!parseJsonLookupFields(lookup.extractedData.rawAnalysis)) {
    appendUniqueObservation(parts, lookup.extractedData.rawAnalysis)
  }
  appendUniqueObservation(parts, draft.obverseDescription, 'Obverse')
  appendUniqueObservation(parts, draft.reverseDescription, 'Reverse')

  return parts.join('\n\n')
}

function hasFieldValue(draft: CoinMutationPayload, field: NormalizableLookupField) {
  const value = draft[field]
  return typeof value === 'string' && !isPlaceholderLookupValue(value)
}

function setMissingLookupField(draft: CoinMutationPayload, field: NormalizableLookupField, value: string | undefined) {
  if (!value || hasFieldValue(draft, field)) return

  switch (field) {
    case 'name':
      draft.name = value
      break
    case 'ruler':
      draft.ruler = value
      break
    case 'denomination':
      draft.denomination = value
      break
    case 'era':
      draft.era = value
      break
    case 'mint':
      draft.mint = value
      break
    case 'material':
      draft.material = normalizeMaterial(value)
      break
    case 'category':
      draft.category = value
      break
    case 'grade':
      draft.grade = value
      break
    case 'obverseInscription':
      draft.obverseInscription = value
      break
    case 'reverseInscription':
      draft.reverseInscription = value
      break
    case 'obverseDescription':
      draft.obverseDescription = value
      break
    case 'reverseDescription':
      draft.reverseDescription = value
      break
  }
}

function applyLookupFieldSource(draft: CoinMutationPayload, source: Record<string, unknown> | null | undefined) {
  for (const field of Object.keys(lookupFieldAliases) as NormalizableLookupField[]) {
    setMissingLookupField(draft, field, getAliasedField(source, field))
  }
}

function applyParsedLookupText(draft: CoinMutationPayload, text: string | undefined) {
  applyLookupFieldSource(draft, parseJsonLookupFields(text))
  const parsedLines = parseLookupLineFields(text)
  for (const [field, value] of Object.entries(parsedLines) as Array<[NormalizableLookupField, string]>) {
    setMissingLookupField(draft, field, value)
  }
}

function deriveNameFromParts(draft: CoinMutationPayload) {
  if (hasFieldValue(draft, 'name')) return
  const parts = [draft.ruler, draft.denomination].filter((part): part is string => Boolean(part?.trim()))
  if (parts.length > 0) {
    draft.name = parts.join(' ')
  }
}

function reliableNgcLabelName(labelText: string | undefined) {
  if (!labelText) return undefined
  const unreliable = /\b(ngc|cert|certification|ancients|authentic|grade|ch\s*vf|vf|xf|ms|fine)\b/i
  const contextOnly = /\b(empire|kingdom|republic|provincial|mint)\b/i
  const dateOnly = /^(?:c\.?\s*)?(?:ad|bc|ce|bce)?\s*[\d\s./-]+(?:ad|bc|ce|bce)?$/i
  const cleanLabelPart = (part: string) => part
    .trim()
    .replace(/,\s*(?:c\.?\s*)?(?:ad|bc|ce|bce)?\s*[\d\s./-]+.*$/i, '')
    .replace(/^(?:ae|ar|av|au|bi|billon|silver|gold|bronze)\s+/i, '')
    .trim()

  const lines = labelText
    .split(/\r?\n/)
    .map(line => line.trim())
    .filter(line => line.length >= 5 && line.length <= 120 && !unreliable.test(line))

  for (const line of lines) {
    const parts = line
      .split(/\s*\/\s*/)
      .map(cleanLabelPart)
      .filter(part => part.length >= 2 && !unreliable.test(part) && !contextOnly.test(part) && !dateOnly.test(part))

    if (parts.length >= 2) {
      return parts.join(' ')
    }
  }

  return lines.find(line => line.length <= 80 && !line.includes('/'))
}

export function normalizeLookupDraft(lookup: CoinLookupResponse): CoinMutationPayload {
  const draft: CoinMutationPayload = { ...(lookup.prefilledDraft ?? {}) }
  if (!draft.notes) {
    draft.notes = draft.aiAnalysis ?? ''
  }

  applyLookupFieldSource(draft, lookup.extractedData.coinFields)
  applyParsedLookupText(draft, draft.notes)
  applyParsedLookupText(draft, draft.aiAnalysis)
  applyParsedLookupText(draft, lookup.extractedData.rawAnalysis)
  deriveNameFromParts(draft)
  setMissingLookupField(draft, 'name', lookup.extractedData.ngc?.description)
  setMissingLookupField(draft, 'name', reliableNgcLabelName(lookup.extractedData.labelText))

  return draft
}

export function normalizedEra(value: unknown): 'ancient' | 'medieval' | 'modern' | undefined {
  if (typeof value !== 'string') return undefined
  const normalized = value.trim().toLowerCase()
  if (normalized === 'ancient' || normalized === 'medieval' || normalized === 'modern') return normalized
  return undefined
}
