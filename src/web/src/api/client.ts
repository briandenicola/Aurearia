// Barrel for the API layer.
//
// This file used to hold every endpoint function in the app (~1,200 lines).
// The endpoints now live in ./endpoints/*.ts grouped by domain, and the shared
// axios instance, refresh-token rotation, and error formatting live in
// ./http.ts. This barrel re-exports all of it so that the ~200 existing
// `from '@/api/client'` imports keep working unchanged.
//
// New code may import from the specific module instead — e.g.
// `import { getCoins } from '@/api/endpoints/coins'` — which is cheaper for
// the bundler to tree-shake and makes a component's real dependencies obvious.

export * from '@/api/http'

export * from '@/api/endpoints/admin'
export * from '@/api/endpoints/agent'
export * from '@/api/endpoints/auctions'
export * from '@/api/endpoints/auth'
export * from '@/api/endpoints/coins'
export * from '@/api/endpoints/collection'
export * from '@/api/endpoints/notifications'
export * from '@/api/endpoints/sets'
export * from '@/api/endpoints/social'
export * from '@/api/endpoints/stats'
export * from '@/api/endpoints/wishlist'
