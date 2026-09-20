import { api } from '@/api/http'
import type { CollectorProfile, CollectorProfileInput } from '@/types/collectorProfile'

export const getCollectorProfile = () =>
  api.get<CollectorProfile>('/collector-profile')

export const replaceCollectorProfile = (profile: CollectorProfileInput) =>
  api.put<CollectorProfile>('/collector-profile', profile)
