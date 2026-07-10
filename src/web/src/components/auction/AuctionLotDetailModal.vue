<template>
  <div class="fixed inset-0 z-[1000] flex overflow-x-hidden overflow-y-auto bg-overlay-full p-4 [-webkit-overflow-scrolling:touch]" @click.self="$emit('close')">
    <div class="card my-auto w-full min-w-0 max-w-[560px] p-0">
      <div class="flex items-start justify-between gap-4 border-b border-border-subtle px-6 py-5">
        <h2 class="min-w-0 text-[1.1rem] leading-[1.35] [overflow-wrap:anywhere]">{{ lot.title }}</h2>
        <div class="header-actions shrink-0 gap-1">
          <button v-if="!isEditing" class="flex items-center rounded-sm p-[0.35rem] text-text-secondary transition hover:bg-gold-glow hover:text-gold" title="Edit details" @click="startEdit">
            <Pencil :size="16" />
          </button>
          <button class="flex items-center rounded-sm p-1 text-text-secondary transition hover:text-text-primary" @click="$emit('close')"><X :size="18" /></button>
        </div>
      </div>

      <div v-if="proxiedImageUrl" class="flex max-h-[300px] w-full items-center justify-center overflow-hidden bg-surface">
        <img :src="proxiedImageUrl" :alt="lot.title" class="h-[300px] w-full object-contain" />
      </div>

      <div v-if="!isEditing" class="px-6 py-5">
        <div v-if="lot.auctionHouse" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Auction House</span>
          <span class="min-w-0 text-right [overflow-wrap:anywhere]">{{ lot.auctionHouse }}</span>
        </div>
        <div v-if="lot.saleName" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Sale</span>
          <span class="min-w-0 text-right [overflow-wrap:anywhere]">{{ lot.saleName }}</span>
        </div>
        <div v-if="lot.lotNumber" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Lot #</span>
          <span class="min-w-0 text-right [overflow-wrap:anywhere]">{{ lot.lotNumber }}</span>
        </div>
        <div v-if="lot.saleDate" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Sale Date</span>
          <span class="min-w-0 text-right [overflow-wrap:anywhere]">{{ formatDate(lot.saleDate) }}</span>
        </div>
        <div v-if="lot.auctionEndTime" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Ends</span>
          <span class="min-w-0 text-right [overflow-wrap:anywhere]">{{ formatDateTime(lot.auctionEndTime) }}</span>
        </div>
        <div v-if="lot.estimate" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Estimate</span>
          <span class="min-w-0 text-right [overflow-wrap:anywhere]">{{ formatCurrency(lot.estimate, lot.currency) }}</span>
        </div>
        <div v-if="lot.currentBid" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Current Bid</span>
          <span class="min-w-0 text-right font-semibold text-gold [overflow-wrap:anywhere]">{{ formatCurrency(lot.currentBid, lot.currency) }}</span>
        </div>
        <div v-if="lot.maxBid" class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Max Bid</span>
          <span class="min-w-0 text-right font-semibold text-gold/80 [overflow-wrap:anywhere]">{{ formatCurrency(lot.maxBid, lot.currency) }}</span>
        </div>
        <div class="flex min-w-0 items-center justify-between gap-3 border-b border-border-subtle py-2 text-[0.88rem]">
          <span class="text-[0.82rem] text-text-secondary">Status</span>
          <span
            class="rounded-full px-[0.55rem] py-[0.15rem] text-sm font-semibold uppercase"
            :class="{
              'bg-[rgba(100,150,255,0.2)] text-[#6496ff]': lot.status === 'watching',
              'bg-gold-glow text-gold': lot.status === 'bidding',
              'bg-[rgba(74,222,128,0.15)] text-[#4ade80]': lot.status === 'won',
              'bg-[rgba(248,113,113,0.15)] text-[#f87171]': lot.status === 'lost',
              'bg-[rgba(120,120,120,0.15)] text-[#999999]': lot.status === 'passed',
            }"
          >
            {{ lot.status }}
          </span>
        </div>
        <div v-if="lot.description" class="mt-3">
          <span class="text-[0.82rem] text-text-secondary">Description</span>
          <p class="mt-1.5 text-body leading-6 text-text-secondary [overflow-wrap:anywhere]">{{ lot.description }}</p>
        </div>
        <div v-if="lot.notes" class="mt-3">
          <span class="text-[0.82rem] text-text-secondary">Notes</span>
          <p class="mt-1.5 text-body leading-6 text-text-secondary [overflow-wrap:anywhere]">{{ lot.notes }}</p>
        </div>

        <section v-if="canManageAlerts" class="mt-4 grid gap-3 border-t border-border-subtle pt-4">
          <div class="grid gap-2">
            <div class="flex items-center justify-between gap-2">
              <span class="text-[0.82rem] text-text-secondary">Price Alerts</span>
              <span v-if="priceAlerts.length" class="chip-sm">{{ priceAlerts.length }}</span>
            </div>
            <div v-if="priceAlerts.length" class="grid gap-[0.35rem]">
              <div v-for="alert in priceAlerts" :key="alert.id" class="flex items-center justify-between gap-2 rounded-sm border border-border-subtle bg-card-hover p-2">
                <div class="min-w-0">
                  <span class="block text-chip font-medium text-text-primary">{{ alert.direction === 'above' ? 'At or above' : 'At or below' }} {{ formatCurrency(alert.targetPrice, lot.currency) }}</span>
                  <span class="mt-[0.15rem] block text-chip text-text-muted">{{ alert.isTriggered ? `Triggered ${formatOptionalDate(alert.triggeredAt)}` : 'Waiting' }}</span>
                </div>
                <button class="btn btn-ghost btn-xs" :disabled="alertBusy" @click="removeAlert(alert.id)">Delete</button>
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <select v-model="alertForm.direction" class="form-input min-w-[150px] flex-1" aria-label="Price alert direction">
                <option value="above">Above current bid</option>
                <option value="below">Below current bid</option>
              </select>
              <input
                v-model.number="alertForm.targetPrice"
                type="number"
                class="form-input max-w-[120px]"
                min="0"
                step="0.01"
                :placeholder="lot.currentBid ? String(lot.currentBid) : 'Target'"
                aria-label="Target price"
              />
              <button class="btn btn-secondary btn-sm" :disabled="alertBusy || !canCreateAlert" @click="saveAlert">Add Alert</button>
            </div>
          </div>

          <div class="grid gap-2">
            <div class="flex items-center justify-between gap-2">
              <span class="text-[0.82rem] text-text-secondary">Bid Reminders</span>
              <span v-if="bidReminders.length" class="chip-sm">{{ bidReminders.length }}</span>
            </div>
            <div v-if="bidReminders.length" class="grid gap-[0.35rem]">
              <div v-for="reminder in bidReminders" :key="reminder.id" class="flex items-center justify-between gap-2 rounded-sm border border-border-subtle bg-card-hover p-2">
                <div class="min-w-0">
                  <span class="block text-chip font-medium text-text-primary">{{ reminder.minutesBefore }} minutes before close</span>
                  <span class="mt-[0.15rem] block text-chip text-text-muted">{{ reminder.isNotified ? `Notified ${formatOptionalDate(reminder.notifiedAt)}` : 'Waiting' }}</span>
                </div>
                <button class="btn btn-ghost btn-xs" :disabled="reminderBusy" @click="removeReminder(reminder.id)">Delete</button>
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <input
                v-model.number="reminderForm.minutesBefore"
                type="number"
                class="form-input max-w-[120px]"
                min="1"
                step="5"
                aria-label="Reminder minutes before close"
              />
              <button class="btn btn-secondary btn-sm" :disabled="reminderBusy || !canCreateReminder" @click="saveReminder">Add Reminder</button>
            </div>
          </div>
          <p v-if="alertMessage" class="m-0 text-chip" :class="alertError ? 'text-[var(--color-negative)]' : 'text-gold'">{{ alertMessage }}</p>
        </section>
      </div>

      <div v-else class="flex flex-col gap-[0.85rem] px-6 py-5">
        <div class="form-group">
          <label class="text-[0.82rem] text-text-secondary">Title</label>
          <input v-model="editForm.title" type="text" class="form-input w-full text-[0.88rem]" />
        </div>
        <div class="form-group">
          <label class="text-[0.82rem] text-text-secondary">Auction URL</label>
          <input v-model="editForm.numisBidsUrl" type="url" class="form-input w-full text-[0.88rem]" placeholder="https://..." />
        </div>
        <div class="grid gap-[0.85rem] md:grid-cols-2">
          <div class="form-group">
            <label class="text-[0.82rem] text-text-secondary">Auction House</label>
            <input v-model="editForm.auctionHouse" type="text" class="form-input w-full text-[0.88rem]" />
          </div>
          <div class="form-group">
            <label class="text-[0.82rem] text-text-secondary">Sale Name</label>
            <input v-model="editForm.saleName" type="text" class="form-input w-full text-[0.88rem]" />
          </div>
        </div>
        <div class="grid gap-[0.85rem] md:grid-cols-2">
          <div class="form-group">
            <label class="text-[0.82rem] text-text-secondary">Lot #</label>
            <input v-model.number="editForm.lotNumber" type="number" class="form-input w-full text-[0.88rem]" min="0" />
          </div>
          <div class="form-group">
            <label class="text-[0.82rem] text-text-secondary">Estimate</label>
            <input v-model.number="editForm.estimate" type="number" class="form-input w-full text-[0.88rem]" min="0" step="0.01" />
          </div>
        </div>
        <div class="grid gap-[0.85rem] md:grid-cols-2">
          <div class="form-group">
            <label class="text-[0.82rem] text-text-secondary">Sale Date</label>
            <input v-model="editForm.saleDate" type="date" class="form-input w-full text-[0.88rem]" />
          </div>
          <div class="form-group">
            <label class="text-[0.82rem] text-text-secondary">End Date / Time</label>
            <input v-model="editForm.auctionEndTime" type="datetime-local" class="form-input w-full text-[0.88rem]" />
          </div>
        </div>
        <div class="form-group">
          <label class="text-[0.82rem] text-text-secondary">Description</label>
          <textarea v-model="editForm.description" class="form-input w-full resize-y text-[0.88rem] leading-[1.45]" rows="3" />
        </div>
        <div class="form-group">
          <label class="text-[0.82rem] text-text-secondary">Notes</label>
          <textarea
            v-model="editForm.notes"
            class="form-input w-full resize-y text-[0.88rem] leading-[1.45]"
            rows="4"
            placeholder="Personal notes about this auction lot..."
          />
        </div>
        <p v-if="editError" class="m-0 text-[0.82rem] text-[var(--color-negative)]">{{ editError }}</p>
        <div class="mt-2 flex justify-end gap-2">
          <button class="btn btn-secondary" :disabled="editSaving" @click="cancelEdit">Cancel</button>
          <button class="btn btn-primary" :disabled="editSaving" @click="saveEdit">
            {{ editSaving ? 'Saving...' : 'Save Changes' }}
          </button>
        </div>
      </div>

      <div v-if="!isEditing" class="flex flex-col gap-3 border-t border-border-subtle px-6 py-4">
        <div class="flex flex-wrap gap-[0.6rem]">
          <select v-model="newStatus" class="form-input min-w-[120px] max-w-[160px] flex-1">
            <option value="watching">Watching</option>
            <option value="bidding">Bidding</option>
            <option value="won">Won</option>
            <option value="lost">Lost</option>
            <option value="passed">Passed</option>
          </select>
          <button class="btn btn-secondary" @click="changeStatus" :disabled="!hasPendingStatusUpdate">
            Update Status
          </button>
        </div>
        <div v-if="newStatus === 'bidding'" class="flex flex-wrap items-center gap-[0.6rem]">
          <label class="text-[0.82rem] text-text-secondary">Max Bid</label>
          <input
            v-model.number="maxBidInput"
            type="number"
            class="form-input max-w-[140px] flex-1"
            :placeholder="lot.currency || 'USD'"
            min="0"
            step="1"
          />
        </div>
        <div class="flex flex-col gap-1.5">
          <label class="inline-flex items-center gap-[0.35rem] text-[0.82rem] text-text-secondary"><CalendarDays :size="14" /> Calendar Event</label>
          <div class="flex flex-wrap items-center gap-2">
            <select v-model="selectedEventId" class="form-input min-w-[160px] flex-1">
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
        <div class="flex flex-wrap gap-[0.6rem]">
          <SafeExternalLink v-if="externalUrl" :href="externalUrl" class="btn btn-primary" target="_blank" rel="noopener noreferrer">
            <ExternalLink :size="14" /> View on {{ providerLabel }}
          </SafeExternalLink>
          <button v-if="lot.status === 'won'" class="btn btn-primary" @click="convertToCoin">
            <ArrowRightCircle :size="14" /> Add to Collection
          </button>
          <button class="btn btn-danger !border-[rgba(248,113,113,0.4)] !bg-transparent px-[0.9rem] py-2 text-[0.82rem] !text-[#f87171] hover:!border-[rgba(248,113,113,0.6)] hover:!bg-[rgba(248,113,113,0.1)]" @click="removeLot">
            <Trash2 :size="14" /> Remove
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { updateAuctionLotStatus, updateAuctionLot, convertAuctionLotToCoin, deleteAuctionLot, listCalendarEvents, linkAuctionLotEvent, createAlert, deleteAlert, createReminder, deleteReminder } from '@/api/client'
import { useProxiedImage } from '@/composables/useProxiedImage'
import type { AuctionLot, AuctionLotStatus, BidReminder, PriceAlert, PriceAlertDirection } from '@/types'
import { X, ExternalLink, ArrowRightCircle, Trash2, CalendarDays, Pencil } from 'lucide-vue-next'
import { formatCurrency } from '@/utils/format'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

