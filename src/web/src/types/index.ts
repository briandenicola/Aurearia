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

export type ShipmentCarrier = 'usps' | 'ups' | 'fedex' | 'parcel' | 'other'
export type ShipmentStatus =
  | 'pending'
  | 'label_created'
  | 'in_transit'
  | 'out_for_delivery'
  | 'delivered'
  | 'exception'
  | 'returned'
  | 'unknown'
export type ShipmentStatusSource = 'manual' | 'carrier_api'

export interface ShipmentEvent {
  id: number
  shipmentId: number
  userId: number
  eventKey: string
  status: ShipmentStatus
  statusSource: ShipmentStatusSource
  occurredAt: string
  location: string
  description: string
  rawStatus: string
  rawPayload: string
  createdAt: string
}

export interface Shipment {
  id: number
  userId: number
  coinId: number
  carrier: ShipmentCarrier
  manualCarrierName: string
  trackingNumber: string
  currentStatus: ShipmentStatus
  currentStatusSource: ShipmentStatusSource
  notes: string
  manualOverrideEnabled: boolean
  manualOverrideStatus: ShipmentStatus | ''
  manualOverrideNote: string
  manualOverrideUpdatedAt: string | null
  lastSyncedAt: string | null
  lastSyncError: string
  estimatedDeliveryAt: string | null
  deliveredAt: string | null
  events?: ShipmentEvent[]
  createdAt: string
  updatedAt: string
}

export interface ShipmentEnvelopeResponse {
  shipment: Shipment
  trackingUrl: string
}

export interface ShipmentUpsertInput {
  carrier: ShipmentCarrier
  trackingNumber: string
  notes?: string
  manualCarrierName?: string
}

export type WishlistSearchAlertCadence = 'manual' | 'daily' | 'weekly' | 'monthly'
export type AlertRunStatus = 'queued' | 'running' | 'completed' | 'failed' | 'partial' | 'rate_limited' | 'cancelled'
export type CandidateProvenanceStatus = 'verified' | 'partial' | 'unverified'
export type AlertCandidateState = 'active' | 'dismissed' | 'converted' | 'suppressed' | 'needs_review'
export type CandidateDismissalReason = 'irrelevant' | 'duplicate' | 'price_too_high' | 'poor_provenance' | 'other'

export interface WishlistSearchAlertCriteria {
  rulerOrIssuer: string
  coinType: string
  dateFrom: number | null
  dateTo: number | null
  mint: string
  mintLocationId: number | null
  material: string
  gradeOrCondition: string
  priceMin: number | null
  priceMax: number | null
  currency: string
  dealerPreference: string
  sourceFilters: string[]
  keywords: string
  notes: string
}

export interface WishlistSearchAlert {
  id: number
  userId: number
  name: string
  rulerOrIssuer: string
  coinType: string
  dateFrom: number | null
  dateTo: number | null
  mint: string
  mintLocationId: number | null
  material: string
  gradeOrCondition: string
  priceMin: number | null
  priceMax: number | null
  currency: string
  dealerPreference: string
  sourceFilters: string[]
  keywords: string
  notes: string
  cadence: WishlistSearchAlertCadence
  isActive: boolean
  lastRunAt: string | null
  createdAt: string
  updatedAt: string
}

export interface WishlistSearchAlertInput {
  name: string
  criteria: WishlistSearchAlertCriteria
  cadence: WishlistSearchAlertCadence
  isActive: boolean
}

export interface WishlistSearchAlertListResponse {
  alerts: WishlistSearchAlert[]
  total: number
  page: number
  limit: number
}

export interface AlertRun {
  id: number
  alertId: number
  userId: number
  triggerType: 'manual' | 'scheduled'
  status: AlertRunStatus
  startedAt: string
  completedAt: string | null
  durationMs: number
  criteriaSnapshot: string
  resultCount: number
  newCount: number
  duplicateCount: number
  dismissedCount: number
  partialWarnings: string[]
  errorMessage: string
  rateLimitStatus: string
  createdAt: string
}

export interface AlertRunResult {
  runId: number
  alertId: number
  status: AlertRunStatus
  startedAt: string
  completedAt: string | null
  resultCount: number
  newCount: number
  duplicateCount: number
  dismissedCount: number
  partialWarnings: string[]
  rateLimitStatus: string
  errorMessage?: string
  candidates?: AlertCandidate[]
}

export interface AlertRunListResponse {
  runs: AlertRun[]
  total: number
  page: number
  limit: number
}

export interface CandidateProvenance {
  id: number
  candidateId: number
  field: string
  value: string
  sourceUrl: string
  observedAt: string
  confidence: 'high' | 'medium' | 'low' | string
  verificationState: CandidateProvenanceStatus
  notes: string
}

export interface AlertCandidate {
  id: number
  userId: number
  alertId: number
  runId: number
  sourceUrl: string
  canonicalSourceUrl: string
  sourceName: string
  title: string
  observedPrice: number | null
  observedCurrency: string
  reasonForMatch: string
  fields: Record<string, string>
  lastSeenAt: string
  firstSeenAt: string
  provenanceStatus: CandidateProvenanceStatus
  lifecycleState: AlertCandidateState
  duplicateOfCandidateId: number | null
  matchingWishlistCoinId: number | null
  convertedCoinId: number | null
  dismissalReason: string
  provenance?: CandidateProvenance[]
  createdAt: string
  updatedAt: string
}

export interface AlertCandidateListResponse {
  candidates: AlertCandidate[]
  total: number
  page: number
  limit: number
}

export interface DismissWishlistSearchAlertCandidateInput {
  reason: CandidateDismissalReason | ''
  notes?: string
}

export interface ConvertWishlistSearchAlertCandidateInput {
  coin: CoinMutationPayload
  acknowledgeDuplicateWarning: boolean
}

