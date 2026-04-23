<template>
  <div v-if="user" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal card">
      <h3>Reset Password for {{ user.username }}</h3>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label class="form-label">New Password</label>
          <input v-model="password" type="password" class="form-input" required minlength="6" />
        </div>
        <p v-if="msg" class="msg" :class="{ error }">{{ msg }}</p>
        <div class="modal-actions">
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
import { ref, watch } from 'vue'
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
    setTimeout(() => { emit('close') }, 1200)
  } catch {
    msg.value = 'Failed to reset password'
    error.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: 1rem;
}

.modal {
  width: 100%;
  max-width: 400px;
}

.modal h3 {
  margin-bottom: 1rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: #e74c3c;
}
</style>
