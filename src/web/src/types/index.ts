// Barrel for the app's shared types.
//
// These used to live in a single 2,438-line index.ts. They are now grouped
// by domain in sibling files; this barrel re-exports all of them so every
// existing `from '@/types'` import keeps working. New code may import from
// the specific module (e.g. '@/types/auctions') to make dependencies clear.

export * from '@/types/admin'
export * from '@/types/agent'
export * from '@/types/auctions'
export * from '@/types/auth'
export * from '@/types/coin'
export * from '@/types/collection'
export * from '@/types/deep-identification'
export * from '@/types/emperors'
export * from '@/types/notifications'
export * from '@/types/numista'
export * from '@/types/scheduler-runs'
export * from '@/types/sets'
export * from '@/types/shipment'
export * from '@/types/social'
export * from '@/types/stats'
export * from '@/types/wishlist'
