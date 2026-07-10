<template>
  <div v-if="user" class="fixed inset-0 z-[200] flex items-center justify-center bg-[rgba(0,0,0,0.6)] p-4" @click.self="$emit('close')">
    <div class="card w-full max-w-[520px] p-6">
      <h3 class="mb-4 text-lg font-medium text-heading">Edit {{ user.username }}</h3>

      <div class="mb-4">
        <label class="form-label">Role</label>
        <div class="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto] md:items-center">
          <select v-model="selectedRole" class="form-input" :disabled="savingRole || isCurrentUser">
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
          <button class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="savingRole || isCurrentUser" @click="updateRole">
            {{ savingRole ? 'Saving...' : 'Update Role' }}
          </button>
        </div>
        <p v-if="isCurrentUser" class="mt-2 text-body text-text-muted">You cannot change your own role.</p>
        <p v-if="roleMsg" class="mt-2 text-body text-gold" :class="{ 'text-[var(--cat-byzantine)]': roleError }">{{ roleMsg }}</p>
      </div>

      <div class="mb-4">
        <label class="form-label">Reset Password</label>
        <div class="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto] md:items-center">
          <input
            v-model="password"
            type="password"
            class="form-input"
            placeholder="New password"
            minlength="6"
          />
          <button class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="resettingPassword" @click="resetPassword">
            {{ resettingPassword ? 'Resetting...' : 'Reset Password' }}
          </button>
        </div>
        <p v-if="passwordMsg" class="mt-2 text-body text-gold" :class="{ 'text-[var(--cat-byzantine)]': passwordError }">{{ passwordMsg }}</p>
      </div>

      <div class="mb-4">
        <label class="form-label">Delete User</label>
        <button class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="deletingUser || isCurrentUser" @click="deleteTargetUser">
          {{ deletingUser ? 'Deleting...' : 'Delete User' }}
        </button>
        <p v-if="isCurrentUser" class="mt-2 text-body text-text-muted">You cannot delete your own account.</p>
        <p v-if="deleteMsg" class="mt-2 text-body text-gold" :class="{ 'text-[var(--cat-byzantine)]': deleteError }">{{ deleteMsg }}</p>
      </div>

      <div class="flex justify-end">
        <button type="button" class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="$emit('close')">Done</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { UserInfo } from '@/types'
import { deleteUser, resetUserPassword, updateUserRole } from '@/api/client'
import { useDialog } from '@/composables/useDialog'

const props = defineProps<{
  user: UserInfo | null
  currentUserId: number
}>()

const emit = defineEmits<{
  close: []
  'role-updated': [payload: { userId: number; role: UserInfo['role'] }]
  deleted: [userId: number]
}>()

const { showConfirm } = useDialog()

const selectedRole = ref<UserInfo['role']>('user')
const password = ref('')

const savingRole = ref(false)
const resettingPassword = ref(false)
const deletingUser = ref(false)

const roleMsg = ref('')
const roleError = ref(false)
const passwordMsg = ref('')
const passwordError = ref(false)
const deleteMsg = ref('')
const deleteError = ref(false)

const isCurrentUser = computed(() => (props.user?.id ?? 0) === props.currentUserId)

watch(() => props.user, (user) => {
  selectedRole.value = user?.role ?? 'user'
  password.value = ''
  roleMsg.value = ''
  roleError.value = false
  passwordMsg.value = ''
  passwordError.value = false
  deleteMsg.value = ''
  deleteError.value = false
}, { immediate: true })

async function updateRole() {
  if (!props.user || isCurrentUser.value) return
  savingRole.value = true
  roleMsg.value = ''
  roleError.value = false
  try {
    await updateUserRole(props.user.id, selectedRole.value)
    roleMsg.value = 'Role updated successfully'
    emit('role-updated', { userId: props.user.id, role: selectedRole.value })
  } catch {
    roleError.value = true
    roleMsg.value = 'Failed to update role'
  } finally {
    savingRole.value = false
  }
}

async function resetPassword() {
  if (!props.user) return
  if (!password.value || password.value.length < 6) {
    passwordError.value = true
    passwordMsg.value = 'Password must be at least 6 characters'
    return
  }
  resettingPassword.value = true
  passwordMsg.value = ''
  passwordError.value = false
  try {
    await resetUserPassword(props.user.id, password.value)
    passwordMsg.value = 'Password reset successfully'
    password.value = ''
  } catch {
    passwordError.value = true
    passwordMsg.value = 'Failed to reset password'
  } finally {
    resettingPassword.value = false
  }
}

async function deleteTargetUser() {
  if (!props.user || isCurrentUser.value) return
  const confirmed = await showConfirm(
    `Delete user "${props.user.username}" and all their data? This cannot be undone.`,
    { title: 'Delete User', variant: 'danger' }
  )
  if (!confirmed) return

  deletingUser.value = true
  deleteMsg.value = ''
  deleteError.value = false
  try {
    await deleteUser(props.user.id)
    emit('deleted', props.user.id)
  } catch {
    deleteError.value = true
    deleteMsg.value = 'Failed to delete user'
  } finally {
    deletingUser.value = false
  }
}
</script>

