// coin types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { MintLocation, StorageLocation, Tag } from '@/types/collection'
import type { NumistaEvidence, NumistaLookupOutcome, SelectedNumistaReference } from '@/types/numista'
import type { CoinSet } from '@/types/sets'

export interface Coin {
  id: number
  name: string
  category: Category
  denomination: string
  ruler: string
  romanImperialFigureId: number | null
  era: string
  dateRange?: string | null
  mint: string
  mintLocationId: number | null
  mintLocation: Pick<MintLocation, 'id' | 'displayName' | 'lat' | 'lng'> | null
  material: Material
  weightGrams: number | null
  diameterMm: number | null
  grade: string
  obverseInscription: string
  reverseInscription: string
  obverseDescription: string
  reverseDescription: string
  rarityRating: string
  purchasePrice: number | null
  currentValue: number | null
  purchaseDate: string | null
  purchaseLocation: string
  vendorSku: string
  vendorInvoice: string
  storageLocationId: number | null
  storageLocation: Pick<StorageLocation, 'id' | 'name'> | null
  notes: string
  aiAnalysis: string
  obverseAnalysis: string
  reverseAnalysis: string
  referenceUrl: string
  referenceText: string
  isWishlist: boolean
  isSold: boolean
  soldPrice: number | null
  soldDate: string | null
  soldTo: string
  isPrivate: boolean
  listingStatus: string
  listingCheckedAt: string | null
  listingCheckReason: string
  sourceAlertCandidateId?: number | null
  userId: number
  images: CoinImage[]
  references?: CoinReference[]
  tags?: Tag[]
  sets?: CoinSet[]
  createdAt: string
  updatedAt: string
}

export interface CoinReference {
  id: number
  coinId: number
  catalog: string
  volume: string
  number: string
  uri: string
  createdAt: string
  updatedAt: string
}

export interface CoinReferenceInput {
  catalog: string
  volume?: string
  number: string
  uri?: string
}

export interface CatalogRegistry {
  id: number
  catalog: string
  displayName: string
  era: 'ancient' | 'medieval' | 'modern'
  volumeRequired: boolean
  createdAt?: string
  updatedAt?: string
}

export interface LegacyMigrationResult {
  succeeded: number
  skipped: number
  failed: number
  message?: string
}

export type CoinMutationPayload = Partial<Omit<Coin, 'references' | 'storageLocation' | 'mintLocation'>> & {
  references?: CoinReferenceInput[]
}

export type IntakeConfidenceLevel = 'low' | 'medium' | 'high'

export interface IntakeConfidenceSummary {
  overall: IntakeConfidenceLevel
  uncertainFields: string[]
}

export interface IntakeEvidenceItem {
  type: string
  source: string
  field: string
  value: string
  confidence: IntakeConfidenceLevel
  notes?: string
}

export interface IntakeDraft {
  draftId: number
  status: 'drafted' | 'confirmed' | 'discarded' | 'expired'
  coin: CoinMutationPayload
  confidenceSummary: IntakeConfidenceSummary
  evidence: IntakeEvidenceItem[]
  unresolvedFields: string[]
  expiresAt: string
}

export interface CoinLookupNGCData {
  certNumber: string
  normalizedCert: string
  lookupURL: string
  grade?: string
  description?: string
}

export interface CoinLookupExtractedData {
  ngc?: CoinLookupNGCData
  labelText?: string
  coinFields?: Record<string, unknown>
  confidence: IntakeConfidenceLevel
  rawAnalysis: string
}

export interface CoinLookupNumistaCandidate {
  id: string
  title: string
  issuer: string
  year: string
  thumbnail?: string
  url: string
}

export interface CoinLookupResponse {
  extractedData: CoinLookupExtractedData
  numistaCandidates: CoinLookupNumistaCandidate[]
  proposedNumistaQuery?: string
  numistaEvidence?: NumistaEvidence
  numistaLookup?: NumistaLookupOutcome | null
  prefilledDraft?: CoinMutationPayload
  candidateReferences?: CoinReferenceInput[]
}

export type CoinLookupImageRole = 'obverse' | 'reverse' | 'notes'

export interface IntakeCommitRequest {
  draftId: number
  confirm: boolean
  overrides?: CoinMutationPayload
}

export interface IntakeCommitResponse {
  draftId: number
  status: 'confirmed'
  coinId: number
}

export type QuickCaptureDraftStatus = 'active' | 'promoting' | 'promoted' | 'discarded'

export type QuickCaptureImageType = 'obverse' | 'reverse' | 'detail' | 'other'

export interface QuickCaptureDraftImage {
  id: number
  draftId: number
  filePath: string
  imageType: QuickCaptureImageType
  isPrimary: boolean
  displayOrder: number
  createdAt: string
}

