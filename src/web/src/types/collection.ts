// collection types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.

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
