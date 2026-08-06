<template>
  <section class="card">
    <div class="mb-4 flex items-center justify-between gap-3 border-b border-border-subtle pb-3">
      <div>
        <h2 class="text-xl font-medium">Shipments</h2>
        <p class="mt-1 text-sm text-text-muted">All shipment statuses except delivered.</p>
      </div>
      <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadShipments">
        {{ loading ? 'Refreshing...' : 'Refresh' }}
      </button>
    </div>

    <div v-if="loading" class="text-sm text-text-secondary">Loading shipment summary...</div>
    <p v-else-if="errorMessage" class="text-sm text-[var(--color-negative)]">{{ errorMessage }}</p>
    <template v-else>
      <p class="mb-3 text-sm text-text-muted">
        Total active shipments: <strong class="text-text-primary">{{ rows.length }}</strong>
      </p>
      <div v-if="rows.length" class="overflow-x-auto rounded-sm border border-border-subtle">
        <table class="min-w-full border-collapse">
          <thead class="bg-card">
            <tr>
              <th class="px-3 py-2 text-left text-[0.7rem] font-semibold uppercase tracking-[0.08em] text-text-muted">Coin</th>
              <th class="px-3 py-2 text-left text-[0.7rem] font-semibold uppercase tracking-[0.08em] text-text-muted">Status</th>
              <th class="px-3 py-2 text-left text-[0.7rem] font-semibold uppercase tracking-[0.08em] text-text-muted">Carrier</th>
              <th class="px-3 py-2 text-left text-[0.7rem] font-semibold uppercase tracking-[0.08em] text-text-muted">Tracking</th>
              <th class="px-3 py-2 text-left text-[0.7rem] font-semibold uppercase tracking-[0.08em] text-text-muted">ETA</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.id" class="border-t border-border-subtle">
              <td class="px-3 py-2 text-sm text-text-primary">
                <RouterLink :to="`/coin/${row.coinId}/shipment`" class="text-gold hover:underline">
                  {{ row.coinName }}
                </RouterLink>
                <p v-if="row.lastSyncError" class="m-0 mt-1 text-xs text-[var(--color-negative)]">
                  {{ row.lastSyncError }}
                </p>
              </td>
              <td class="px-3 py-2 text-sm text-text-secondary">
                <span class="chip-sm">{{ statusLabel(row.currentStatus) }}</span>
              </td>
              <td class="px-3 py-2 text-sm text-text-secondary">{{ carrierLabel(row.carrier, row.manualCarrierName) }}</td>
              <td class="px-3 py-2 text-sm text-text-secondary">{{ row.trackingNumber }}</td>
              <td class="px-3 py-2 text-sm text-text-secondary">{{ row.estimatedDeliveryAt ? formatDate(row.estimatedDeliveryAt) : '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="text-sm text-text-muted">No active shipments right now.</p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getCoinShipment, getCoins } from '@/api/client'
import type { Coin, Shipment, ShipmentCarrier, ShipmentStatus } from '@/types'

type ActiveShipmentRow = Shipment & { coinName: string }

const loading = ref(false)
const errorMessage = ref('')
const rows = ref<ActiveShipmentRow[]>([])

const trackedStatuses: ShipmentStatus[] = [
  'pending',
  'label_created',
  'in_transit',
  'out_for_delivery',
  'exception',
  'returned',
  'unknown',
]

function statusLabel(status: string) {
  return status.replaceAll('_', ' ').replace(/\b\w/g, c => c.toUpperCase())
}

function carrierLabel(carrier: ShipmentCarrier, manualName: string) {
  if (carrier === 'parcel') return 'ParcelApp'
  if (carrier === 'other' && manualName) return manualName
  return carrier.toUpperCase()
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString()
}

function apiErrorMessage(err: unknown): string {
  const maybe = err as { response?: { data?: { error?: string; message?: string } }; message?: string }
  return maybe?.response?.data?.error || maybe?.response?.data?.message || maybe?.message || ''
}

function isShipmentNotFound(err: unknown): boolean {
  return apiErrorMessage(err).toLowerCase().includes('shipment not found')
}

async function fetchAllCollectionCoins(): Promise<Coin[]> {
  const all: Coin[] = []
  let page = 1
  let total = 0

  do {
    const res = await getCoins({ sold: 'false', page, limit: 100 })
    const coins = res.data.coins || []
    all.push(...coins)
    total = res.data.total || all.length
    page += 1
    if (!coins.length) break
  } while (all.length < total)

  return all
}

async function mapInBatches<T, R>(items: T[], batchSize: number, mapper: (item: T) => Promise<R>): Promise<R[]> {
  const out: R[] = []
  for (let i = 0; i < items.length; i += batchSize) {
    const batch = items.slice(i, i + batchSize)
    const results = await Promise.all(batch.map(mapper))
    out.push(...results)
  }
  return out
}

function statusOrder(status: ShipmentStatus): number {
  if (status === 'out_for_delivery') return 0
  if (status === 'in_transit') return 1
  if (status === 'pending') return 2
  if (status === 'label_created') return 3
  if (status === 'exception') return 4
  if (status === 'returned') return 5
  return 6
}

async function loadShipments() {
  loading.value = true
  errorMessage.value = ''
  try {
    const coins = await fetchAllCollectionCoins()
    const results = await mapInBatches(coins, 10, async (coin) => {
      try {
        const res = await getCoinShipment(coin.id)
        return { coin, shipment: res.data.shipment } as { coin: Coin; shipment: Shipment } | null
      } catch (err: unknown) {
        if (isShipmentNotFound(err)) return null
        throw err
      }
    })

    rows.value = results
      .filter((entry): entry is { coin: Coin; shipment: Shipment } => !!entry)
      .filter(({ shipment }) => trackedStatuses.includes(shipment.currentStatus))
      .map(({ coin, shipment }) => ({
        ...shipment,
        coinName: coin.name,
      }))
      .sort((a, b) => {
        const statusDelta = statusOrder(a.currentStatus) - statusOrder(b.currentStatus)
        if (statusDelta !== 0) return statusDelta
        const aETA = a.estimatedDeliveryAt ? new Date(a.estimatedDeliveryAt).getTime() : Number.MAX_SAFE_INTEGER
        const bETA = b.estimatedDeliveryAt ? new Date(b.estimatedDeliveryAt).getTime() : Number.MAX_SAFE_INTEGER
        if (aETA !== bETA) return aETA - bETA
        return b.id - a.id
      })
  } catch (err: unknown) {
    errorMessage.value = apiErrorMessage(err) || 'Failed to load shipment summary'
  } finally {
    loading.value = false
  }
}

onMounted(loadShipments)

defineExpose({ loadShipments })
</script>
