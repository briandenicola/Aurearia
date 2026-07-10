<template>
  <div v-if="user" class="fixed inset-0 bg-[rgba(0,0,0,0.6)] flex items-center justify-center z-[200] p-4" @click.self="$emit('close')">
    <div class="card w-full max-w-[400px]">
      <h3 class="mb-4">Reset Password for {{ user.username }}</h3>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label class="form-label">New Password</label>
          <input v-model="password" type="password" class="form-input" required minlength="6" />
        </div>
        <p
          v-if="msg"
          class="text-body my-2"
          :class="error ? 'text-[#e74c3c]' : 'text-gold'"
        >{{ msg }}</p>
        <div class="flex justify-end gap-2 mt-4">
          <button type="button" class="btn btn-secondary btn-sm" @click="$emit('close')">Cancel</button>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="loading">
            {{ loading ? 'Resetting...' : 'Reset Password' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import type { UserInfo } from '@/types'
import { resetUserPassword } from '@/api/client'

const props = defineProps<{
  user: UserInfo | null
}>()

const emit = defineEmits<{
  close: []
}>()

const password = ref('')
const msg = ref('')
const error = ref(false)
const loading = ref(false)
let closeTimer: ReturnType<typeof setTimeout> | null = null

watch(() => props.user, () => {
  password.value = ''
  msg.value = ''
  error.value = false
})

async function handleSubmit() {
  if (!props.user) return
  loading.value = true
  msg.value = ''
  try {
    await resetUserPassword(props.user.id, password.value)
    msg.value = 'Password reset successfully'
    closeTimer = setTimeout(() => { emit('close') }, 1200)
  } catch {
    msg.value = 'Failed to reset password'
    error.value = true
  } finally {
    loading.value = false
  }
}

onBeforeUnmount(() => {
  if (closeTimer) clearTimeout(closeTimer)
})
</script>

