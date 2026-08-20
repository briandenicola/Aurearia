// social types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { Category, CoinImage, Material } from '@/types/coin'

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
