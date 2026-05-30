<template>
  <div v-if="user" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal card">
      <h3>Edit {{ user.username }}</h3>

      <div class="form-group">
        <label class="form-label">Role</label>
        <div class="inline-actions">
          <select v-model="selectedRole" class="form-input" :disabled="savingRole || isCurrentUser">
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
          <button class="btn btn-secondary btn-sm" :disabled="savingRole || isCurrentUser" @click="updateRole">
            {{ savingRole ? 'Saving...' : 'Update Role' }}
          </button>
        </div>
        <p v-if="isCurrentUser" class="text-muted">You cannot change your own role.</p>
        <p v-if="roleMsg" class="msg" :class="{ error: roleError }">{{ roleMsg }}</p>
      </div>

      <div class="form-group">
        <label class="form-label">Reset Password</label>
        <div class="inline-actions">
          <input
            v-model="password"
            type="password"
            class="form-input"
            placeholder="New password"
            minlength="6"
          />
          <button class="btn btn-secondary btn-sm" :disabled="resettingPassword" @click="resetPassword">
            {{ resettingPassword ? 'Resetting...' : 'Reset Password' }}
          </button>
        </div>
        <p v-if="passwordMsg" class="msg" :class="{ error: passwordError }">{{ passwordMsg }}</p>
      </div>

      <div class="form-group">
        <label class="form-label">Delete User</label>
        <button class="btn btn-danger btn-sm" :disabled="deletingUser || isCurrentUser" @click="deleteTargetUser">
          {{ deletingUser ? 'Deleting...' : 'Delete User' }}
        </button>
        <p v-if="isCurrentUser" class="text-muted">You cannot delete your own account.</p>
        <p v-if="deleteMsg" class="msg" :class="{ error: deleteError }">{{ deleteMsg }}</p>
      </div>

      <div class="modal-actions">
        <button type="button" class="btn btn-primary btn-sm" @click="$emit('close')">Done</button>
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
  max-width: 520px;
}

.modal h3 {
  margin: 0 0 1rem 0;
}

.form-group {
  margin-bottom: 1rem;
}

.inline-actions {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.5rem;
  align-items: center;
}

.msg {
  margin: 0.5rem 0 0 0;
  color: var(--accent-gold);
}

.msg.error {
  color: var(--cat-byzantine);
}

.text-muted {
  margin: 0.5rem 0 0 0;
  color: var(--text-muted);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 640px) {
  .inline-actions {
    grid-template-columns: 1fr;
  }
}
</style>