export interface QuickCaptureDraft {
  id: number
  userId: number
  workingTitle: string
  dateRange: string
  era: string
  acquisitionSource: string
  purchasePrice: number | null
  notes: string
  source: string
  ngcCertNumber: string
  ngcLookupUrl: string
  ngcGrade: string
  labelText: string
  aiConfidence: string
  selectedNumistaReference?: SelectedNumistaReference | null
  status: QuickCaptureDraftStatus
  promotedCoinId: number | null
  promotedAt: string | null
  discardedAt: string | null
  images: QuickCaptureDraftImage[]
  createdAt: string
  updatedAt: string
}

export interface QuickCaptureDraftListResponse {
  drafts: QuickCaptureDraft[]
  total: number
  page: number
  limit: number
}

export interface QuickCaptureDraftInput {
  workingTitle?: string
  dateRange?: string
  era?: string
  acquisitionSource?: string
  purchasePrice?: number | null
  notes?: string
  source?: string
  ngcCertNumber?: string
  ngcLookupUrl?: string
  ngcGrade?: string
  labelText?: string
  aiConfidence?: string
  selectedNumistaId?: string
  selectedNumistaUrl?: string
  obverseImage?: File | null
  reverseImage?: File | null
  detailImages?: File[]
}

export interface QuickCaptureDraftUpdateInput {
  workingTitle: string
  dateRange: string
  era: string
  acquisitionSource: string
  purchasePrice: number | null
  notes: string
  source?: string
  ngcCertNumber?: string
  ngcLookupUrl?: string
  ngcGrade?: string
  labelText?: string
  aiConfidence?: string
  selectedNumistaId?: string
  selectedNumistaUrl?: string
  clearSelectedNumista?: boolean
  removeImageIds?: string // comma-separated IDs
  replaceObverse?: boolean
  replaceReverse?: boolean
  obverseImage?: File | null
  reverseImage?: File | null
  detailImages?: File[]
}

export interface QuickCapturePromoteOverrides {
  name?: string
  category?: string
  material?: string
  era?: string
  purchasePrice?: number | null
  purchaseLocation?: string
  notes?: string
}

export interface QuickCapturePromoteRequest {
  confirm: boolean
  target?: 'collection' | 'wishlist'
  overrides?: QuickCapturePromoteOverrides
}

export interface QuickCapturePromotionResponse {
  draftId: number
  status: 'promoted'
  coinId: number
  alreadyPromoted: boolean
  target: 'collection' | 'wishlist'
}

export interface CoinImage {
  id: number
  coinId: number
  filePath: string
  imageType: ImageType
  isPrimary: boolean
  createdAt: string
}

export type Category = string

export type Material = 'Gold' | 'Silver' | 'Bronze' | 'Copper' | 'Electrum' | 'Other'

export type ImageType = 'obverse' | 'reverse' | 'detail' | 'other'

export type CoinEra = string

export const CATEGORIES: Category[] = ['Roman', 'Greek', 'Byzantine', 'Modern', 'Other']

export const MATERIALS: Material[] = ['Gold', 'Silver', 'Bronze', 'Copper', 'Electrum', 'Other']

export const IMAGE_TYPES: ImageType[] = ['obverse', 'reverse', 'detail', 'other']

export const COIN_ERAS: CoinEra[] = ['ancient', 'medieval', 'modern']

export const CATEGORY_COLORS: Record<string, string> = {
  Roman: '#7b2d8e',
  Greek: '#6b8e23',
  Byzantine: '#8b1a1a',
  Modern: '#4682b4',
  Other: '#888888',
}

export interface CoinListResponse {
  coins: Coin[]
  total: number
  page: number
  limit: number
}

export interface ValueSnapshot {
  id: number
  userId: number
  totalValue: number
  totalInvested: number
  coinCount: number
  recordedAt: string
}

export interface CoinJournal {
  id: number
  coinId: number
  userId: number
  entry: string
  createdAt: string
}

export interface ValueComparable {
  source: string
  price: string
  url: string
}

export interface ValueEstimate {
  estimatedValue: number
  confidence: 'high' | 'medium' | 'low'
  reasoning: string
  comparables: ValueComparable[]
}

export interface CoinValueHistory {
  id: number
  coinId: number
  userId: number
  value: number
  confidence: string
  recordedAt: string
}

// T002: Coin detail page types for #219
export interface CoinDetailSectionLink {
  id: string
  title: string
  description: string
  route: string
  icon?: string
}

export interface CoinDetailMetadataRow {
  key: string
  label: string
  value: string
  valueClass?: string
  fullWidth?: boolean
  url?: string | null
}

// Feature 355 — Wishlist Purchase Reminders
export interface PurchaseReminder {
  id: number
  coinId: number
  coinName?: string
  remindDate: string
  timezone: string
  status: 'pending' | 'notified' | 'cancelled'
  notifiedAt?: string | null
  cancelledAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface PurchaseReminderListResponse {
  reminders: PurchaseReminder[]
}
