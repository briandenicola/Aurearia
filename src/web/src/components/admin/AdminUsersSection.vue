<template>
  <section class="card">
    <h2 class="text-xl font-medium mb-5 pb-3 border-b border-border-subtle">User Management</h2>
    <div v-if="loading" class="loading-overlay"><div class="spinner"></div></div>
    <table v-else class="data-table [&_th]:px-2 [&_th]:py-3 [&_td]:px-2 [&_td]:py-3">
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
            <span class="font-medium">{{ user.username }}</span>
            <span v-if="user.id === currentUserId" class="text-label text-text-muted ml-[0.3rem]">(you)</span>
          </td>
          <td>
            <span class="badge" :class="`badge-${user.role === 'admin' ? 'roman' : 'modern'}`">
              {{ user.role }}
            </span>
          </td>
          <td class="px-2 py-3 border-b border-border-subtle text-body text-text-secondary">{{ formatDate(user.createdAt) }}</td>
          <td>
            <div v-if="user.id !== currentUserId" class="flex gap-[0.4rem]">
              <button class="btn btn-secondary btn-sm" @click="$emit('edit', user)">Edit</button>
            </div>
            <span v-else class="text-text-muted">—</span>
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
  edit: [user: UserInfo]
}>()

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}
</script>
