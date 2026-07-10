<template>
  <div ref="pullContainer" class="container" :style="pullDistance > 0 ? `transform: translateY(${pullDistance}px); transition: none` : ''">
    <div
      class="pointer-events-none fixed left-1/2 z-[100] flex -translate-x-1/2 items-center gap-2 rounded-full border border-border-subtle bg-card px-4 py-[0.4rem] opacity-0 shadow-[0_2px_12px_rgba(0,0,0,0.3)] transition-opacity"
      :class="{ 'pointer-events-auto opacity-100': pullDistance > 0 || refreshing }"
      :style="`top: ${-50 + pullDistance * 0.6}px; opacity: ${Math.min(pullDistance / 60, 1)}`"
    >
      <div class="h-[18px] w-[18px] rounded-full border-2 border-border-subtle border-t-gold" :class="{ 'animate-spin': refreshing }" :style="refreshing ? '' : `transform: rotate(${pullDistance * 3}deg)`"></div>
      <span class="whitespace-nowrap text-sm text-text-secondary">{{ refreshing ? 'Refreshing...' : pullDistance >= 60 ? 'Release to refresh' : 'Pull to refresh' }}</span>
    </div>
    <div class="page-header">
      <h1>Auction Calendar</h1>
      <div v-if="isPwa" class="pwa-actions">
        <button class="pwa-icon-btn" @click="showAddEvent = true" title="Add Event">
          <CirclePlus :size="22" />
        </button>
      </div>
      <div v-else class="header-actions">
        <button class="btn btn-primary" @click="showAddEvent = true">
          <Plus :size="16" /> Add Event
        </button>
      </div>
    </div>

    <!-- Month Navigation -->
    <div class="mb-5 flex items-center justify-center gap-6">
      <button class="btn btn-secondary" @click="prevMonth"><ChevronLeft :size="18" /></button>
      <h2 class="m-0 min-w-[200px] text-center text-lg text-text-primary">{{ monthLabel }}</h2>
      <button class="btn btn-secondary" @click="nextMonth"><ChevronRight :size="18" /></button>
    </div>

    <!-- Calendar Grid -->
    <div class="mb-8 grid grid-cols-7 gap-px overflow-hidden rounded-md border border-border-subtle bg-border-subtle">
      <div v-for="day in dayNames" :key="day" class="bg-card px-2 py-2 text-center text-sm font-semibold uppercase tracking-[0.08em] text-text-secondary">{{ day }}</div>
      <div
        v-for="(cell, idx) in calendarCells"
        :key="idx"
        class="relative min-h-[60px] bg-card p-2"
        :class="{ 'opacity-30': !cell.currentMonth }"
      >
        <span class="inline-flex text-chip text-text-primary" :class="cell.isToday ? 'h-6 w-6 items-center justify-center rounded-full bg-gold font-bold text-surface' : ''">{{ cell.day }}</span>
        <div class="mt-1 flex flex-wrap gap-[3px]">
          <span
            v-for="n in Math.min(cell.lots?.length ?? 0, 3)"
            :key="'lot-' + n"
            class="h-[7px] w-[7px] rounded-full bg-gold"
            title="Auction lot"
          ></span>
          <span
            v-for="n in Math.min(cell.events?.length ?? 0, 3)"
            :key="'ev-' + n"
            class="h-[7px] w-[7px] rounded-full bg-modern-cat"
            title="Event"
          ></span>
        </div>
      </div>
    </div>

    <!-- Event List -->
    <div class="mt-4">
      <h2 class="mb-4 text-lg text-text-primary">Events This Month</h2>

      <div v-if="loading" class="py-8 text-center text-text-secondary">Loading calendar...</div>

      <template v-else>
        <!-- Auction Lots -->
        <div v-if="lots.length" class="mb-6">
          <h3 class="mb-3 text-label font-semibold uppercase tracking-[0.08em] text-gold">Auction Lots</h3>
          <div v-for="lot in lots" :key="'lot-' + lot.id" class="mb-2 rounded-md border border-border-subtle bg-card p-4 shadow-[var(--shadow-card)]">
            <div class="flex items-start gap-4">
              <div v-if="getProxiedImageUrl(lot.imageUrl)" class="h-16 w-16 shrink-0 overflow-hidden rounded-sm">
                <img :src="getProxiedImageUrl(lot.imageUrl)" class="h-full w-full object-cover" alt="" />
              </div>
              <div class="min-w-0 flex-1">
                <h4 class="mb-[0.35rem] text-base text-text-primary">{{ lot.title }}</h4>
                <div class="flex flex-wrap gap-3 text-chip text-text-secondary">
                  <span v-if="lot.auctionHouse" class="inline-flex items-center gap-1"><Building :size="13" /> {{ lot.auctionHouse }}</span>
                  <span v-if="lot.saleDate" class="inline-flex items-center gap-1"><CalendarIcon :size="13" /> {{ formatDate(lot.saleDate) }}</span>
                  <span v-if="lot.currentBid" class="text-gold">Current bid: {{ lot.currentBid }}</span>
                  <span v-if="lot.estimate">Est: {{ lot.estimate }}</span>
                </div>
                <SafeExternalLink v-if="auctionLotUrl(lot)" :href="auctionLotUrl(lot) ?? ''" target="_blank" rel="noopener" class="mt-[0.35rem] inline-flex items-center gap-1 text-chip text-gold hover:underline">
                  <ExternalLink :size="13" /> View on {{ auctionProviderLabel(lot) }}
                </SafeExternalLink>
              </div>
            </div>
          </div>
        </div>

        <!-- Manual Events -->
        <div v-if="events.length" class="mb-6">
          <h3 class="mb-3 text-label font-semibold uppercase tracking-[0.08em] text-modern-cat">Events</h3>
          <div v-for="ev in events" :key="'ev-' + ev.id" class="mb-2 cursor-pointer rounded-md border border-border-subtle bg-card p-4 shadow-[var(--shadow-card)] transition-colors hover:border-gold-dim" @click="openEvent(ev.id)">
            <div class="flex items-start gap-4">
              <div class="min-w-0 flex-1">
                <h4 class="mb-[0.35rem] text-base text-text-primary">{{ ev.title }}</h4>
                <div class="flex flex-wrap gap-3 text-chip text-text-secondary">
                  <span v-if="ev.auctionHouse" class="inline-flex items-center gap-1"><Building :size="13" /> {{ ev.auctionHouse }}</span>
                  <span v-if="ev.startDate" class="inline-flex items-center gap-1">
                    <CalendarIcon :size="13" />
                    {{ formatDate(ev.startDate) }}
                    <template v-if="ev.endDate"> - {{ formatDate(ev.endDate) }}</template>
                  </span>
                </div>
                <p v-if="ev.notes" class="mt-[0.35rem] text-body leading-[1.4] text-text-secondary">{{ ev.notes }}</p>
                <SafeExternalLink v-if="ev.url" :href="ev.url" target="_blank" rel="noopener" class="mt-[0.35rem] inline-flex items-center gap-1 text-chip text-gold hover:underline" @click.stop>
                  <ExternalLink :size="13" /> Visit
                </SafeExternalLink>
              </div>
              <button class="shrink-0 rounded-sm p-1 text-text-secondary transition-colors hover:text-error-bg" @click.stop="handleDeleteEvent(ev.id)" title="Delete event">
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
    <div v-if="selectedEvent" class="fixed inset-0 z-[100] flex items-center justify-center bg-overlay px-4" @click.self="selectedEvent = null">
      <div class="max-h-[90vh] w-[90%] max-w-[520px] overflow-y-auto rounded-md border border-border-subtle bg-card p-6 shadow-[var(--shadow-card)]">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="m-0 text-xl text-text-primary">Edit Event</h2>
          <button class="rounded-sm p-1 text-text-secondary transition-colors hover:text-gold" @click="selectedEvent = null"><X :size="18" /></button>
        </div>
        <form @submit.prevent="handleUpdateEvent">
          <div class="form-group">
            <label for="edit-title" class="form-label">Title</label>
            <input id="edit-title" v-model="editEvent.title" class="form-input" type="text" required />
          </div>
          <div class="form-group">
            <label for="edit-house" class="form-label">Auction House</label>
            <input id="edit-house" v-model="editEvent.auctionHouse" class="form-input" type="text" />
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div class="form-group">
              <label for="edit-start" class="form-label">Start Date</label>
              <input id="edit-start" v-model="editEvent.startDate" class="form-input" type="date" />
            </div>
            <div class="form-group">
              <label for="edit-end" class="form-label">End Date</label>
              <input id="edit-end" v-model="editEvent.endDate" class="form-input" type="date" />
            </div>
          </div>
          <div class="form-group">
            <label for="edit-url" class="form-label">URL</label>
            <input id="edit-url" v-model="editEvent.url" class="form-input" type="url" />
          </div>
          <div class="form-group">
            <label for="edit-notes" class="form-label">Notes</label>
            <textarea id="edit-notes" v-model="editEvent.notes" class="form-textarea" rows="3"></textarea>
          </div>
          <div class="mt-5 flex justify-end gap-2">
            <button type="button" class="btn btn-secondary" @click="selectedEvent = null">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="savingEvent">
              {{ savingEvent ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </form>

        <!-- Linked Auction Lots -->
        <div class="mt-6 border-t border-border-subtle pt-4">
          <h3 class="mb-3 flex items-center gap-2 text-base text-text-primary">Linked Auction Lots <span v-if="linkedLots.length" class="rounded-full bg-gold-glow px-2 py-[0.1rem] text-sm font-semibold text-gold">{{ linkedLots.length }}</span></h3>
          <div v-if="linkedLots.length" class="flex flex-col gap-2">
            <div v-for="lot in linkedLots" :key="lot.id" class="flex items-center gap-3 rounded-sm border border-border-subtle bg-surface p-[0.6rem]">
              <div v-if="getProxiedImageUrl(lot.imageUrl)" class="h-10 w-10 shrink-0 overflow-hidden rounded-sm">
                <img :src="getProxiedImageUrl(lot.imageUrl)" class="h-full w-full object-cover" alt="" />
              </div>
              <div class="flex min-w-0 flex-1 flex-col gap-[0.15rem]">
                <span class="truncate text-body text-text-primary">{{ lot.title }}</span>
                <span class="flex items-center gap-[0.35rem] text-sm text-text-secondary">
                  <template v-if="lot.lotNumber">Lot {{ lot.lotNumber }}</template>
                  <template v-if="lot.lotNumber && lot.status"> · </template>
                  <span class="rounded-full bg-gold-glow px-2 py-[0.1rem] text-label font-semibold uppercase" :class="lot.status === 'watching' ? 'text-modern-cat' : lot.status === 'bidding' ? 'text-gold' : lot.status === 'won' ? 'text-greek' : lot.status === 'lost' ? 'text-error-bg' : 'text-text-muted'">{{ lot.status }}</span>
                </span>
              </div>
              <SafeExternalLink v-if="auctionLotUrl(lot)" :href="auctionLotUrl(lot) ?? ''" target="_blank" rel="noopener" class="shrink-0 p-1 text-text-secondary transition-colors hover:text-gold" @click.stop>
                <ExternalLink :size="13" />
              </SafeExternalLink>
            </div>
          </div>
          <p v-else class="m-0 text-body text-text-secondary">No auction lots linked to this event. Link lots from the Auctions page.</p>
        </div>
      </div>
    </div>

    <!-- Add Event Modal -->
    <div v-if="showAddEvent" class="fixed inset-0 z-[100] flex items-center justify-center bg-overlay px-4" @click.self="showAddEvent = false">
      <div class="max-h-[90vh] w-[90%] max-w-[520px] overflow-y-auto rounded-md border border-border-subtle bg-card p-6 shadow-[var(--shadow-card)]">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="m-0 text-xl text-text-primary">Add Event</h2>
          <button class="rounded-sm p-1 text-text-secondary transition-colors hover:text-gold" @click="showAddEvent = false"><X :size="18" /></button>
        </div>
        <form @submit.prevent="handleCreateEvent">
          <div class="form-group">
            <label for="ev-title" class="form-label">Title</label>
            <input id="ev-title" v-model="newEvent.title" class="form-input" type="text" required placeholder="Event title" />
          </div>
          <div class="form-group">
            <label for="ev-house" class="form-label">Auction House (optional)</label>
            <input id="ev-house" v-model="newEvent.auctionHouse" class="form-input" type="text" placeholder="e.g. Heritage Auctions" />
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div class="form-group">
              <label for="ev-start" class="form-label">Start Date</label>
              <input id="ev-start" v-model="newEvent.startDate" class="form-input" type="date" />
            </div>
            <div class="form-group">
              <label for="ev-end" class="form-label">End Date</label>
              <input id="ev-end" v-model="newEvent.endDate" class="form-input" type="date" />
            </div>
          </div>
          <div class="form-group">
            <label for="ev-url" class="form-label">URL (optional)</label>
            <input id="ev-url" v-model="newEvent.url" class="form-input" type="url" placeholder="https://..." />
          </div>
          <div class="form-group">
            <label for="ev-notes" class="form-label">Notes (optional)</label>
            <textarea id="ev-notes" v-model="newEvent.notes" class="form-textarea" rows="3" placeholder="Any additional notes"></textarea>
          </div>
          <div class="mt-5 flex justify-end gap-2">
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
import { ref, computed, onBeforeUnmount, onMounted, watch } from 'vue'
import {
  CirclePlus, Plus, ChevronLeft, ChevronRight, X, Trash2,
  ExternalLink, Building, Calendar as CalendarIcon
} from 'lucide-vue-next'
import { getCalendar, getCalendarEvent, createCalendarEvent, updateCalendarEvent, deleteCalendarEvent, proxyImage } from '@/api/client'
import type { AuctionLot } from '@/types'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { usePwa } from '@/composables/usePwa'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

interface CalendarLot {
  id: number
  type: string
  title: string
  auctionHouse?: string
  status?: string
  currentBid?: string
  estimate?: string
  source?: string
  sourceUrl?: string
  numisBidsUrl?: string
  imageUrl?: string
  saleDate?: string
  auctionEndTime?: string
}

function auctionLotUrl(lot: CalendarLot | AuctionLot): string | null {
  return lot.sourceUrl || lot.numisBidsUrl || null
}

function auctionProviderLabel(lot: CalendarLot | AuctionLot): string {
  return lot.source === 'cng' ? 'CNG' : 'NumisBids'
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
const { isPwa } = usePwa()

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
const proxiedImageBySource = ref<Map<string, string>>(new Map())
const pendingProxyLoads = new Set<string>()
const objectUrls = new Set<string>()

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

async function loadProxiedImage(imageUrl: string) {
  if (!imageUrl || pendingProxyLoads.has(imageUrl)) return
  pendingProxyLoads.add(imageUrl)
  try {
    const res = await proxyImage(imageUrl)
    const blob = res.data
    if (!(blob instanceof Blob) || blob.size === 0) {
      proxiedImageBySource.value.set(imageUrl, '')
      return
    }
    const objectUrl = URL.createObjectURL(blob)
    objectUrls.add(objectUrl)
    proxiedImageBySource.value.set(imageUrl, objectUrl)
  } catch (err) {
    console.warn('Failed to proxy calendar image', err)
    proxiedImageBySource.value.set(imageUrl, '')
  } finally {
    pendingProxyLoads.delete(imageUrl)
  }
}

function getProxiedImageUrl(imageUrl: string | undefined) {
  if (!imageUrl) return ''
  const cached = proxiedImageBySource.value.get(imageUrl)
  if (cached !== undefined) {
    return cached
  }
  proxiedImageBySource.value.set(imageUrl, '')
  void loadProxiedImage(imageUrl)
  return ''
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

onBeforeUnmount(() => {
  for (const objectUrl of objectUrls) {
    URL.revokeObjectURL(objectUrl)
  }
  objectUrls.clear()
  pendingProxyLoads.clear()
  proxiedImageBySource.value.clear()
})

const pullContainer = ref<HTMLElement | null>(null)
const { pullDistance, refreshing } = usePullToRefresh(pullContainer, async () => {
  await loadCalendar()
})
</script>
