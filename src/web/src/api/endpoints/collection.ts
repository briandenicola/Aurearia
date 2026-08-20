// collection endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  GeocodeCandidate,
  ImperialFigureRole,
  MintLocation,
  NoteInput,
  NoteListResponse,
  RomanImperialFigure,
  StorageLocation,
  Tag,
  UserNote,
} from '@/types'

// User notes
export const getNotes = () => api.get<NoteListResponse>('/notes')

export const createNote = (note: NoteInput) => api.post<UserNote>('/notes', note)

export const updateNote = (id: number, note: NoteInput) => api.put<UserNote>(`/notes/${id}`, note)

export const deleteNote = (id: number) => api.delete(`/notes/${id}`)

// Tags
export const getTags = () => api.get<{ tags: Tag[] }>('/tags')

export const createTag = (data: { name: string; color?: string }) => api.post<Tag>('/tags', data)

export const updateTag = (id: number, data: { name?: string; color?: string }) => api.put<Tag>(`/tags/${id}`, data)

export const deleteTag = (id: number) => api.delete(`/tags/${id}`)

// Storage Locations
export const getStorageLocations = () => api.get<{ storageLocations: StorageLocation[] }>('/storage-locations')

export const createStorageLocation = (data: { name: string; sortOrder?: number }) => api.post<StorageLocation>('/storage-locations', data)

export const updateStorageLocation = (id: number, data: { name?: string; sortOrder?: number }) => api.put<StorageLocation>(`/storage-locations/${id}`, data)

export const deleteStorageLocation = (id: number) => api.delete(`/storage-locations/${id}`)

export type MintLocationInput = {
  displayName: string
  lat: number
  lng: number
  region?: string
  aliases: string[]
}

export type MintLocationsResponse = MintLocation[] | { mintLocations?: MintLocation[] }

// Mint Locations
export const getMintLocations = () => api.get<MintLocationsResponse>('/mint-locations')

export const createMintLocation = (data: MintLocationInput) =>
  api.post<MintLocation>('/mint-locations', data)

export const updateMintLocation = (id: number, data: MintLocationInput) =>
  api.put<MintLocation>(`/mint-locations/${id}`, data)

export const deleteMintLocation = (id: number) => api.delete(`/mint-locations/${id}`)

export const geocodeMintName = (query: string) =>
  api.get<{ candidates: GeocodeCandidate[] }>('/mint-locations/geocode', { params: { query } })

// Roman Imperial Figures (F028)
export const searchRomanImperialFigures = (params: { q?: string; role?: ImperialFigureRole; limit?: number }) =>
  api.get<{ figures: RomanImperialFigure[] }>('/roman-imperial-figures', { params })

export const getRomanImperialFigure = (id: number) =>
  api.get<RomanImperialFigure>(`/roman-imperial-figures/${id}`)