const props = defineProps<{
  lot: AuctionLot
  priceAlerts?: PriceAlert[]
  bidReminders?: BidReminder[]
}>()

const emit = defineEmits<{
  close: []
  updated: []
  alertsUpdated: []
}>()

const router = useRouter()

const newStatus = ref<AuctionLotStatus>(props.lot.status)
const maxBidInput = ref<number | null>(props.lot.maxBid ?? null)
const calendarEvents = ref<Array<{ id: number; title: string; auctionHouse: string; startDate: string | null }>>([])
const selectedEventId = ref<number | string>(props.lot.eventId ?? '')

const lotImageSource = computed(() => props.lot.imageUrl ?? '')
const { proxiedImageUrl } = useProxiedImage(lotImageSource)
const providerLabel = computed(() => props.lot.source === 'cng' ? 'CNG' : 'NumisBids')
const externalUrl = computed(() => props.lot.sourceUrl || props.lot.numisBidsUrl)
const normalizedMaxBidInput = computed(() => typeof maxBidInput.value === 'number' && !Number.isNaN(maxBidInput.value) ? maxBidInput.value : null)
const maxBidChanged = computed(() => newStatus.value === 'bidding' && normalizedMaxBidInput.value !== null && normalizedMaxBidInput.value !== (props.lot.maxBid ?? null))
const hasPendingStatusUpdate = computed(() => newStatus.value !== props.lot.status || maxBidChanged.value)
const priceAlerts = computed(() => props.priceAlerts ?? [])
const bidReminders = computed(() => props.bidReminders ?? [])
const canManageAlerts = computed(() => props.lot.status === 'watching' || props.lot.status === 'bidding')
const alertBusy = ref(false)
const reminderBusy = ref(false)
const alertMessage = ref('')
const alertError = ref(false)
const alertForm = reactive<{ targetPrice: number | null; direction: PriceAlertDirection }>({
  targetPrice: props.lot.currentBid ?? props.lot.maxBid ?? props.lot.estimate ?? null,
  direction: 'above',
})
const reminderForm = reactive<{ minutesBefore: number | null }>({
  minutesBefore: 30,
})
const canCreateAlert = computed(() => typeof alertForm.targetPrice === 'number' && alertForm.targetPrice > 0)
const canCreateReminder = computed(() => typeof reminderForm.minutesBefore === 'number' && reminderForm.minutesBefore > 0)

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

