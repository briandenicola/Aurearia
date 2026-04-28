<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="lot-detail card">
      <div class="detail-header">
        <h2>{{ lot.title }}</h2>
        <button class="btn-close" @click="$emit('close')"><X :size="18" /></button>
      </div>

      <div v-if="lot.imageUrl" class="detail-image-container">
        <img :src="proxiedImageUrl" :alt="lot.title" class="detail-image" />
      </div>

      <div class="detail-body">
        <div class="detail-row" v-if="lot.auctionHouse">
          <span class="detail-label">Auction House</span>
          <span>{{ lot.auctionHouse }}</span>
        </div>
        <div class="detail-row" v-if="lot.saleName">
          <span class="detail-label">Sale</span>
          <span>{{ lot.saleName }}</span>
        </div>
        <div class="detail-row" v-if="lot.lotNumber">
          <span class="detail-label">Lot #</span>
          <span>{{ lot.lotNumber }}</span>
        </div>
        <div class="detail-row" v-if="lot.saleDate">
          <span class="detail-label">Sale Date</span>
          <span>{{ formatDate(lot.saleDate) }}</span>
        </div>
        <div class="detail-row" v-if="lot.estimate">
          <span class="detail-label">Estimate</span>
          <span>{{ formatCurrency(lot.estimate, lot.currency) }}</span>
        </div>
        <div class="detail-row" v-if="lot.currentBid">
          <span class="detail-label">Current Bid</span>
          <span class="bid-value">{{ formatCurrency(lot.currentBid, lot.currency) }}</span>
        </div>
        <div class="detail-row" v-if="lot.maxBid">
          <span class="detail-label">Max Bid</span>
          <span class="max-bid-value">{{ formatCurrency(lot.maxBid, lot.currency) }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">Status</span>
          <span class="status-tag" :class="`status-${lot.status}`">{{ lot.status }}</span>
        </div>
        <div v-if="lot.description" class="detail-description">
          <span class="detail-label">Description</span>
          <p>{{ lot.description }}</p>
        </div>
      </div>

      <div class="detail-actions">
        <div class="action-row">
          <select v-model="newStatus" class="form-input status-select">
            <option value="watching">Watching</option>
            <option value="bidding">Bidding</option>
            <option value="won">Won</option>
            <option value="lost">Lost</option>
            <option value="passed">Passed</option>
          </select>
          <button class="btn btn-secondary" @click="changeStatus" :disabled="newStatus === lot.status">
            Update Status
          </button>
        </div>
        <div v-if="newStatus === 'bidding'" class="action-row bid-input-row">
          <label class="detail-label">Max Bid</label>
          <input
            v-model.number="maxBidInput"
            type="number"
            class="form-input bid-input"
            :placeholder="lot.currency || 'USD'"
            min="0"
            step="1"
          />
        </div>
        <div class="action-row event-link-row">
          <label class="detail-label"><CalendarDays :size="14" /> Calendar Event</label>
          <div class="event-link-controls">
            <select v-model="selectedEventId" class="form-input event-select">
              <option value="">None</option>
              <option v-for="evt in calendarEvents" :key="evt.id" :value="evt.id">
                {{ evt.title }}
              </option>
            </select>
            <button
              class="btn btn-secondary btn-sm"
              @click="linkEvent"
              :disabled="(selectedEventId === '' ? null : Number(selectedEventId)) === (lot.eventId ?? null)"
            >
              Link
            </button>
          </div>
        </div>
        <div class="action-row">
          <a :href="lot.numisBidsUrl" class="btn btn-primary" target="_blank" rel="noopener noreferrer">
            <ExternalLink :size="14" /> View on NumisBids
          </a>
          <button v-if="lot.status === 'won'" class="btn btn-primary" @click="convertToCoin">
            <ArrowRightCircle :size="14" /> Add to Collection
          </button>
          <button class="btn btn-danger" @click="removeLot">
            <Trash2 :size="14" /> Remove
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { updateAuctionLotStatus, convertAuctionLotToCoin, deleteAuctionLot, listCalendarEvents, linkAuctionLotEvent } from '@/api/client'
import type { AuctionLot, AuctionLotStatus } from '@/types'
import { X, ExternalLink, ArrowRightCircle, Trash2, CalendarDays } from 'lucide-vue-next'
import { formatCurrency } from '@/utils/format'

const props = defineProps<{
  lot: AuctionLot
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const router = useRouter()
const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

const newStatus = ref<AuctionLotStatus>(props.lot.status)
const maxBidInput = ref<number | null>(props.lot.maxBid ?? null)
const calendarEvents = ref<Array<{ id: number; title: string; auctionHouse: string; startDate: string | null }>>([])
const selectedEventId = ref<number | string>(props.lot.eventId ?? '')

const proxiedImageUrl = computed(() => {
  if (!props.lot.imageUrl) return ''
  const token = localStorage.getItem('token') ?? ''
  return `${API_BASE}/api/proxy-image?url=${encodeURIComponent(props.lot.imageUrl)}&token=${encodeURIComponent(token)}`
})

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

async function fetchCalendarEvents() {
  try {
    const res = await listCalendarEvents()
    calendarEvents.value = res.data?.events ?? []
  } catch { /* ignore */ }
}

async function linkEvent() {
  const eventId = selectedEventId.value === '' ? null : Number(selectedEventId.value)
  try {
    await linkAuctionLotEvent(props.lot.id, eventId)
    emit('updated')
  } catch { /* ignore */ }
}

async function changeStatus() {
  try {
    const bid = newStatus.value === 'bidding' ? maxBidInput.value : undefined
    await updateAuctionLotStatus(props.lot.id, newStatus.value, bid)

    if (newStatus.value === 'won') {
      try {
        const coinRes = await convertAuctionLotToCoin(props.lot.id)
        emit('close')
        router.push(`/edit/${coinRes.data.id}`)
        return
      } catch { /* fall through */ }
    }

    emit('updated')
  } catch { /* ignore */ }
}

async function convertToCoin() {
  try {
    const coinRes = await convertAuctionLotToCoin(props.lot.id)
    emit('close')
    router.push(`/edit/${coinRes.data.id}`)
  } catch { /* ignore */ }
}

async function removeLot() {
  try {
    await deleteAuctionLot(props.lot.id)
    emit('close')
    emit('updated')
  } catch { /* ignore */ }
}

onMounted(fetchCalendarEvents)
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.lot-detail {
  max-width: 560px;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
  padding: 0;
}

.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
  gap: 1rem;
}

.detail-header h2 {
  font-size: 1.1rem;
  line-height: 1.35;
  margin: 0;
}

.btn-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}

