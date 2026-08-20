// social endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import axios from 'axios'
import { API_BASE, api } from '@/api/http'
import type {
  CoinComment,
  CoinRating,
  FollowUser,
  LimitedCoin,
  PublicProfile,
} from '@/types'

// Follow
export const followUser = (userId: number) => api.post(`/social/follow/${userId}`)

export const unfollowUser = (userId: number) => api.delete(`/social/follow/${userId}`)

export const acceptFollower = (userId: number) => api.put(`/social/followers/${userId}/accept`)

export const blockFollower = (userId: number) => api.put(`/social/followers/${userId}/block`)

// Showcases
export const listShowcases = () => api.get('/showcases')

export const getShowcase = (id: number) => api.get(`/showcases/${id}`)

export const createShowcase = (data: { title: string; description?: string }) => api.post('/showcases', data)

export const updateShowcase = (id: number, data: { title?: string; description?: string; isActive?: boolean }) => api.put(`/showcases/${id}`, data)

export const deleteShowcase = (id: number) => api.delete(`/showcases/${id}`)

export const setShowcaseCoins = (id: number, coinIds: number[]) => api.put(`/showcases/${id}/coins`, { coinIds })

// Uses bare axios rather than the shared `api` instance on purpose: the
// public showcase route must be requested without an Authorization header.
export const getPublicShowcase = (slug: string) => axios.get(`${API_BASE}/api/showcase/${slug}`)

export const unblockFollower = (userId: number) => api.delete(`/social/followers/${userId}/block`)

export const getFollowers = () => api.get<{ followers: FollowUser[] }>('/social/followers')

export const getFollowing = () => api.get<{ following: FollowUser[] }>('/social/following')

export const getBlockedUsers = () => api.get<{ blocked: { id: number; username: string; avatarPath: string }[] }>('/social/blocked')

// User discovery
export const searchUsers = (query: string) => api.get<{ users: FollowUser[] }>('/users/search', { params: { q: query } })

export const getPublicProfile = (username: string) => api.get<PublicProfile>(`/users/${encodeURIComponent(username)}`)

// Follower coins
export const getFollowingCoins = (userId: number) =>
  api.get<{ coins: LimitedCoin[]; username: string }>(`/social/following/${userId}/coins`)

export const getFollowingCoinDetail = (userId: number, coinId: number) =>
  api.get<LimitedCoin & { comments: CoinComment[]; rating: CoinRating }>(`/social/following/${userId}/coins/${coinId}`)

// Comments & ratings
export const addComment = (coinId: number, comment: string, rating?: number) =>
  api.post<CoinComment>(`/social/coins/${coinId}/comments`, { comment, rating: rating || 0 })

export const deleteComment = (coinId: number, commentId: number) =>
  api.delete(`/social/coins/${coinId}/comments/${commentId}`)

export const rateCoin = (coinId: number, rating: number) =>
  api.put<CoinRating>(`/social/coins/${coinId}/rating`, { rating })
