<template>
  <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 max-md:mb-4">
    <button class="btn btn-ghost btn-xs justify-self-start whitespace-nowrap" @click="router.push(backTarget)">
      <ArrowLeft :size="14" />
      Back to Gallery
    </button>
    <div class="flex min-w-0 items-center justify-end gap-[0.45rem]">
      <AppIconButton
        v-if="!isSold"
        :disabled="pinBusy"
        :title="pinBusy ? 'Updating coin Quick Access pin' : pinLabel"
        :aria-label="pinBusy ? 'Updating coin Quick Access pin' : pinLabel"
        :aria-busy="pinBusy"
        :active="coinPinned"
        :aria-pressed="coinPinned"
        @click="togglePin"
      >
        <PinOff v-if="coinPinned" :size="24" />
        <Pin v-else :size="24" />
      </AppIconButton>
      <AppIconButton
        v-if="showReminderAction"
        :title="reminderActive ? 'Edit Reminder' : 'Set Reminder'"
        :aria-label="reminderActive ? 'Edit Reminder' : 'Set Reminder'"
        :active="reminderActive"
        @click="emit('reminder')"
      >
        <BellRing :size="24" />
      </AppIconButton>
      <AppIconButton
        :disabled="sharing"
        :title="sharing ? 'Sharing...' : 'Share'"
        :aria-label="sharing ? 'Sharing...' : 'Share'"
        @click="$emit('share')"
      >
        <Share2 :size="24" />
      </AppIconButton>
      <AppIconButton title="Edit" aria-label="Edit" @click="$emit('edit')">
        <Pencil :size="24" />
      </AppIconButton>
      <AppIconButton title="Delete" aria-label="Delete" @click="$emit('delete')">
        <Trash2 :size="24" />
      </AppIconButton>
      <CoinDetailOverflowMenu
        :coin-id="coinId"
        :is-wishlist="isWishlist"
        :is-sold="isSold"
        :duplicating="duplicating"
        @sell="emit('sell')"
        @duplicate="emit('duplicate')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { computed } from 'vue'
import { ArrowLeft, BellRing, Pencil, Pin, PinOff, Share2, Trash2 } from 'lucide-vue-next'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import CoinDetailOverflowMenu from '@/components/coin/CoinDetailOverflowMenu.vue'
import { useQuickAccess } from '@/composables/useQuickAccess'
import { useToast } from '@/composables/useToast'

const props = withDefaults(defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
  sharing?: boolean
  duplicating?: boolean
  showReminderAction?: boolean
  reminderActive?: boolean
}>(), {
  sharing: false,
  duplicating: false,
  showReminderAction: false,
  reminderActive: false,
})

const emit = defineEmits<{
  share: []
  sell: []
  duplicate: []
  edit: []
  delete: []
  reminder: []
}>()

const router = useRouter()
const backTarget = computed(() => props.isWishlist ? '/wishlist' : '/')
const { error, pin, unpin, isPinned, isBusy } = useQuickAccess()
const { showToast } = useToast()
const pinned = isPinned('coin', props.coinId)
const busy = isBusy('coin', props.coinId)
const coinPinned = computed(() => pinned.value)
const pinBusy = computed(() => busy.value)
const pinLabel = computed(() => coinPinned.value ? 'Unpin coin from Quick Access' : 'Pin coin to Quick Access')

async function togglePin() {
  if (pinBusy.value) return
  try {
    if (coinPinned.value) await unpin('coin', props.coinId)
    else await pin('coin', props.coinId)
  } catch {
    showToast(error.value || 'Unable to update Quick Access.', 'error')
  }
}
</script>