.btn-close:hover {
  color: var(--text-primary);
}

.detail-image-container {
  width: 100%;
  max-height: 300px;
  overflow: hidden;
  background: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.detail-image {
  width: 100%;
  height: 300px;
  object-fit: contain;
}

.detail-body {
  padding: 1.25rem 1.5rem;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 0.88rem;
}

.detail-label {
  color: var(--text-secondary);
  font-size: 0.82rem;
}

.bid-value {
  font-weight: 600;
  color: var(--accent-gold);
}

.max-bid-value {
  font-weight: 600;
  color: var(--accent-gold);
  opacity: 0.8;
}

.bid-input-row {
  align-items: center;
}

.bid-input {
  flex: 1;
  max-width: 140px;
}

.status-tag {
  padding: 0.15rem 0.55rem;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.status-watching { background: rgba(100, 150, 255, 0.2); color: #6496ff; }
.status-bidding { background: var(--accent-gold-glow); color: var(--accent-gold); }
.status-won { background: rgba(74, 222, 128, 0.15); color: #4ade80; }
.status-lost { background: rgba(248, 113, 113, 0.15); color: #f87171; }
.status-passed { background: rgba(120, 120, 120, 0.15); color: #999; }

.detail-description {
  margin-top: 0.75rem;
}

.detail-description p {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-top: 0.4rem;
  line-height: 1.5;
  max-height: 120px;
  overflow-y: auto;
}

.detail-actions {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.action-row {
  display: flex;
  gap: 0.6rem;
  flex-wrap: wrap;
}

.status-select {
  flex: 1;
  min-width: 120px;
  max-width: 160px;
}

.btn-danger {
  background: transparent;
  border: 1px solid rgba(248, 113, 113, 0.4);
  color: #f87171;
  padding: 0.5rem 0.9rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.82rem;
  display: flex;
  align-items: center;
  gap: 0.35rem;
  transition: all var(--transition-fast);
}

.btn-danger:hover {
  background: rgba(248, 113, 113, 0.1);
  border-color: rgba(248, 113, 113, 0.6);
}

.event-link-row {
  flex-direction: column;
  gap: 0.4rem;
}

.event-link-row .detail-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.event-link-controls {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.event-select {
  flex: 1;
  min-width: 160px;
}

.btn-sm {
  padding: 0.35rem 0.7rem;
  font-size: 0.8rem;
}
</style>
