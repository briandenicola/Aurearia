<template>
  <div ref="pullContainer" class="container" :style="pullDistance > 0 ? `transform: translateY(${pullDistance}px); transition: none` : ''">
    <div class="pull-indicator" :class="{ visible: pullDistance > 0 || refreshing, refreshing }" :style="`top: ${-50 + pullDistance * 0.6}px; opacity: ${Math.min(pullDistance / 60, 1)}`">
      <div class="pull-spinner" :style="refreshing ? '' : `transform: rotate(${pullDistance * 3}deg)`"></div>
      <span class="pull-text">{{ refreshing ? 'Refreshing...' : pullDistance >= 60 ? 'Release to refresh' : 'Pull to refresh' }}</span>
    </div>
    <div class="page-header">
      <h1>Auction Calendar</h1>
      <button class="btn btn-primary" @click="showAddEvent = true">
        <Plus :size="16" /> Add Event
      </button>
    </div>

    <!-- Month Navigation -->
    <div class="month-nav">
      <button class="btn btn-secondary" @click="prevMonth"><ChevronLeft :size="18" /></button>
      <h2 class="month-label">{{ monthLabel }}</h2>
      <button class="btn btn-secondary" @click="nextMonth"><ChevronRight :size="18" /></button>
    </div>

    <!-- Calendar Grid -->
    <div class="calendar-grid">
      <div v-for="day in dayNames" :key="day" class="day-header">{{ day }}</div>
      <div
        v-for="(cell, idx) in calendarCells"
        :key="idx"
        class="day-cell"
        :class="{
          'other-month': !cell.currentMonth,
          'is-today': cell.isToday,
          'has-events': (cell.lots?.length ?? 0) > 0 || (cell.events?.length ?? 0) > 0
        }"
      >
        <span class="day-number">{{ cell.day }}</span>
        <div class="day-indicators">
          <span
            v-for="n in Math.min(cell.lots?.length ?? 0, 3)"
            :key="'lot-' + n"
            class="indicator lot-indicator"
            title="Auction lot"
          ></span>
          <span
            v-for="n in Math.min(cell.events?.length ?? 0, 3)"
            :key="'ev-' + n"
            class="indicator event-indicator"
            title="Event"
          ></span>
        </div>
      </div>
    </div>

    <!-- Event List -->
    <div class="event-list-section">
      <h2>Events This Month</h2>

      <div v-if="loading" class="loading-state">Loading calendar...</div>

      <template v-else>
        <!-- Auction Lots -->
        <div v-if="lots.length" class="event-group">
          <h3 class="group-title lot-accent">Auction Lots</h3>
          <div v-for="lot in lots" :key="'lot-' + lot.id" class="card event-card">
            <div class="event-card-body">
              <div v-if="lot.imageUrl" class="lot-thumb-container">
                <img :src="lot.imageUrl" class="lot-thumb" alt="" />
              </div>
              <div class="event-info">
                <h4>{{ lot.title }}</h4>
                <div class="event-meta">
                  <span v-if="lot.auctionHouse"><Building :size="13" /> {{ lot.auctionHouse }}</span>
                  <span v-if="lot.saleDate"><CalendarIcon :size="13" /> {{ formatDate(lot.saleDate) }}</span>
                  <span v-if="lot.currentBid" class="bid-info">Current bid: {{ lot.currentBid }}</span>
                  <span v-if="lot.estimate" class="estimate-info">Est: {{ lot.estimate }}</span>
                </div>
                <a v-if="lot.numisBidsUrl" :href="lot.numisBidsUrl" target="_blank" rel="noopener" class="lot-link">
                  <ExternalLink :size="13" /> View on NumisBids
                </a>
              </div>
            </div>
          </div>
        </div>

        <!-- Manual Events -->
        <div v-if="events.length" class="event-group">
          <h3 class="group-title event-accent">Events</h3>
          <div v-for="ev in events" :key="'ev-' + ev.id" class="card event-card event-card-clickable" @click="openEvent(ev.id)">
            <div class="event-card-body">
              <div class="event-info">
                <h4>{{ ev.title }}</h4>
                <div class="event-meta">
                  <span v-if="ev.auctionHouse"><Building :size="13" /> {{ ev.auctionHouse }}</span>
                  <span v-if="ev.startDate">
                    <CalendarIcon :size="13" />
                    {{ formatDate(ev.startDate) }}
                    <template v-if="ev.endDate"> - {{ formatDate(ev.endDate) }}</template>
                  </span>
                </div>
                <p v-if="ev.notes" class="event-notes">{{ ev.notes }}</p>
                <a v-if="ev.url" :href="ev.url" target="_blank" rel="noopener" class="lot-link" @click.stop>
                  <ExternalLink :size="13" /> Visit
                </a>
              </div>
              <button class="btn-remove" @click.stop="handleDeleteEvent(ev.id)" title="Delete event">
                <Trash2 :size="16" />
              </button>
            </div>
          </div>
        </div>

        <div v-if="!lots.length && !events.length" class="empty-state">
          <CalendarIcon :size="48" />
          <h3>Nothing scheduled this month</h3>
          <p>Auction lots and manually added events will appear here.</p>
        </div>
      </template>
    </div>

    <!-- Event Detail Drawer -->
    <div v-if="selectedEvent" class="modal-overlay" @click.self="selectedEvent = null">
      <div class="modal card event-detail-modal">
        <div class="modal-header">
          <h2>Edit Event</h2>
          <button class="btn-close" @click="selectedEvent = null"><X :size="18" /></button>
        </div>
        <form @submit.prevent="handleUpdateEvent">
          <div class="form-group">
            <label for="edit-title">Title</label>
            <input id="edit-title" v-model="editEvent.title" type="text" required />
          </div>
          <div class="form-group">
            <label for="edit-house">Auction House</label>
            <input id="edit-house" v-model="editEvent.auctionHouse" type="text" />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label for="edit-start">Start Date</label>
              <input id="edit-start" v-model="editEvent.startDate" type="date" />
            </div>
            <div class="form-group">
              <label for="edit-end">End Date</label>
              <input id="edit-end" v-model="editEvent.endDate" type="date" />
            </div>
          </div>
          <div class="form-group">
            <label for="edit-url">URL</label>
            <input id="edit-url" v-model="editEvent.url" type="url" />
          </div>
          <div class="form-group">
            <label for="edit-notes">Notes</label>
            <textarea id="edit-notes" v-model="editEvent.notes" rows="3"></textarea>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="selectedEvent = null">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="savingEvent">
              {{ savingEvent ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </form>

        <!-- Linked Auction Lots -->
        <div class="linked-lots-section">
          <h3 class="linked-lots-title">Linked Auction Lots <span v-if="linkedLots.length" class="linked-count">{{ linkedLots.length }}</span></h3>
          <div v-if="linkedLots.length" class="linked-lots-list">
            <div v-for="lot in linkedLots" :key="lot.id" class="linked-lot-item">
              <div v-if="lot.imageUrl" class="lot-thumb-container">
                <img :src="proxiedUrl(lot.imageUrl)" class="lot-thumb" alt="" />
              </div>
              <div class="linked-lot-info">
                <span class="linked-lot-title">{{ lot.title }}</span>
                <span class="linked-lot-meta">
                  <template v-if="lot.lotNumber">Lot {{ lot.lotNumber }}</template>
                  <template v-if="lot.lotNumber && lot.status"> · </template>
                  <span class="status-tag" :class="`status-${lot.status}`">{{ lot.status }}</span>
                </span>
              </div>
              <a v-if="lot.numisBidsUrl" :href="lot.numisBidsUrl" target="_blank" rel="noopener" class="lot-ext-link" @click.stop>
                <ExternalLink :size="13" />
              </a>
            </div>
          </div>
          <p v-else class="no-linked-lots">No auction lots linked to this event. Link lots from the Auctions page.</p>
        </div>
      </div>
    </div>

    <!-- Add Event Modal -->
    <div v-if="showAddEvent" class="modal-overlay" @click.self="showAddEvent = false">
      <div class="modal card">
        <div class="modal-header">
          <h2>Add Event</h2>
          <button class="btn-close" @click="showAddEvent = false"><X :size="18" /></button>
        </div>
        <form @submit.prevent="handleCreateEvent">
          <div class="form-group">
            <label for="ev-title">Title</label>
            <input id="ev-title" v-model="newEvent.title" type="text" required placeholder="Event title" />
          </div>
          <div class="form-group">
            <label for="ev-house">Auction House (optional)</label>
            <input id="ev-house" v-model="newEvent.auctionHouse" type="text" placeholder="e.g. Heritage Auctions" />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label for="ev-start">Start Date</label>
              <input id="ev-start" v-model="newEvent.startDate" type="date" />
            </div>
            <div class="form-group">
              <label for="ev-end">End Date</label>
              <input id="ev-end" v-model="newEvent.endDate" type="date" />
            </div>
          </div>
          <div class="form-group">
            <label for="ev-url">URL (optional)</label>
            <input id="ev-url" v-model="newEvent.url" type="url" placeholder="https://..." />
          </div>
          <div class="form-group">
            <label for="ev-notes">Notes (optional)</label>
            <textarea id="ev-notes" v-model="newEvent.notes" rows="3" placeholder="Any additional notes"></textarea>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showAddEvent = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="creatingEvent">
              {{ creatingEvent ? 'Adding...' : 'Add Event' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  Plus, ChevronLeft, ChevronRight, X, Trash2,
  ExternalLink, Building, Calendar as CalendarIcon
} from 'lucide-vue-next'
import { getCalendar, getCalendarEvent, createCalendarEvent, updateCalendarEvent, deleteCalendarEvent } from '@/api/client'
import type { AuctionLot } from '@/types'
import { usePullToRefresh } from '@/composables/usePullToRefresh'

interface CalendarLot {
  id: number
  type: string
  title: string
  auctionHouse?: string
  status?: string
  currentBid?: string
  estimate?: string
  numisBidsUrl?: string
  imageUrl?: string
  saleDate?: string
  auctionEndTime?: string
}

interface CalendarEvent {
  id: number
  type: string
  title: string
  auctionHouse?: string
  startDate?: string
  endDate?: string
  url?: string
  notes?: string
}

interface CalendarCell {
  day: number
  currentMonth: boolean
  isToday: boolean
  dateStr: string
  lots?: CalendarLot[]
  events?: CalendarEvent[]
}

const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const monthNames = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']

const loading = ref(true)
const currentYear = ref(new Date().getFullYear())
const currentMonth = ref(new Date().getMonth())
const lots = ref<CalendarLot[]>([])
const events = ref<CalendarEvent[]>([])
const showAddEvent = ref(false)
const creatingEvent = ref(false)
const selectedEvent = ref<CalendarEvent | null>(null)
const linkedLots = ref<AuctionLot[]>([])
const savingEvent = ref(false)
const editEvent = ref({ title: '', auctionHouse: '', startDate: '', endDate: '', url: '', notes: '' })

const newEvent = ref({
  title: '',
  auctionHouse: '',
  startDate: '',
  endDate: '',
  url: '',
  notes: ''
})

const monthLabel = computed(() => `${monthNames[currentMonth.value] ?? ''} ${currentYear.value}`)

const rangeStart = computed(() => {
  const d = new Date(currentYear.value, currentMonth.value, 1)
  return d.toISOString().split('T')[0] ?? ''
})

const rangeEnd = computed(() => {
  const d = new Date(currentYear.value, currentMonth.value + 1, 0)
  return d.toISOString().split('T')[0] ?? ''
})

const calendarCells = computed<CalendarCell[]>(() => {
  const year = currentYear.value
  const month = currentMonth.value
  const firstDay = new Date(year, month, 1).getDay()
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const daysInPrevMonth = new Date(year, month, 0).getDate()
  const today = new Date()
  const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`

  const cells: CalendarCell[] = []

  // Build a map of date string -> lots/events
  const lotsByDate = new Map<string, CalendarLot[]>()
  const eventsByDate = new Map<string, CalendarEvent[]>()

  for (const lot of lots.value) {
    const dateStr = lot.saleDate?.split('T')?.[0] ?? ''
    if (dateStr) {
      if (!lotsByDate.has(dateStr)) lotsByDate.set(dateStr, [])
      lotsByDate.get(dateStr)!.push(lot)
    }
  }

  for (const ev of events.value) {
    const dateStr = ev.startDate?.split('T')?.[0] ?? ''
    if (dateStr) {
      if (!eventsByDate.has(dateStr)) eventsByDate.set(dateStr, [])
      eventsByDate.get(dateStr)!.push(ev)
    }
  }

  // Previous month padding
  for (let i = firstDay - 1; i >= 0; i--) {
    const day = daysInPrevMonth - i
    const m = month === 0 ? 12 : month
    const y = month === 0 ? year - 1 : year
    const ds = `${y}-${String(m).padStart(2, '0')}-${String(day).padStart(2, '0')}`
    cells.push({ day, currentMonth: false, isToday: false, dateStr: ds })
  }

  // Current month
  for (let d = 1; d <= daysInMonth; d++) {
    const ds = `${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    cells.push({
      day: d,
      currentMonth: true,
      isToday: ds === todayStr,
      dateStr: ds,
      lots: lotsByDate.get(ds),
      events: eventsByDate.get(ds)
    })
  }

  // Next month padding (fill to 42 cells = 6 rows)
  const remaining = 42 - cells.length
  for (let d = 1; d <= remaining; d++) {
    const m = month + 2 > 12 ? 1 : month + 2
    const y = month + 2 > 12 ? year + 1 : year
    const ds = `${y}-${String(m).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    cells.push({ day: d, currentMonth: false, isToday: false, dateStr: ds })
  }

  return cells
})

function prevMonth() {
  if (currentMonth.value === 0) {
    currentMonth.value = 11
    currentYear.value--
  } else {
    currentMonth.value--
  }
}

function nextMonth() {
  if (currentMonth.value === 11) {
    currentMonth.value = 0
    currentYear.value++
  } else {
    currentMonth.value++
  }
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

async function loadCalendar() {
  loading.value = true
  try {
    const res = await getCalendar(rangeStart.value, rangeEnd.value)
    lots.value = res.data?.lots ?? []
    events.value = res.data?.events ?? []
  } catch {
    lots.value = []
    events.value = []
  } finally {
    loading.value = false
  }
}

async function handleCreateEvent() {
  if (!newEvent.value.title.trim()) return
  creatingEvent.value = true
  try {
    const data: Record<string, string | undefined> = {
      title: newEvent.value.title.trim(),
      auctionHouse: newEvent.value.auctionHouse.trim() || undefined,
      startDate: newEvent.value.startDate ? newEvent.value.startDate + 'T00:00:00Z' : undefined,
      endDate: newEvent.value.endDate ? newEvent.value.endDate + 'T00:00:00Z' : undefined,
      url: newEvent.value.url.trim() || undefined,
      notes: newEvent.value.notes.trim() || undefined
    }
    await createCalendarEvent(data as Parameters<typeof createCalendarEvent>[0])
    showAddEvent.value = false
    newEvent.value = { title: '', auctionHouse: '', startDate: '', endDate: '', url: '', notes: '' }
    await loadCalendar()
  } finally {
    creatingEvent.value = false
  }
}

async function handleDeleteEvent(id: number) {
  try {
    await deleteCalendarEvent(id)
    events.value = events.value.filter(e => e.id !== id)
  } catch {
    // silently fail
  }
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

function proxiedUrl(imageUrl: string) {
  const token = localStorage.getItem('token') ?? ''
  return `${API_BASE}/api/proxy-image?url=${encodeURIComponent(imageUrl)}&token=${encodeURIComponent(token)}`
}

function toDateInput(dateStr?: string | null): string {
  if (!dateStr) return ''
  return dateStr.split('T')?.[0] ?? ''
}

async function openEvent(eventId: number) {
  try {
    const res = await getCalendarEvent(eventId)
    const ev = res.data?.event
    if (!ev) return
    selectedEvent.value = { id: ev.id, type: 'event', title: ev.title, auctionHouse: ev.auctionHouse, startDate: ev.startDate ?? undefined, endDate: ev.endDate ?? undefined, url: ev.url, notes: ev.notes }
    linkedLots.value = res.data?.lots ?? []
    editEvent.value = {
      title: ev.title,
      auctionHouse: ev.auctionHouse ?? '',
      startDate: toDateInput(ev.startDate),
      endDate: toDateInput(ev.endDate),
      url: ev.url ?? '',
      notes: ev.notes ?? ''
    }
  } catch { /* ignore */ }
}

async function handleUpdateEvent() {
  if (!selectedEvent.value) return
  savingEvent.value = true
  try {
    const data: Record<string, unknown> = {
      title: editEvent.value.title.trim(),
      auctionHouse: editEvent.value.auctionHouse.trim(),
      startDate: editEvent.value.startDate ? editEvent.value.startDate + 'T00:00:00Z' : null,
      endDate: editEvent.value.endDate ? editEvent.value.endDate + 'T00:00:00Z' : null,
      url: editEvent.value.url.trim(),
      notes: editEvent.value.notes.trim()
    }
    await updateCalendarEvent(selectedEvent.value.id, data)
    selectedEvent.value = null
    await loadCalendar()
  } finally {
    savingEvent.value = false
  }
}

watch([currentYear, currentMonth], () => loadCalendar())

onMounted(loadCalendar)

const pullContainer = ref<HTMLElement | null>(null)
const { pullDistance, refreshing } = usePullToRefresh(pullContainer, async () => {
  await loadCalendar()
})
</script>

<style scoped>
.container { max-width: 1200px; margin: 0 auto; padding: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
.page-header h1 { font-size: 1.75rem; color: var(--text-primary); }
.btn {display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.5rem 1rem; border-radius: 8px; border: none; cursor: pointer; font-weight: 500; font-size: 0.875rem; }
.btn-primary { background: var(--accent-gold); color: #1e1e1e; }
.btn-secondary { background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border-subtle); }
.btn-danger { background: #dc3545; color: white; }
.loading-state { text-align: center; padding: 2rem; color: var(--text-secondary); }
.empty-state { text-align: center; padding: 3rem; color: var(--text-secondary); }
.empty-state h3 { color: var(--text-primary); margin: 0.75rem 0 0.5rem; }

/* Month Nav */
.month-nav { display: flex; align-items: center; justify-content: center; gap: 1.5rem; margin-bottom: 1.25rem; }
.month-label { font-size: 1.25rem; color: var(--text-primary); margin: 0; min-width: 200px; text-align: center; }

/* Calendar Grid */
.calendar-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px; background: var(--border-subtle); border: 1px solid var(--border-subtle); border-radius: 12px; overflow: hidden; margin-bottom: 2rem; }
.day-header { background: var(--bg-card); padding: 0.5rem; text-align: center; font-size: 0.75rem; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; }
.day-cell { background: var(--bg-card); padding: 0.5rem; min-height: 60px; position: relative; }
.day-cell.other-month { opacity: 0.3; }
.day-cell.is-today .day-number { background: var(--accent-gold); color: #1e1e1e; border-radius: 50%; width: 24px; height: 24px; display: inline-flex; align-items: center; justify-content: center; font-weight: 700; }
.day-number { font-size: 0.8rem; color: var(--text-primary); }
.day-indicators { display: flex; gap: 3px; margin-top: 4px; flex-wrap: wrap; }
.indicator { width: 7px; height: 7px; border-radius: 50%; }
.lot-indicator { background: var(--accent-gold); }
.event-indicator { background: #17a2b8; }

/* Event List */
.event-list-section { margin-top: 1rem; }
.event-list-section > h2 { font-size: 1.25rem; color: var(--text-primary); margin-bottom: 1rem; }
.event-group { margin-bottom: 1.5rem; }
.group-title { font-size: 0.9rem; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.75rem; }
.group-title.lot-accent { color: var(--accent-gold); }
.group-title.event-accent { color: #17a2b8; }

.event-card { background: var(--bg-card); border: 1px solid var(--border-subtle); border-radius: 12px; padding: 1rem; margin-bottom: 0.5rem; }
.event-card-body { display: flex; gap: 1rem; align-items: flex-start; }
.lot-thumb-container { flex-shrink: 0; width: 64px; height: 64px; border-radius: 8px; overflow: hidden; }
.lot-thumb { width: 100%; height: 100%; object-fit: cover; }
.event-info { flex: 1; min-width: 0; }
.event-info h4 { margin: 0 0 0.35rem; color: var(--text-primary); font-size: 0.95rem; }
.event-meta { display: flex; flex-wrap: wrap; gap: 0.75rem; font-size: 0.8rem; color: var(--text-secondary); }
.event-meta span { display: inline-flex; align-items: center; gap: 0.25rem; }
.bid-info { color: var(--accent-gold); }
.estimate-info { color: var(--text-secondary); }
.event-notes { font-size: 0.85rem; color: var(--text-secondary); margin: 0.35rem 0 0; line-height: 1.4; }
.lot-link { display: inline-flex; align-items: center; gap: 0.25rem; font-size: 0.8rem; color: var(--accent-gold); text-decoration: none; margin-top: 0.35rem; }
.lot-link:hover { text-decoration: underline; }
.btn-remove { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0.25rem; flex-shrink: 0; border-radius: 4px; }
.btn-remove:hover { color: #dc3545; }

.event-card-clickable { cursor: pointer; transition: border-color var(--transition-fast); }
.event-card-clickable:hover { border-color: var(--accent-gold-dim); }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.6); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 1rem; }
.modal { width: 90%; max-width: 520px; padding: 1.5rem; max-height: 90vh; overflow-y: auto; }
.modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
.modal-header h2 { margin: 0; color: var(--text-primary); }
.btn-close { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0.25rem; }
.form-group { margin-bottom: 1rem; }
.form-group label { display: block; margin-bottom: 0.35rem; color: var(--text-secondary); font-size: 0.875rem; }
.form-group input,
.form-group textarea,
.form-group select { width: 100%; background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 0.5rem 0.75rem; font-size: 0.875rem; box-sizing: border-box; }
.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.modal-actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1.25rem; }

/* Linked Lots in Event Detail */
.linked-lots-section { margin-top: 1.5rem; padding-top: 1rem; border-top: 1px solid var(--border-subtle); }
.linked-lots-title { font-size: 0.95rem; color: var(--text-primary); margin: 0 0 0.75rem; display: flex; align-items: center; gap: 0.5rem; }
.linked-count { background: var(--accent-gold-glow); color: var(--accent-gold); font-size: 0.75rem; padding: 0.1rem 0.5rem; border-radius: var(--radius-full); font-weight: 600; }

.linked-lots-list { display: flex; flex-direction: column; gap: 0.5rem; }

.linked-lot-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
}

.linked-lot-item .lot-thumb-container { width: 40px; height: 40px; border-radius: 6px; overflow: hidden; flex-shrink: 0; }
.linked-lot-item .lot-thumb { width: 100%; height: 100%; object-fit: cover; }

.linked-lot-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.15rem; }
.linked-lot-title { font-size: 0.85rem; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.linked-lot-meta { font-size: 0.75rem; color: var(--text-secondary); display: flex; align-items: center; gap: 0.35rem; }

.lot-ext-link { color: var(--text-secondary); padding: 0.25rem; flex-shrink: 0; }
.lot-ext-link:hover { color: var(--accent-gold); }

.no-linked-lots { font-size: 0.85rem; color: var(--text-secondary); margin: 0; }

.status-tag { padding: 0.1rem 0.4rem; border-radius: var(--radius-full); font-size: 0.7rem; font-weight: 600; text-transform: uppercase; }
.status-watching { background: rgba(100, 150, 255, 0.2); color: #6496ff; }
.status-bidding { background: var(--accent-gold-glow); color: var(--accent-gold); }
.status-won { background: rgba(74, 222, 128, 0.15); color: #4ade80; }
.status-lost { background: rgba(248, 113, 113, 0.15); color: #f87171; }
.status-passed { background: rgba(120, 120, 120, 0.15); color: #999; }

.btn-secondary { text-decoration: none; }

/* --- Pull to refresh --- */
.pull-indicator {
  position: fixed;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.3);
  z-index: 100;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.2s;
}

.pull-indicator.visible {
  pointer-events: auto;
}

.pull-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
}

.pull-indicator.refreshing .pull-spinner {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.pull-text {
  font-size: 0.75rem;
  color: var(--text-secondary);
  white-space: nowrap;
}
</style>
