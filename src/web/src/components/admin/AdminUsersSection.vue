<template>
  <section class="admin-section card">
    <h2>User Management</h2>
    <div v-if="loading" class="loading-overlay"><div class="spinner"></div></div>
    <table v-else class="users-table">
      <thead>
        <tr>
          <th>Username</th>
          <th>Role</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id">
          <td>
            <span class="username">{{ user.username }}</span>
            <span v-if="user.id === currentUserId" class="you-badge">(you)</span>
          </td>
          <td>
            <span class="badge" :class="`badge-${user.role === 'admin' ? 'roman' : 'modern'}`">
              {{ user.role }}
            </span>
          </td>
          <td class="date-cell">{{ formatDate(user.createdAt) }}</td>
          <td>
            <div v-if="user.id !== currentUserId" class="action-btns">
              <button class="btn btn-secondary btn-sm" @click="$emit('reset', user)">
                Reset
              </button>
              <button class="btn btn-danger btn-sm" @click="$emit('delete', user)">
                Delete
              </button>
            </div>
            <span v-else class="text-muted">—</span>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
import type { UserInfo } from '@/types'

defineProps<{
  users: UserInfo[]
  loading: boolean
  currentUserId: number
}>()

defineEmits<{
  reset: [user: UserInfo]
  delete: [user: UserInfo]
}>()

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}
</script>

<style scoped>
.users-table {
  width: 100%;
  border-collapse: collapse;
}

.users-table th,
.users-table td {
  text-align: left;
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.users-table th {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
}

.username {
  font-weight: 500;
}

.you-badge {
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-left: 0.3rem;
}

.date-cell {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.action-btns {
  display: flex;
  gap: 0.4rem;
}

.text-muted {
  color: var(--text-muted);
}

@media (max-width: 640px) {
  .users-table {
    font-size: 0.85rem;
  }
  .action-btns {
    flex-direction: column;
  }
}
</style>
