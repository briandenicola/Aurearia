<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150 ease-out"
      enter-from-class="opacity-0"
      leave-active-class="transition-opacity duration-100 ease-in"
      leave-to-class="opacity-0"
    >
      <div v-if="dialog.state.value.visible" class="fixed inset-0 z-[2000] flex items-center justify-center bg-[rgba(0,0,0,0.7)] backdrop-blur-[2px]" @click.self="onCancel">
        <div class="w-[90%] max-w-[400px] animate-slide-up rounded-md border border-border-subtle bg-card p-7 shadow-card" role="alertdialog" :aria-label="dialog.state.value.title || 'Dialog'">
          <h3 v-if="dialog.state.value.title" class="mb-2 text-base text-heading">{{ dialog.state.value.title }}</h3>
          <p class="m-0 text-base leading-6 text-text-secondary">{{ dialog.state.value.message }}</p>
          <div class="mt-6 flex justify-end gap-3">
            <button
              v-if="dialog.state.value.type === 'confirm'"
              class="btn btn-secondary"
              @click="onCancel"
            >
              {{ dialog.state.value.cancelLabel }}
            </button>
            <button
              class="btn"
              :class="dialog.state.value.variant === 'danger' ? 'btn-danger' : 'btn-primary'"
              @click="dialog.handleConfirm()"
            >
              {{ dialog.state.value.confirmLabel }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useDialog } from '@/composables/useDialog'

const dialog = useDialog()

function onCancel() {
  dialog.handleCancel()
}
</script>