function formatDateTime(dateStr: string) {
  return new Date(dateStr).toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

function formatOptionalDate(dateStr: string | null) {
  if (!dateStr) return ''
  return formatDate(dateStr)
}

function setAlertMessage(message: string, isError = false) {
  alertMessage.value = message
  alertError.value = isError
}

// Edit mode
const isEditing = ref(false)
const editSaving = ref(false)
const editError = ref('')

interface EditForm {
  title: string
  numisBidsUrl: string
  auctionHouse: string
  saleName: string
  lotNumber: number | null
  saleDate: string
  auctionEndTime: string
  description: string
  notes: string
  estimate: number | null
}

const editForm = reactive<EditForm>({
  title: '',
  numisBidsUrl: '',
  auctionHouse: '',
  saleName: '',
  lotNumber: null,
  saleDate: '',
  auctionEndTime: '',
  description: '',
  notes: '',
  estimate: null,
})

function isoToDateInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

function isoToDateTimeLocalInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}T${hh}:${mi}`
}

function startEdit() {
  editError.value = ''
  editForm.title = props.lot.title || ''
  editForm.numisBidsUrl = externalUrl.value || ''
  editForm.auctionHouse = props.lot.auctionHouse || ''
  editForm.saleName = props.lot.saleName || ''
  editForm.lotNumber = props.lot.lotNumber || null
  editForm.saleDate = isoToDateInput(props.lot.saleDate)
  editForm.auctionEndTime = isoToDateTimeLocalInput(props.lot.auctionEndTime)
  editForm.description = props.lot.description || ''
  editForm.notes = props.lot.notes || ''
  editForm.estimate = props.lot.estimate
  isEditing.value = true
}

function cancelEdit() {
  isEditing.value = false
  editError.value = ''
}

async function saveEdit() {
  editError.value = ''
  const title = editForm.title.trim()
  const url = editForm.numisBidsUrl.trim()
  if (!title) {
    editError.value = 'Title is required'
    return
  }
  if (!url) {
    editError.value = 'URL is required'
    return
  }
  if (!/^https?:\/\//i.test(url)) {
    editError.value = 'URL must start with http:// or https://'
    return
  }

  editSaving.value = true
  try {
    await updateAuctionLot(props.lot.id, {
      title,
      numisBidsUrl: url,
      auctionHouse: editForm.auctionHouse.trim(),
      saleName: editForm.saleName.trim(),
      lotNumber: editForm.lotNumber ?? 0,
      saleDate: editForm.saleDate ? new Date(editForm.saleDate).toISOString() : null,
      auctionEndTime: editForm.auctionEndTime ? new Date(editForm.auctionEndTime).toISOString() : null,
      description: editForm.description,
      notes: editForm.notes,
      estimate: editForm.estimate,
    })
    isEditing.value = false
    emit('updated')
  } catch {
    editError.value = 'Failed to save changes'
  } finally {
    editSaving.value = false
  }
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

async function saveAlert() {
  if (!canCreateAlert.value) return
  alertBusy.value = true
  setAlertMessage('')
  try {
    await createAlert({
      auctionLotId: props.lot.id,
      targetPrice: alertForm.targetPrice ?? 0,
      direction: alertForm.direction,
    })
    emit('alertsUpdated')
    setAlertMessage('Price alert saved')
  } catch {
    setAlertMessage('Failed to save price alert', true)
  } finally {
    alertBusy.value = false
  }
}

async function removeAlert(id: number) {
  alertBusy.value = true
  setAlertMessage('')
  try {
    await deleteAlert(id)
    emit('alertsUpdated')
    setAlertMessage('Price alert deleted')
  } catch {
    setAlertMessage('Failed to delete price alert', true)
  } finally {
    alertBusy.value = false
  }
}

async function saveReminder() {
  if (!canCreateReminder.value) return
  reminderBusy.value = true
  setAlertMessage('')
  try {
    await createReminder({
      auctionLotId: props.lot.id,
      minutesBefore: reminderForm.minutesBefore ?? 30,
    })
    emit('alertsUpdated')
    setAlertMessage('Bid reminder saved')
  } catch {
    setAlertMessage('Failed to save bid reminder', true)
  } finally {
    reminderBusy.value = false
  }
}

async function removeReminder(id: number) {
  reminderBusy.value = true
  setAlertMessage('')
  try {
    await deleteReminder(id)
    emit('alertsUpdated')
    setAlertMessage('Bid reminder deleted')
  } catch {
    setAlertMessage('Failed to delete bid reminder', true)
  } finally {
    reminderBusy.value = false
  }
}

async function changeStatus() {
  try {
    const bid = maxBidChanged.value ? normalizedMaxBidInput.value : undefined
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

