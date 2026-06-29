<template>
  <Teleport to="body">
    <div class="toast-stack" aria-live="polite" aria-atomic="true">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="toast-message"
        :class="`toast-${toast.kind}`"
        role="status"
      >
        <span>{{ toast.message }}</span>
        <button class="toast-dismiss" type="button" aria-label="Dismiss notification" @click="removeToast(toast.id)">
          <X :size="14" />
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'

const { toasts, removeToast } = useToast()
</script>

<style scoped>
.toast-stack {
  position: fixed;
  right: 1rem;
  bottom: 1rem;
  z-index: 1200;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-width: min(22rem, calc(100vw - 2rem));
}

.toast-message {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  color: var(--text-primary);
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-card);
  font-size: 0.85rem;
}

.toast-success {
  border-color: var(--accent-gold);
}

.toast-info {
  border-color: var(--border-accent);
}

.toast-error {
  border-color: var(--color-negative);
}

.toast-dismiss {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 0.15rem;
  color: var(--text-secondary);
  background: transparent;
  border: 0;
  cursor: pointer;
}

.toast-dismiss:hover {
  color: var(--text-primary);
}

@media (max-width: 480px) {
  .toast-stack {
    right: 0.75rem;
    bottom: 0.75rem;
    max-width: calc(100vw - 1.5rem);
  }
}
</style>
