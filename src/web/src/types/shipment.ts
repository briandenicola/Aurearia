// shipment types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.

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
