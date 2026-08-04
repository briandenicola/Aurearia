<template>
  <section class="mb-6">
    <h3 class="mb-3 font-display text-base font-medium text-heading">Shipment Tracking</h3>
    <div class="rounded-sm border border-border-subtle bg-card p-4">
      <div v-if="loading" class="text-body text-text-secondary">Loading shipment...</div>

      <template v-else-if="shipment">
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <span class="chip-sm">{{ carrierLabel(shipment.carrier, shipment.manualCarrierName) }}</span>
          <span class="chip-sm">{{ statusLabel(shipment.currentStatus) }}</span>
          <a v-if="trackingUrl" :href="trackingUrl" target="_blank" rel="noopener" class="text-chip text-gold hover:underline">Open carrier tracking</a>
        </div>
        <div class="mb-4 grid gap-2 text-body text-text-secondary">
          <p class="m-0"><strong class="text-text-primary">Tracking:</strong> {{ shipment.trackingNumber }}</p>
          <p v-if="shipment.notes" class="m-0"><strong class="text-text-primary">Notes:</strong> {{ shipment.notes }}</p>
          <p class="m-0"><strong class="text-text-primary">Status:</strong> {{ statusLabel(shipment.currentStatus) }}</p>
          <p v-if="shipment.lastSyncError" class="m-0 text-[var(--color-negative)]"><strong>ParcelApp:</strong> {{ shipment.lastSyncError }}</p>
        </div>

        <div class="mb-4 grid gap-2 border-t border-border-subtle pt-4">
          <label class="form-label">Update Shipment Status</label>
          <p class="m-0 text-body text-text-secondary">Shipment status is managed manually.</p>
          <div class="grid gap-2 md:grid-cols-2">
            <select v-model="manualStatus" class="form-input">
              <option v-for="value in statusOptions" :key="value" :value="value">{{ statusLabel(value) }}</option>
            </select>
            <input v-model="manualNote" class="form-input" placeholder="Optional status note" />
          </div>
          <button class="btn btn-secondary btn-sm w-fit" :disabled="saving" @click="saveStatus">
            {{ saving ? 'Saving...' : 'Save Status' }}
          </button>
        </div>

        <div class="mb-4 grid gap-2 border-t border-border-subtle pt-4">
          <h4 class="m-0 text-sm font-semibold text-text-primary">Timeline</h4>
          <p v-if="!shipment.events?.length" class="m-0 text-body text-text-muted">No tracking events yet.</p>
          <ul v-else class="m-0 grid list-none gap-2 p-0">
            <li v-for="event in shipment.events" :key="event.id" class="rounded-sm border border-border-subtle bg-card-hover px-3 py-2 text-body text-text-secondary">
              <p class="m-0 font-medium text-text-primary">{{ statusLabel(event.status) }}</p>
              <p class="m-0">{{ formatDate(event.occurredAt) }}<span v-if="event.location"> • {{ event.location }}</span></p>
              <p v-if="event.description" class="m-0">{{ event.description }}</p>
            </li>
          </ul>
        </div>

        <div class="flex flex-wrap gap-2 border-t border-border-subtle pt-4">
          <button class="btn btn-secondary btn-sm" :disabled="syncing || shipment.carrier !== 'parcel'" @click="syncShipment">{{ syncing ? 'Checking...' : 'Check ParcelApp Now' }}</button>
          <button class="btn btn-danger btn-sm" :disabled="deleting" @click="removeShipment">{{ deleting ? 'Removing...' : 'Remove Shipment' }}</button>
        </div>
      </template>

      <template v-else>
        <p class="m-0 mb-4 text-body text-text-secondary">No shipment attached to this coin yet.</p>
        <p class="m-0 mb-4 text-body text-text-secondary">Enter the tracking number only. Aurearia uses ParcelApp and the coin title as the delivery description.</p>
        <div class="grid gap-3 md:grid-cols-2">
          <input v-model="form.trackingNumber" class="form-input" placeholder="Tracking number" />
          <input v-model="form.notes" class="form-input md:col-span-2" placeholder="Optional notes" />
        </div>
        <div class="mt-4">
          <button class="btn btn-primary btn-sm" :disabled="saving" @click="saveShipment">{{ saving ? 'Saving...' : 'Save Shipment' }}</button>
        </div>
      </template>

      <p v-if="message" class="mt-3 mb-0 text-body" :class="messageError ? 'text-[var(--color-negative)]' : 'text-gold'">{{ message }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { deleteCoinShipment, getCoinShipment, setCoinShipmentManualOverride, syncCoinShipment, upsertCoinShipment } from '@/api/client'
import type { Shipment, ShipmentCarrier, ShipmentStatus, ShipmentUpsertInput } from '@/types'

const props = defineProps<{ coinId: number }>()
const emit = defineEmits<{ changed: [] }>()

const loading = ref(true)
const saving = ref(false)
const syncing = ref(false)
const deleting = ref(false)
const shipment = ref<Shipment | null>(null)
const trackingUrl = ref('')
const message = ref('')
const messageError = ref(false)
const manualStatus = ref<ShipmentStatus>('in_transit')
const manualNote = ref('')
const form = reactive<ShipmentUpsertInput>({
  carrier: 'parcel',
  trackingNumber: '',
  notes: '',
  manualCarrierName: '',
})

const statusOptions: ShipmentStatus[] = ['pending', 'label_created', 'in_transit', 'out_for_delivery', 'delivered', 'exception', 'returned', 'unknown']

function setMessage(next: string, isError = false) {
  message.value = next
  messageError.value = isError
}

function apiErrorMessage(err: unknown): string {
  const maybe = err as { response?: { data?: { error?: string; message?: string } }; message?: string }
  return maybe?.response?.data?.error || maybe?.response?.data?.message || maybe?.message || ''
}

function statusLabel(status: string) {
  return status.replaceAll('_', ' ').replace(/\b\w/g, c => c.toUpperCase())
}

function carrierLabel(carrier: ShipmentCarrier, manualName: string) {
  if (carrier === 'parcel') return 'ParcelApp'
  if (carrier === 'other' && manualName) return manualName
  return carrier.toUpperCase()
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

async function loadShipment() {
  loading.value = true
  try {
    const res = await getCoinShipment(props.coinId)
    shipment.value = res.data.shipment
    trackingUrl.value = res.data.trackingUrl ?? ''
    manualStatus.value = (shipment.value.currentStatus || shipment.value.manualOverrideStatus || 'in_transit') as ShipmentStatus
    manualNote.value = shipment.value.manualOverrideNote || ''
  } catch (err: unknown) {
    const msg = apiErrorMessage(err).toLowerCase()
    if (msg.includes('shipment not found')) {
      shipment.value = null
      trackingUrl.value = ''
    } else {
      setMessage(apiErrorMessage(err) || 'Failed to load shipment', true)
    }
  } finally {
    loading.value = false
  }
}

function buildUpsertInput(): ShipmentUpsertInput | null {
  const trackingNumber = form.trackingNumber.trim()
  if (!trackingNumber) {
    setMessage('Tracking number is required', true)
    return null
  }
  return {
    carrier: 'parcel',
    trackingNumber,
    notes: form.notes?.trim() || '',
    manualCarrierName: '',
  }
}

async function saveShipment() {
  const input = buildUpsertInput()
  if (!input) return
  saving.value = true
  setMessage('')
  try {
    const res = await upsertCoinShipment(props.coinId, input)
    shipment.value = res.data.shipment
    trackingUrl.value = res.data.trackingUrl ?? ''
    if (shipment.value.lastSyncError) {
      setMessage(`Shipment saved. ParcelApp check failed: ${shipment.value.lastSyncError}`, true)
    } else {
      setMessage('Shipment saved')
    }
    emit('changed')
  } catch (err: unknown) {
    setMessage(apiErrorMessage(err) || 'Failed to save shipment', true)
  } finally {
    saving.value = false
  }
}

async function saveStatus() {
  if (!shipment.value) return
  saving.value = true
  setMessage('')
  try {
    const res = await setCoinShipmentManualOverride(props.coinId, {
      enabled: true,
      status: manualStatus.value,
      note: manualNote.value.trim(),
    })
    shipment.value = res.data.shipment
    trackingUrl.value = res.data.trackingUrl ?? ''
    setMessage('Shipment status saved')
  } catch (err: unknown) {
    setMessage(apiErrorMessage(err) || 'Failed to save shipment status', true)
  } finally {
    saving.value = false
  }
}

async function syncShipment() {
  if (!shipment.value) return
  syncing.value = true
  setMessage('')
  try {
    const res = await syncCoinShipment(props.coinId)
    shipment.value = res.data.shipment
    trackingUrl.value = res.data.trackingUrl ?? ''
    if (shipment.value.lastSyncError) {
      setMessage(`ParcelApp check failed: ${shipment.value.lastSyncError}`, true)
    } else {
      setMessage('ParcelApp status checked')
    }
  } catch (err: unknown) {
    setMessage(apiErrorMessage(err) || 'Failed to check ParcelApp status', true)
  } finally {
    syncing.value = false
  }
}

async function removeShipment() {
  deleting.value = true
  setMessage('')
  try {
    await deleteCoinShipment(props.coinId)
    shipment.value = null
    trackingUrl.value = ''
    setMessage('Shipment removed')
    emit('changed')
  } catch (err: unknown) {
    setMessage(apiErrorMessage(err) || 'Failed to remove shipment', true)
  } finally {
    deleting.value = false
  }
}

onMounted(loadShipment)
</script>
