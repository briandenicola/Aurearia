export interface Coin {
  id: number
  name: string
  category: Category
  denomination: string
  ruler: string
  era: string
  mint: string
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
  userId: number
  images: CoinImage[]
  tags?: Tag[]
  createdAt: string
  updatedAt: string
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

export type Category = 'Roman' | 'Greek' | 'Byzantine' | 'Modern' | 'Other'
export type Material = 'Gold' | 'Silver' | 'Bronze' | 'Copper' | 'Electrum' | 'Other'
export type ImageType = 'obverse' | 'reverse' | 'detail' | 'other'

export const CATEGORIES: Category[] = ['Roman', 'Greek', 'Byzantine', 'Modern', 'Other']
export const MATERIALS: Material[] = ['Gold', 'Silver', 'Bronze', 'Copper', 'Electrum', 'Other']
export const IMAGE_TYPES: ImageType[] = ['obverse', 'reverse', 'detail', 'other']

export const CATEGORY_COLORS: Record<Category, string> = {
  Roman: '#7b2d8e',
  Greek: '#6b8e23',
  Byzantine: '#8b1a1a',
  Modern: '#4682b4',
  Other: '#888888',
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
}

export interface AuthResponse {
  token: string
  refreshToken: string
  user: User
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
  createdAt: string
}

export interface AppSettings {
  AIProvider: string
  OllamaURL: string
  OllamaModel: string
  ObversePrompt: string
  ReversePrompt: string
  TextExtractionPrompt: string
  OllamaTimeout: string
  SearXNGURL: string
  LogLevel: string
  [key: string]: string
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
}

export type Theme = 'dark' | 'light'

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
  createdAt: string
  lastUsedAt: string | null
  revokedAt: string | null
}

export interface AgentChatMessage {
  role: 'user' | 'assistant'
  content: string
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
}

export interface AgentChatResponse {
  message: string
  suggestions: CoinSuggestion[]
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

export interface AuctionLot {
  id: number
  numisBidsUrl: string
  saleId: string
  lotNumber: number
  auctionHouse: string
  saleName: string
  saleDate: string | null
  title: string
  description: string
  category: Category
  estimate: number | null
  currentBid: number | null
  maxBid: number | null
  currency: string
  status: AuctionLotStatus
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
  triggerType: string
  triggerUserId: number | null
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

export interface Notification {
  id: number
  userId: number
  type: 'wishlist_unavailable' | 'friend_new_coin'
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
