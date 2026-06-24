<template>
  <Transition name="install-slide">
    <div v-if="visible" class="install-prompt">
      <div class="install-content">
        <div class="install-icon">
          <Download :size="24" />
        </div>
        <div class="install-text">
          <h4>Aurearia - Coin Collection</h4>
          <p v-if="platform === 'ios-safari'">
            Tap the <strong>Share</strong> button
            <Share :size="14" class="inline-icon" />
            then <strong>"Add to Home Screen"</strong>
          </p>
          <p v-else-if="platform === 'ios-edge'">
            Tap the <strong>menu</strong> button
            <MoreHorizontal :size="14" class="inline-icon" />
            then <strong>"Add to Phone"</strong>
          </p>
          <p v-else-if="platform === 'ios-other'">
            For the best experience, open in <strong>Safari</strong>, tap
            <Share :size="14" class="inline-icon" />
            then <strong>"Add to Home Screen"</strong>
          </p>
          <p v-else-if="platform === 'android'">
            Tap the <strong>menu</strong>
            <MoreVertical :size="14" class="inline-icon" />
            then <strong>"Add to Home Screen"</strong> or <strong>"Install App"</strong>
          </p>
          <p v-else>
            Use your browser menu to <strong>"Install"</strong> or <strong>"Add to Home Screen"</strong>
          </p>
        </div>
        <button class="install-dismiss" @click="dismiss" aria-label="Dismiss">
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

<style scoped>
.install-prompt {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 150;
  background: var(--bg-card);
  border-top: 1px solid var(--border-subtle);
  box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.4);
  padding: 1rem 1.25rem;
  padding-bottom: max(1rem, env(safe-area-inset-bottom));
}

.install-content {
  display: flex;
  align-items: flex-start;
  gap: 0.85rem;
  max-width: 480px;
  margin: 0 auto;
}

.install-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: var(--accent-gold-glow);
  color: var(--accent-gold);
  display: flex;
  align-items: center;
  justify-content: center;
}

.install-text {
  flex: 1;
  min-width: 0;
}

.install-text h4 {
  font-family: 'Cinzel', serif;
  font-size: 0.95rem;
  color: var(--accent-gold);
  margin: 0 0 0.3rem;
}

.install-text p {
  font-size: 0.82rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}

.inline-icon {
  display: inline-block;
  vertical-align: middle;
  margin: 0 0.1rem;
  color: var(--accent-gold);
}

.install-dismiss {
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
}

.install-dismiss:hover {
  color: var(--text-primary);
  background: var(--accent-gold-glow);
}

/* Slide-up transition */
.install-slide-enter-active,
.install-slide-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.install-slide-enter-from,
.install-slide-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
