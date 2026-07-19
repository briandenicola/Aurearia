<template>
  <Transition
    enter-active-class="transition-all duration-300 ease-out"
    enter-from-class="translate-y-full opacity-0"
    leave-active-class="transition-all duration-300 ease-in"
    leave-to-class="translate-y-full opacity-0"
  >
    <div v-if="visible" class="fixed inset-x-0 bottom-0 z-[155] border-t border-border-subtle bg-card px-5 pt-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-[0_-4px_20px_rgba(0,0,0,0.4)]">
      <div class="mx-auto flex max-w-[480px] items-center gap-[0.85rem]">
        <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-md bg-gold-glow text-gold">
          <RefreshCw :size="22" />
        </div>
        <div class="min-w-0 flex-1">
          <h4 class="mb-[0.15rem] text-base text-gold">Update available</h4>
          <p class="m-0 text-[0.82rem] leading-6 text-text-secondary">A new version of Aurearia is ready. Refresh to update.</p>
        </div>
        <button class="btn btn-primary shrink-0" type="button" @click="refresh">Refresh</button>
        <button class="shrink-0 rounded-sm p-1 text-text-secondary transition-colors hover:bg-gold-glow hover:text-text-primary" type="button" @click="dismiss" aria-label="Dismiss">
          <X :size="18" />
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { RefreshCw, X } from 'lucide-vue-next'
import { usePwaUpdate } from '@/composables/usePwaUpdate'

const { updateAvailable, refresh } = usePwaUpdate()
const dismissed = ref(false)
const visible = computed(() => updateAvailable.value && !dismissed.value)

function dismiss() {
  dismissed.value = true
}
</script>