export interface ConvertWishlistSearchAlertCandidateResponse {
  coin?: Coin
  candidate: AlertCandidate
  warnings: string[]
}

export interface AdjustWishlistSearchAlertCriteriaInput {
  candidateIds: number[]
  criteria: WishlistSearchAlertCriteria
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

export interface UserNote {
  id: number
  userId: number
  title: string
  body: string
  createdAt: string
  updatedAt: string
}

export interface NoteInput {
  title: string
  body: string
}

export interface NoteListResponse {
  notes: UserNote[]
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

export interface Tag {
  id: number
  userId: number
  name: string
  color: string
}

export interface StorageLocation {
  id: number
  userId?: number
  name: string
  sortOrder?: number
}

export interface MintLocation {
  id: number
  userId?: number | null
  displayName: string
  lat: number
  lng: number
  region: string
  aliases: string[]
  createdAt: string
  updatedAt: string
  nomismaUri?: string | null
  nomismaLabel?: string
  nomismaLinkedAt?: string | null
}

export interface GeocodeCandidate {
  displayName: string
  lat: number
  lng: number
}

// Nomisma.org authority linking (global mint locations only; admin-only)
export type NomismaSearchStatus = 'ok' | 'no_match' | 'unavailable'

export interface NomismaCandidate {
  uri: string
  label: string
  score: number
  match: boolean
}

export interface NomismaSearchResponse {
  status: NomismaSearchStatus
  candidates: NomismaCandidate[]
}

export interface CollectionSetOption {
  id: number
  name: string
  color: string
  filterValue: string
  source: 'tag' | 'set'
}

export type CoinSetType = 'standard' | 'goal' | 'smart' | 'agentic'
export type LegacyCoinSetType = 'open' | 'defined' | 'tracker' | 'dynamic'
export type CoinSetTypeResponse = CoinSetType | LegacyCoinSetType

export function normalizeCoinSetType(setType: CoinSetTypeResponse): CoinSetType {
  if (setType === 'open') return 'standard'
  if (setType === 'defined') return 'goal'
  if (setType === 'tracker' || setType === 'dynamic') return 'agentic'
  return setType
}

export interface CoinSet {
  id: number
  userId: number
  name: string
  description?: string
  color: string
  icon?: string
  setType: CoinSetTypeResponse
  parentSetId?: number | null
  targetCompletionDate?: string | null
  createdAt: string
  updatedAt: string
}

export interface CoinSetSummary {
  id: number
  name: string
  color: string
  icon?: string
  setType: CoinSetTypeResponse
  coinCount: number
  totalValue: number
  completionPercentage?: number | null
  valueChangePercent?: number | null
  agenticStatus?: string | null
}

export interface CoinSetDetail extends CoinSetSummary {
  description?: string
  parentSetId?: number | null
  targetCompletionDate?: string | null
  totalInvested: number
  avgValuePerCoin?: number | null
  highestValueCoinId?: number | null
  agenticPrompt?: string | null
  agenticStatus?: string | null
}

export type CoinRecommendationTargetType = 'set' | 'tag'
export type CoinRecommendationStatus = 'pending' | 'accepted' | 'rejected' | 'dismissed'

export interface CoinRecommendation {
  id: number
  targetType: CoinRecommendationTargetType
  targetId: number
  targetName: string
  score: number
  confidence: 'high' | 'medium' | 'low'
  reasons: string[]
  status: CoinRecommendationStatus
}

export interface CreateCoinSetRequest {
  name: string
  description?: string
  color?: string
  icon?: string
  setType: CoinSetType
  parentSetId?: number | null
  targetCompletionDate?: string | null
  smartCriteria?: Record<string, unknown> | null
  templateId?: string | null
  agenticPrompt?: string | null
}

export interface CreateCoinSetFromCsvRequest extends CreateCoinSetRequest {
  csv: string
}

export interface SetBuilderRun {
  id: number
  userId: number
  prompt: string
  status: 'queued' | 'running' | 'completed' | 'failed'
  provider?: string
  model?: string
  transcriptSummary?: string
  errorMessage?: string
  terminationReason?: string
  usedTurns?: number
  createdAt: string
  updatedAt: string
}

export interface CreateSetBuilderRunRequest {
  prompt: string
}

export interface SetProposalSlot {
  id: number
  proposalId: number
  label: string
  criteria?: Record<string, unknown> | null
  group?: string
  sortOrder: number
  verificationStatus: 'verified' | 'unverified'
  sourceNote?: string
  validationNote?: string
  createdAt: string
  updatedAt: string
}

export interface UpdateSetProposalSlotRequest {
  label: string
  criteria?: Record<string, unknown> | null
  group?: string
  sortOrder: number
  verificationStatus: 'verified' | 'unverified'
  sourceNote?: string
  validationNote?: string
}

export interface UpdateSetProposalRequest {
  proposedName: string
  description?: string
  color?: string
  selectedScope?: string
  slots: UpdateSetProposalSlotRequest[]
}

export interface RegenerateSetProposalRequest {
  feedback: string
}

export interface SetProposalPrematchSummary {
  estimatedFilled?: number
  estimatedTotal?: number
  notes?: string
}

export interface SetProposal {
  id: number
  userId: number
  builderRunId: number
  run?: SetBuilderRun
  originalPrompt: string
  status: 'pending' | 'approved' | 'rejected' | 'expired' | 'creation_failed'
  proposedName: string
  proposedSlug?: string
  description?: string
  color: string
  selectedScope?: string
  scopeOptions?: {
    scopeSummary?: string
    groupBy?: string
    options?: Array<{
      label: string
      description?: string
      estimated_slot_count?: number
      recommended?: boolean
    }>
  } | null
  rosterPayload?: {
    transcriptSummary?: string
    turnsUsed?: number
  } | null
  preMatchSummary?: SetProposalPrematchSummary | null
  expiresAt: string
  rejectedAt?: string | null
  rejectionReason?: string
  approvalSetId?: number | null
  errorMessage?: string
  slots?: SetProposalSlot[]
  createdAt: string
  updatedAt: string
}

export type UpdateCoinSetRequest = Partial<CreateCoinSetRequest>

export interface AddCoinToSetRequest {
  coinId: number
  notes?: string
  targetId?: number
}

export interface ReorderSetCoinsRequest {
  coinIds: number[]
}

// US2: Defined/Goal Sets and Completion
export interface CoinSetTarget {
  id: number
  setId: number
  label: string
  year?: number | null
  mintMark?: string | null
  denomination?: string | null
  country?: string | null
  material?: string | null
  matchRules?: Record<string, unknown> | null
  sortOrder: number
  createdAt?: string
}

export interface CoinSetCompletion {
  totalTargets: number
  completedTargets: number
  completionPercentage: number
  missingTargets: CoinSetTarget[]
  targets?: CoinSetTarget[]
  targetMatches?: Array<{
    target: CoinSetTarget
    coin?: Coin | null
  }>
  collectionItems?: number
  wishlistItems?: number
}

export interface CoinSetTemplate {
  id: string
  name: string
  category: string
  description: string
  version: number
  targets?: CoinSetTemplateTarget[]
}

export interface CoinSetTemplateTarget {
  label: string
  year?: number | null
  mintMark?: string | null
  denomination?: string | null
  country?: string | null
  material?: string | null
  sortOrder: number
}

// US3: Snapshots and Trends
export interface CoinSetSnapshot {
  id?: number
  setId?: number
  userId?: number
  snapshotDate: string
  totalValue: number
  totalInvested: number
  coinCount: number
  completionPercentage?: number | null
  avgValuePerCoin?: number | null
  highestValueCoinId?: number | null
}

export interface CoinSetAnalytics {
  roiPercent?: number | null
  bestPerformerCoinId?: number | null
  worstPerformerCoinId?: number | null
  acquisitionRatePerMonth?: number | null
  projectedCompletionDate?: string | null
}

export interface CoinSetComparison {
  setId: number
  name: string
  startValue: number
  endValue: number
  valueChange: number
  valueChangePercent: number
  completionChange?: number | null
}

export type SmartCriteriaOperator = 'and' | 'or'
export type SmartCriteriaRuleOp = 'eq' | 'neq' | 'contains' | 'startsWith' | 'in' | 'between' | 'gte' | 'lte' | 'isNull' | 'isNotNull'

export interface SmartCriteriaRule {
  field: string
  op: SmartCriteriaRuleOp
  value?: unknown
}

export interface SmartCriteriaGroup {
  operator: SmartCriteriaOperator
  rules: Array<SmartCriteriaRule | SmartCriteriaGroup>
}

export interface SmartSetPreview {
  coinIds: number[]
  coinCount: number
  totalValue: number
}

export interface SmartCriteriaTemplate {
  id: number
  userId: number
  name: string
  description: string
  criteria: SmartCriteriaGroup
  createdAt: string
  updatedAt: string
}

export interface SuggestedSmartCriteria {
  id: string
  name: string
  description: string
  criteria: SmartCriteriaGroup
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

// F028: Roman Emperor collection tracker
export type ImperialFigureRole = 'emperor' | 'empress' | 'caesar' | 'usurper' | 'other'
export type ImperialFigureRegion = 'west' | 'east'
export type RarityTier = 'common' | 'scarce' | 'rare' | 'very_rare'

export interface RomanImperialFigure {
  id: number
  name: string
  aliases: string[]
  role: ImperialFigureRole
  region: ImperialFigureRegion
  dynasty: string
  reignStart: number
  reignEnd: number
  sortOrder: number
  rarityTier: RarityTier
  notes?: string
}

export interface ImperialFigureSlot {
  figure: RomanImperialFigure
  coin: Coin | null
  coins: Coin[]
  highlightedCoinId: number | null
}

export interface DynastyProgress {
  dynasty: string
  owned: number
  total: number
  figures: ImperialFigureSlot[]
}

export interface CategoryProgress {
  roles: ImperialFigureRole[]
  owned: number
  total: number
  percentage: number
  dynasties: DynastyProgress[]
}

export interface EmperorTrackerResult {
  emperor: CategoryProgress
  suggestions: RomanImperialFigure[]
  usurpers?: CategoryProgress
  empresses?: CategoryProgress
  other?: CategoryProgress
}

export interface User {
  id: number
  username: string
  role: 'admin' | 'user'
  email: string
  avatarPath: string
  isPublic: boolean
  bio: string
  zipCode: string
  numisBidsUsername?: string
  numisBidsConfigured?: boolean
  cngUsername?: string
  cngConfigured?: boolean
  parcelAppConfigured?: boolean
  pushoverEnabled?: boolean
  coinOfDayEnabled?: boolean
  emperorTrackerEnabled?: boolean
  emperorTrackerShowUsurpers?: boolean
  emperorTrackerShowEmpresses?: boolean
  emperorTrackerShowOtherFigures?: boolean
}

export interface AuthResponse {
  token: string
  refreshToken: string
  user: User
}

export type OIDCProviderType = 'entra' | 'pocket_id' | 'generic'
export type OIDCTestStatus = 'unknown' | 'ok' | 'failed'

export interface OIDCPublicProvider {
  id: number
  name: string
  displayName: string
  providerType: OIDCProviderType
}

export interface OIDCPublicProvidersResponse {
  providers: OIDCPublicProvider[]
}

export interface OIDCStartFlowRequest {
  redirectPath: string
  callbackPath?: string
}

export interface OIDCStartFlowResponse {
  authorizationUrl: string
  expiresAt: string
}

export interface OIDCLinkedIdentity {
  id: number
  providerId: number
  providerDisplayName: string
  issuer: string
  subjectPreview: string
  email: string
  emailVerified: boolean
  createdAt: string
  lastLoginAt?: string | null
}

export interface OIDCLinkedIdentitiesResponse {
  identities: OIDCLinkedIdentity[]
}

export interface OIDCLinkCallbackResponse {
  message: string
  identity: OIDCLinkedIdentity
}

export interface OIDCMessageResponse {
  message: string
}

export interface OIDCAdminProvider {
  id: number
  name: string
  displayName: string
  providerType: OIDCProviderType
  enabled: boolean
  issuerUrl: string
  clientId: string
  clientSecretConfigured: boolean
  scopes: string[]
  callbackPath: string
  requireVerifiedEmail?: boolean
  lastTestedAt?: string | null
  lastTestStatus: OIDCTestStatus
  lastTestMessage: string
  createdAt?: string
  updatedAt?: string
}

export interface OIDCAdminProvidersResponse {
  providers: OIDCAdminProvider[]
}

export interface OIDCAdminProviderInput {
  name: string
  displayName: string
  providerType: OIDCProviderType
  enabled: boolean
  issuerUrl: string
  clientId: string
  clientSecret?: string
  scopes: string[]
  callbackPath?: string
  requireVerifiedEmail?: boolean
}

export type OIDCAdminProviderUpdate = Partial<OIDCAdminProviderInput>

export interface OIDCProviderTestResponse {
  available: boolean
  message: string
  issuer: string
  authorizationEndpoint: string
  tokenEndpoint: string
}

export interface WebAuthnCredentialInfo {
  id: number
  credentialId: string
  name: string
  createdAt: string
}

export interface CoinListResponse {
  coins: Coin[]
  total: number
  page: number
  limit: number
}

export interface StatsResponse {
  totalCoins: number
  totalWishlist: number
  byCategory: { category: string; count: number }[]
  byMaterial: { material: string; count: number }[]
  byGrade: { grade: string; count: number }[]
  byEra: { era: string; count: number }[]
  byRuler: { ruler: string; count: number }[]
  byPriceRange: { range: string; count: number }[]
  values: {
    totalPurchasePrice: number
    totalCurrentValue: number
    avgPurchasePrice: number
    avgCurrentValue: number
  }
}

export type InvestmentBreakdownDimension = 'purchase-year' | 'material'

export interface InvestmentBreakdownSegment {
  label: string
  year: number | null
  month: number | null
  invested: number
  currentValue: number
  gainLoss: number
  gainLossPct: number | null
  coinCount: number
  missingCurrentValueCount: number
  missingPurchasePriceCount: number
}

export interface InvestmentMovementCoin {
  coinId: number
  name: string
  initialValue: number
  currentValue: number
  changeAmount: number
  changePct: number
  changeExplanation?: string | null
}

export interface StaleValuationCoin {
  coinId: number
  name: string
  lastValuationAt: string | null
}

export type InvestmentBreakdownResponse =
  | InvestmentBreakdownSegment[]
  | {
      dimension?: InvestmentBreakdownDimension
      segments: InvestmentBreakdownSegment[]
      topIncreases?: InvestmentMovementCoin[]
      topDrops?: InvestmentMovementCoin[]
      staleValuations?: StaleValuationCoin[]
    }

export type HealthGrade = 'A' | 'B' | 'C' | 'D' | 'F'
export type HealthTrendDirection = 'up' | 'flat' | 'down' | 'unavailable'
export type HealthChecklistDimension = 'metadata' | 'images' | 'valuation' | 'ai'
export type HealthChecklistSeverity = 'high' | 'medium' | 'low'
export type HealthQuickAction = 'edit_metadata' | 'upload_images' | 'run_valuation' | 'run_ai_analysis'

export interface HealthWeights {
  metadata: number
  imageCoverage: number
  valuationFreshness: number
  aiCoverage: number
}

export interface HealthDimensions {
  metadata: number
  imageCoverage: number
  valuationFreshness: number
  aiCoverage: number
}

export interface CollectionHealthTrend {
  status: 'available' | 'unavailable'
  delta: number | null
  direction: HealthTrendDirection
}

export interface CollectionHealthSummary {
  score: number
  grade: HealthGrade
  eligibleCoinCount: number
  weights: HealthWeights
  dimensions: HealthDimensions
  trend30d: CollectionHealthTrend
}

export interface MissingChecklistItem {
  key: string
  dimension: HealthChecklistDimension
  label: string
  severity: HealthChecklistSeverity
  actionHint: HealthQuickAction
}

export interface CoinHealthItem {
  coinId: number
  title: string
  score: number
  grade: HealthGrade
  dimensions: HealthDimensions
  missingItems: MissingChecklistItem[]
  quickActions: HealthQuickAction[]
}

export interface CoinHealthListResponse {
  coins: CoinHealthItem[]
  pagination: {
    page: number
    limit: number
    total: number
  }
}

export interface MissingFieldStat {
  key: string
  count: number
  percentage: number
}

export interface AdminHealthSummaryResponse {
  medianScore: number
  lowScorePercentage: number
  lowScoreThreshold: number
  eligibleCoinCount: number
  topMissingFields: MissingFieldStat[]
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

export interface NumistaType {
  id: number
  title: string
  category: string
  issuer?: { name: string }
  min_year?: number
  max_year?: number
  obverse_thumbnail?: string
  reverse_thumbnail?: string
}

export interface NumistaSearchResponse {
  count: number
  types: NumistaType[]
}

export type NumistaLookupPath = 'direct' | 'photo'
export type NumistaLookupStatus = 'success' | 'empty' | 'unconfigured' | 'quota-limited' | 'timeout' | 'unavailable'
export type NumistaQuerySource = 'generated' | 'user-edited' | 'manual'
export type NumistaSearchAttempt = 'primary' | 'relaxed'
export type NumistaQueryGenerationVersion = 'numista-query-v2'
export type NumistaEnrichmentState = 'not_requested' | 'enriched' | 'cached' | 'failed'
export type NumistaRelevanceField = 'exact_id' | 'title' | 'issuer' | 'denomination' | 'mint' | 'date' | 'material' | 'inscription'
export type NumistaRelevanceKind = 'match' | 'conflict' | 'unavailable'
export type NumistaRelevanceBand = 'strong' | 'possible' | 'weak'

export interface NumistaEvidence {
  title?: string
  issuer?: string
  denomination?: string
  mint?: string
  dateText?: string
  material?: string
  obverseInscription?: string
  reverseInscription?: string
  reverseType?: string
  visibleText?: string
  exactNumistaId?: number
}

export interface NumistaQueryProposalRequest {
  path: NumistaLookupPath
  evidence: NumistaEvidence
}

export interface NumistaQueryProposal {
  query: string
  querySource: 'generated'
  generationVersion: NumistaQueryGenerationVersion
}

export interface NumistaLookupRequest {
  query: string
  path: NumistaLookupPath
  evidence: NumistaEvidence
  querySource: NumistaQuerySource
  generationVersion?: NumistaQueryGenerationVersion
}

export interface NumistaEnrichmentRequest extends NumistaLookupRequest {
  candidates: NumistaCandidate[]
}

export interface NumistaRelevanceReason {
  field: NumistaRelevanceField
  kind: NumistaRelevanceKind
  code: string
  label: string
}

export interface NumistaRelevanceAssessment {
  scoringVersion: 'numista-v1'
  score: number
  band: NumistaRelevanceBand
  reasons: NumistaRelevanceReason[]
}

export interface NumistaCandidate {
  id: number
  canonicalUrl: string
  title: string
  issuer?: string
  denomination?: string
  mint?: string
  minYear?: number
  maxYear?: number
  yearDisplay?: string
  material?: string
  obverseInscription?: string
  reverseInscription?: string
  obverseThumbnail?: string
  reverseThumbnail?: string
  providerPosition: number
  enrichmentState: NumistaEnrichmentState
  assessment: NumistaRelevanceAssessment
}

export interface NumistaCacheMetadata {
  hit: boolean
  coalesced: boolean
  createdAt: string
  expiresAt: string
  ageSeconds: number
}

export interface NumistaLookupOutcome {
  status: NumistaLookupStatus
  effectiveQuery: string
  candidates: NumistaCandidate[]
  guidanceCode?: string
  retryAfterSeconds?: number
  cache?: NumistaCacheMetadata
  stage: 'broad' | 'enriched'
  querySource: NumistaQuerySource
  searchAttempt: NumistaSearchAttempt
  searchAttemptCount: 1 | 2
}

export interface NumistaSettings {
  NumistaSearchTTLHours: string
  NumistaDetailTTLHours: string
  NumistaEnrichmentLimit: string
  NumistaSearchResultLimit: string
  NumistaSearchTimeoutSeconds: string
  NumistaDetailTimeoutSeconds: string
}

export type NumistaStatusCounts = Partial<Record<NumistaLookupStatus, number>>

export interface NumistaHealthSummary {
  configured: boolean
  configurationValid: boolean
  lastOutcome?: NumistaLookupStatus | null
  lastCheckedAt?: string | null
  statusCounts: NumistaStatusCounts
  broadRequestCount: number
  detailRequestCount: number
  freshCacheHitCount: number
  coalescedRequestCount: number
  providerLoadCount: number
  providerFailureCount: number
  cancelledRequestCount: number
  freshCacheHitRate: number
  p50ElapsedMs: number
  p95ElapsedMs: number
  enrichmentAttempted: number
  enrichmentSucceeded: number
  enrichmentFailed: number
  lastQuotaLimitedAt?: string | null
  lastRetryAfterSeconds?: number | null
}

export interface SelectedNumistaReference {
  catalog: 'Numista'
  number: string
  uri: string
}

export interface UserInfo {
  id: number
  username: string
  role: 'admin' | 'user'
  email: string
  avatarPath: string
  isPublic: boolean
  bio: string
  zipCode: string
  emailMissing: boolean
  numisBidsUsername: string
  numisBidsConfigured: boolean
  cngUsername?: string
  cngConfigured?: boolean
  parcelAppConfigured?: boolean
  pushoverEnabled?: boolean
  coinOfDayEnabled?: boolean
  emperorTrackerEnabled?: boolean
  emperorTrackerShowUsurpers?: boolean
  emperorTrackerShowEmpresses?: boolean
  emperorTrackerShowOtherFigures?: boolean
  lockedUntil?: string | null
  failedLoginAttempts?: number
  createdAt: string
}

export interface AppSettings extends Partial<NumistaSettings> {
  AIProvider: string
  OllamaURL: string
  OllamaModel: string
  ObversePrompt: string
  ReversePrompt: string
  TextExtractionPrompt: string
  OllamaTimeout: string
  SearXNGURL: string
  LogLevel: string
  PublicAppURL?: string
  RegistrationMode?: string
  AuctionAlertsCheckEnabled?: string
  AuctionAlertsCheckStartTime?: string
  AuctionAlertsCheckInterval?: string
  WishlistSearchAlertsCheckEnabled?: string
  WishlistSearchAlertsCheckStartTime?: string
  CoinCategories?: string
  CoinEras?: string
  [key: string]: string | undefined
}

export interface SecuritySummary {
  failedLogins: number
  lockedAccounts: number
  activeBans: number
  recentEvents: number
  loginFailures?: number
  activeIpRuleCount?: number
}

export interface SecurityEvent {
  id: number | string
  timestamp: string
  type: string
  severity: string
  username?: string | null
  ip?: string | null
  clientIp?: string | null
  outcome?: string | null
  message?: string | null
  userAgent?: string | null
  createdAt?: string
}

export interface SecurityEventFilters {
  type?: string
  severity?: string
  username?: string
  ip?: string
  clientIp?: string
  outcome?: string
  since?: string
  limit?: number
}

export interface SecurityEventsResponse {
  events: SecurityEvent[]
  total?: number
}

export interface SecurityIpRule {
  id: number
  cidr: string
  reason: string
  expiresAt?: string | null
  createdBy?: string | number | null
  createdAt?: string
}

export interface CreateSecurityIpRuleRequest {
  cidr: string
  duration?: string
  durationMinutes?: number
  expiresAt?: string
  reason: string
}

export interface SecurityExposureCheck {
  publicIp?: string
  proxy?: boolean
  proxyWarning?: string
  cors?: boolean
  corsWarning?: string
  webAuthn?: boolean
  webAuthnWarning?: string
  publicAppUrl?: boolean
  publicAppURL?: boolean
  publicAppUrlWarning?: string
  registration?: boolean
  registrationWarning?: string
  agentToken?: boolean
  agentTokenWarning?: string
  warnings?: string[]
  checks?: Record<string, boolean | string | null | undefined>
  config?: {
    publicAppURL?: string
    webauthnOrigin?: string
    trustedProxiesConfigured?: boolean
    agentInternalTokenSet?: boolean
    registrationMode?: string
    backupStatus?: string
  }
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

export type AIJobStatus = string
export type AIJobType = 'coin_analysis' | 'coin_grading' | 'coin_value_estimate' | 'value_estimate' | 'valuation' | (string & {})

export interface CoinGradingResult {
  gradingReport: string
}

export interface AIJob {
  id: string
  userId?: number
  coinId: number
  jobType: AIJobType
  side?: 'obverse' | 'reverse' | null
  status: AIJobStatus
  result?: unknown
  errorMessage?: string | null
  createdAt: string
  updatedAt: string
  startedAt?: string | null
  completedAt?: string | null
}

export interface AIJobStartResponse {
  id?: string | number
  jobId?: string | number
  job?: AIJob
  status: AIJobStatus
  jobType: string
  coinId: number
  side?: 'obverse' | 'reverse' | null
  result?: unknown
  errorMessage?: string | null
  createdAt?: string
  updatedAt?: string
  startedAt?: string | null
  completedAt?: string | null
}

export interface CoinValueHistory {
  id: number
  coinId: number
  userId: number
  value: number
  confidence: string
  recordedAt: string
}

export interface CoinShow {
  name: string
  dates: string
  location: string
  venue: string
  url: string
  description: string
  entryFee: string
  notableDealers: string[]
}

export interface PortfolioSummary {
  totalCoins: number
  totalValue: number
  totalInvested: number
  categories: { category: string; count: number }[]
  materials: { material: string; count: number }[]
  eras: { era: string; count: number }[]
  rulers: { ruler: string; count: number }[]
  topCoins: { name: string; category: string; currentValue: number | null; ruler: string; era: string }[]
  missingFields?: Record<string, number>
}

export type Theme = 'dark' | 'light' | 'british-museum' | 'louvre' | 'capitoline' | 'byzantine' | 'modern-greek'

export const LOG_LEVELS = ['trace', 'debug', 'info', 'warn', 'error'] as const

export interface LogEntry {
  timestamp: string
  level: string
  message: string
}

export interface ApiKey {
  id: number
  userId: number
  keyPrefix: string
  name: string
  capabilities: string // "read" or "read,write"
  createdAt: string
  lastUsedAt: string | null
  revokedAt: string | null
}

export interface AgentChatMessage {
  role: 'user' | 'assistant'
  content: string
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

export interface AgentChatAppContext {
  route?: string
  activeCoinId?: number
}

export interface CollectionCoinSummary {
  id: number
  name: string
  category?: string
  era?: string
  ruler?: string
  material?: string
  currentValue?: number | null
}

export interface CollectionAggregateSummary {
  totalCoins: number
  totalWishlist: number
  totalSold: number
  totalCurrentUsd: number
  totalPurchaseUsd: number
}

export interface CollectionReadResult {
  resultType: string
  total?: number
  coins?: CollectionCoinSummary[]
  aggregate?: CollectionAggregateSummary
}

export interface CollectionDisambiguation {
  message: string
  candidates: CollectionCoinSummary[]
}

export interface CollectionProposalPreview {
  proposalId: string
  proposalToken: string
  coinId: number
  coinName: string
  changedFields: string[]
  changes: Record<string, unknown>
  expiresAt: string
}

export interface CollectionChatResponse {
  kind: 'read_result' | 'proposal' | 'disambiguation' | 'validation_error'
  message: string
  readResult?: CollectionReadResult
  disambiguation?: CollectionDisambiguation
  proposal?: CollectionProposalPreview
  errorCode?: string
}

export interface CoinSuggestion {
  name: string
  description: string
  category: string
  era: string
  ruler: string
  material: string
  denomination: string
  estPrice: string
  imageUrl: string
  sourceUrl: string
  sourceName: string
  candidateReferences?: CoinReferenceInput[]
}

export interface AgentChatResponse {
  message: string
  suggestions: CoinSuggestion[]
  collection?: CollectionChatResponse
}

export interface FollowUser {
  id: number
  username: string
  avatarPath: string
  isPublic: boolean
  bio: string
  isFollowing: boolean
  followStatus: string // '', 'pending', 'accepted', 'blocked'
  coinCount: number
  status?: string // used in followers list: 'pending' | 'accepted'
}

export interface PublicProfile extends FollowUser {
  followerCount: number
  followingCount: number
}

export interface CoinComment {
  id: number
  coinId: number
  userId: number
  username: string
  avatarPath: string
  comment: string
  rating: number
  createdAt: string
}

export interface CoinRating {
  average: number
  count: number
  userRating: number
}

export interface LimitedCoin {
  id: number
  name: string
  category: Category
  denomination: string
  ruler: string
  era: string
  material: Material
  grade: string
  images: CoinImage[]
}

export type AuctionLotStatus = 'watching' | 'bidding' | 'won' | 'lost' | 'passed'
export type AuctionSource = 'numisbids' | 'cng'
export type AuctionLotStatusSource = 'sync' | 'manual'

export interface AuctionLot {
  id: number
  numisBidsUrl: string
  source: AuctionSource
  sourceUrl: string
  sourceLotId?: string
  sourceSaleId?: string
  saleId: string
  lotNumber: number
  auctionHouse: string
  saleName: string
  saleDate: string | null
  auctionEndTime: string | null
  title: string
  description: string
  notes: string
  category: Category
  estimate: number | null
  initialBid: number | null
  currentBid: number | null
  maxBid: number | null
  winningBid: number | null
  currency: string
  status: AuctionLotStatus
  statusSource?: AuctionLotStatusSource
  imageUrl: string
  coinId: number | null
  coin?: Coin
  eventId: number | null
  userId: number
  createdAt: string
  updatedAt: string
}

export interface AuctionLotListResponse {
  lots: AuctionLot[]
  total: number
}

export type PriceAlertDirection = 'above' | 'below'

export interface PriceAlert {
  id: number
  auctionLotId: number
  lotTitle?: string
  targetPrice: number
  direction: PriceAlertDirection
  isTriggered: boolean
  triggeredAt: string | null
  createdAt: string
}

export interface BidReminder {
  id: number
  auctionLotId: number
  lotTitle?: string
  minutesBefore: number
  isNotified: boolean
  notifiedAt: string | null
  createdAt: string
}

export type BidRecommendationConfidence = 'insufficient_data' | 'low' | 'medium' | 'high'

export interface BidRecommendation {
  suggestedMaxBid: number | null
  confidence: BidRecommendationConfidence
  sampleSize: number
  rationale: string
}

export type MarketSignalStatus = 'unavailable' | 'ok'
export type MarketTrendDirection = 'rising' | 'stable' | 'declining' | 'unknown'

export interface MarketSignal {
  status: MarketSignalStatus
  trendDirection?: MarketTrendDirection
  priceLow?: number
  priceHigh?: number
  currency?: string
  sampleSize?: number
  rationale: string
  sources?: string[]
}

// Deep Agentic Coin Identification (344-deep-agentic-coin-identification).
// Contract anchor: specs/344-deep-agentic-coin-identification/contracts/deep-identification.openapi.yaml
export type DeepProviderId = 'nomisma' | 'numista' | 'ngc' | 'ocre' | 'rpc'
export type DeepJobStatus = 'queued' | 'running' | 'partial' | 'completed' | 'failed' | 'cancelled'
export type DeepJobSource = 'intake' | 'saved_coin'
export type DeepProviderStatus =
  | 'pending'
  | 'running'
  | 'contributed'
  | 'no_match'
  | 'failed'
  | 'timed_out'
  | 'skipped'
  | 'not_automated'
  | 'unavailable'

export interface DeepJob {
  id: number
  coinId?: number | null
  source: DeepJobSource
  status: DeepJobStatus
  partialSuccess: boolean
  selectedProviders?: DeepProviderId[]
  requestedProviders?: DeepProviderId[]
  retryOfJobId?: number | null
  cancelRequested: boolean
  lastSeq: number
  eventsAvailable: boolean
  failureCode?: string
  failureMessage?: string
  appliedCoinId?: number | null
  appliedDraftId?: number | null
  appliedAt?: string | null
  startedAt?: string | null
  completedAt?: string | null
  expiresAt: string
  createdAt: string
}

export interface DeepReportCoverage {
  provider: DeepProviderId
  status: DeepProviderStatus
  note?: string
  linkOut?: string | null
}

export interface DeepClaim {
  field: string
  value: string
  confidence?: number
  citation: string
  excerpt?: string
}

export interface DeepReportDisagreement {
  field: string
  claims: DeepClaim[]
  resolution: 'unresolved' | 'preferred'
}

export interface DeepReportAttribution {
  provider: DeepProviderId
  text: string
  identifier?: string | null
}

export interface DeepReport {
  schemaVersion: number
  narrative: string
  coverage: DeepReportCoverage[]
  disagreements?: DeepReportDisagreement[]
  unresolvedQuestions?: string[]
  attributions?: DeepReportAttribution[]
  partialSuccess: boolean
  generatedAt: string
}

export interface DeepProposalFieldEntry {
  proposed: unknown
  confidence?: number
  evidence?: DeepClaim[]
  ownerEdited: boolean
  ownerValue: unknown
  accepted: boolean | null
}

export interface DeepProposal {
  schemaVersion: number
  targetCoinId?: number | null
  fields: Record<string, DeepProposalFieldEntry>
  sourceReportGeneratedAt?: string
}

export interface DeepJobEnvelope {
  job: DeepJob
  reused?: boolean
  report?: DeepReport | null
  proposal?: DeepProposal | null
}

export interface DeepJobListResponse {
  jobs: DeepJob[]
  nextCursor?: string
}

export interface CreateDeepIdentificationJobInput {
  coinId?: number
  obverseImage?: File | null
  reverseImage?: File | null
  hintImages?: File[]
  notes?: string
  providers?: DeepProviderId[]
}

export interface DeepProposalFieldEdit {
  ownerValue?: unknown
  accepted?: boolean | null
}

export interface UpdateDeepIdentificationProposalInput {
  fields: Record<string, DeepProposalFieldEdit>
}

export type DeepApplyTarget = 'draft' | 'coin'

export interface ApplyDeepIdentificationProposalInput {
  target: DeepApplyTarget
  fields?: string[]
}

export interface DeepApplyResult {
  jobId: number
  draftId?: number | null
  coinId?: number | null
  appliedFields: string[]
  appliedAt: string
}

export interface ListDeepIdentificationJobsParams {
  coinId?: number
  activeOnly?: boolean
  status?: DeepJobStatus
  limit?: number
  cursor?: string
}

// SSE envelope from GET /api/deep-identification/jobs/{id}/events
// (contracts/sse-events.md §1/§2). Not a passthrough of the internal
// LangGraph stream - a persisted, replayable, application-owned shape.
export type DeepStreamEventType =
  | 'job_accepted'
  | 'status_changed'
  | 'router_selected'
  | 'provider_started'
  | 'provider_result'
  | 'evaluation'
  | 'synthesis_started'
  | 'progress'
  | 'terminal'

export interface DeepStreamEvent {
  seq: number
  jobId: number
  type: DeepStreamEventType | string
  ts: string
  payload: Record<string, unknown>
}

export interface DeepStreamTruncatedPayload {
  status: DeepJobStatus
  earliestSeq: number
  lastSeq: number
}

export interface DeepStreamTerminalPayload {
  status: DeepJobStatus
  partialSuccess: boolean
  failureCode?: string
  hasReport: boolean
  hasProposal: boolean
}

export interface CalendarEventDetail {
  id: number
  title: string
  auctionHouse: string
  startDate: string | null
  endDate: string | null
  url: string
  notes: string
  createdAt: string
  updatedAt: string
}

export interface AvailabilityRunSummary {
  runId: number
  coinsChecked: number
  available: number
  unavailable: number
  unknown: number
  durationMs: number
}

export interface AvailabilityResult {
  id: number
  runId: number
  coinId: number
  coinName: string
  url: string
  status: string
  reason: string
  httpStatus: number | null
  agentUsed: boolean
  checkedAt: string
}

export interface AvailabilityRun {
  id: number
  userId: number
  userName?: string
  triggerType: string
  triggerUserId: number | null
  status: string
  failMessage?: string
  coinsChecked: number
  available: number
  unavailable: number
  unknown: number
  errors: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  results?: AvailabilityResult[]
  createdAt: string
}

export interface ValuationResult {
  id: number
  runId: number
  coinId: number
  coinName: string
  previousValue: number | null
  estimatedValue: number
  confidence: string
  reasoning: string
  changeExplanation?: string | null
  status: string
  errorMessage?: string
  checkedAt: string
}

export interface ValuationRun {
  id: number
  userId: number
  triggerType: string
  triggerUserId: number | null
  status: string
  totalCoins: number
  coinsChecked: number
  coinsUpdated: number
  coinsSkipped: number
  errors: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage?: string
  results?: ValuationResult[]
  createdAt: string
}

export interface AuctionEndingRun {
  id: number
  triggerType: 'scheduled' | 'manual'
  triggerUserId: number | null
  status: 'queued' | 'running' | 'success' | 'error'
  lotsChecked: number
  alertsSent: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage: string
  createdAt: string
}

export interface AuctionWatchBidDigestRun {
  id: number
  triggerType: 'scheduled' | 'manual'
  triggerUserId: number | null
  status: 'running' | 'success' | 'error'
  lotsChecked: number
  digestsSent: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage: string
  createdAt: string
}

export interface AuctionAlertReminderRun {
  id: number
  triggerType: 'scheduled' | 'manual'
  triggerUserId: number | null
  status: 'running' | 'success' | 'error'
  lotsChecked?: number
  alertsChecked?: number
  alertsTriggered?: number
  alertsSent?: number
  priceAlertsTriggered?: number
  remindersChecked?: number
  remindersNotified?: number
  remindersSent?: number
  bidRemindersSent?: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage?: string
  createdAt: string
}

export interface CollectionHealthSnapshotRunResult {
  message?: string
  users?: number
  snapshotsCreated?: number
  skipped?: number
  errors?: number
  durationMs?: number
}

export interface CollectionHealthSnapshotRun {
  id: number
  triggerType: 'manual' | 'scheduled'
  status: 'running' | 'success' | 'error'
  usersEligible: number
  usersSnapshotted: number
  usersFailed: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage?: string
  createdAt: string
}

export interface SchedulerStatus {
  name: string
  enabled: boolean
  isRunning: boolean
  nextRunIn: number
}

export interface CoinOfDayRun {
  id: number
  triggerType: 'manual' | 'scheduled'
  triggerUserId: number | null
  status: 'queued' | 'running' | 'completed' | 'failed'
  startedAt: string
  completedAt: string | null
  picked: number
  skipped: number
  errors: number
  errorMessage: string
  createdAt: string
  updatedAt: string
}

export interface Notification {
  id: number
  userId: number
  type: 'wishlist_unavailable' | 'friend_new_coin' | 'follow_request' | 'coin_of_day' | 'api_key_rotation_required' | 'set_milestone' | 'agentic_set_proposal_ready' | 'agentic_set_proposal_failed' | 'agentic_set_created' | 'agentic_set_creation_failed' | 'ai_job_completed' | 'ai_job_failed' | 'valuation_complete'
  title: string
  message: string
  referenceId: number
  referenceUrl?: string
  isRead: boolean
  createdAt: string
}

export interface NotificationListResponse {
  notifications: Notification[]
  total: number
  page: number
  limit: number
}

export interface FeaturedCoin {
  id: number
  userId: number
  coinId: number
  coin?: Coin
  summary: string
  featuredAt: string
  createdAt: string
}
