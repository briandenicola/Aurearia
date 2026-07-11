<template>
  <Transition
    enter-active-class="transition-all duration-300 ease-out"
    enter-from-class="translate-y-full opacity-0"
    leave-active-class="transition-all duration-300 ease-in"
    leave-to-class="translate-y-full opacity-0"
  >
    <div v-if="visible" class="fixed inset-x-0 bottom-0 z-[150] border-t border-border-subtle bg-card px-5 pt-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-[0_-4px_20px_rgba(0,0,0,0.4)]">
      <div class="mx-auto flex max-w-[480px] items-start gap-[0.85rem]">
        <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-md bg-gold-glow text-gold">
          <Download :size="24" />
        </div>
        <div class="min-w-0 flex-1">
          <h4 class="mb-[0.3rem] text-base text-gold">Aurearia - Coin Collection</h4>
          <p v-if="platform === 'ios-safari'" class="m-0 text-[0.82rem] leading-6 text-text-secondary">
            Tap the <strong>Share</strong> button
            <Share :size="14" class="mx-[0.1rem] inline-block align-middle text-gold" />
            then <strong>"Add to Home Screen"</strong>
          </p>
          <p v-else-if="platform === 'ios-edge'" class="m-0 text-[0.82rem] leading-6 text-text-secondary">
            Tap the <strong>menu</strong> button
            <MoreHorizontal :size="14" class="mx-[0.1rem] inline-block align-middle text-gold" />
            then <strong>"Add to Phone"</strong>
          </p>
          <p v-else-if="platform === 'ios-other'" class="m-0 text-[0.82rem] leading-6 text-text-secondary">
            For the best experience, open in <strong>Safari</strong>, tap
            <Share :size="14" class="mx-[0.1rem] inline-block align-middle text-gold" />
            then <strong>"Add to Home Screen"</strong>
          </p>
          <p v-else-if="platform === 'android'" class="m-0 text-[0.82rem] leading-6 text-text-secondary">
            Tap the <strong>menu</strong>
            <MoreVertical :size="14" class="mx-[0.1rem] inline-block align-middle text-gold" />
            then <strong>"Add to Home Screen"</strong> or <strong>"Install App"</strong>
          </p>
          <p v-else class="m-0 text-[0.82rem] leading-6 text-text-secondary">
            Use your browser menu to <strong>"Install"</strong> or <strong>"Add to Home Screen"</strong>
          </p>
        </div>
        <button class="shrink-0 rounded-sm p-1 text-text-secondary transition-colors hover:bg-gold-glow hover:text-text-primary" @click="dismiss" aria-label="Dismiss">
          <X :size="18" />
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Download, X, Share, MoreVertical, MoreHorizontal } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'

const DISMISS_KEY = 'pwa-install-dismissed'

const visible = ref(false)
const platform = ref<'ios-safari' | 'ios-edge' | 'ios-other' | 'android' | 'other'>('other')

function detectPlatform(): typeof platform.value {
  const ua = navigator.userAgent || ''
  const isIOS = /iPad|iPhone|iPod/.test(ua) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
  if (isIOS) {
    if (/EdgiOS|Edg/i.test(ua)) return 'ios-edge'
    if (/CriOS/i.test(ua)) return 'ios-other'
    if (/FxiOS/i.test(ua)) return 'ios-other'
    return 'ios-safari'
  }
  if (/Android/i.test(ua)) return 'android'
  return 'other'
}

function isMobile(): boolean {
  return /Mobi|Android|iPhone|iPad|iPod/i.test(navigator.userAgent)
    || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
}

function isStandalone(): boolean {
  const { isPwa } = usePwa()
  return isPwa
}

function dismiss() {
  visible.value = false
  localStorage.setItem(DISMISS_KEY, 'true')
}

onMounted(() => {
  if (isStandalone()) return
  if (!isMobile()) return
  if (localStorage.getItem(DISMISS_KEY)) return

  platform.value = detectPlatform()
  visible.value = true
})
</script>
