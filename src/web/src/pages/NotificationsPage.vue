<template>
  <div class="container">
    <div class="page-header">
      <h1>Notifications</h1>
      <button
        v-if="notifications.length > 0 && hasUnread"
        class="btn btn-secondary btn-sm"
        @click="handleMarkAllRead"
      >
        Mark all read
      </button>
    </div>

    <div v-if="loading && notifications.length === 0" class="loading-state">
      Loading notifications...
    </div>

    <div v-else-if="notifications.length === 0" class="empty-state card">
      <BellOff :size="48" class="empty-icon" />
      <p>No notifications yet</p>
      <p class="empty-desc">
        You will be notified when a wishlist item becomes unavailable or a user you follow adds a new coin.
      </p>
    </div>

    <div v-else class="notification-list">
      <div
        v-for="n in notifications"
        :key="n.id"
        class="notification-item card"
        :class="{ unread: !n.isRead }"
        @click="handleClick(n)"
      >
        <div class="notification-icon">
          <AlertTriangle v-if="n.type === 'wishlist_unavailable'" :size="20" />
          <UserPlus v-else-if="n.type === 'friend_new_coin'" :size="20" />
          <Bell v-else :size="20" />
        </div>
        <div class="notification-body">
          <div class="notification-title">{{ n.title }}</div>
          <div class="notification-message">{{ n.message }}</div>
          <div class="notification-time">{{ formatTime(n.createdAt) }}</div>
        </div>
        <button
          class="notification-dismiss"
          title="Delete"
          @click.stop="handleDelete(n.id)"
        >
          <X :size="16" />
        </button>
      </div>

      <div v-if="hasMore" class="load-more">
        <button class="btn btn-secondary btn-sm" @click="loadMore" :disabled="loading">
          {{ loading ? 'Loading...' : 'Load more' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, BellOff, AlertTriangle, UserPlus, X } from 'lucide-vue-next'
import {
  getNotifications,
  markNotificationRead,
  markAllNotificationsRead,
  deleteNotification,
} from '@/api/client'
import { useNotifications } from '@/composables/useNotifications'
import type { Notification } from '@/types'

const router = useRouter()
const { refresh: refreshBadge } = useNotifications()
const notifications = ref<Notification[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)

const hasUnread = computed(() => notifications.value.some((n) => !n.isRead))
const hasMore = computed(() => notifications.value.length < total.value)

async function fetchNotifications(pageNum: number) {
  loading.value = true
  try {
    const res = await getNotifications(pageNum, limit)
    if (pageNum === 1) {
      notifications.value = res.data.notifications ?? []
    } else {
      notifications.value.push(...(res.data.notifications ?? []))
    }
    total.value = res.data.total
    page.value = pageNum
  } finally {
    loading.value = false
  }
}

function loadMore() {
  fetchNotifications(page.value + 1)
}

async function handleClick(n: Notification) {
  if (!n.isRead) {
    await markNotificationRead(n.id)
    n.isRead = true
    refreshBadge()
  }

  if (n.type === 'wishlist_unavailable' && n.referenceId) {
    router.push(`/coin/${n.referenceId}`)
  } else if (n.type === 'friend_new_coin' && n.referenceId) {
    router.push(`/coin/${n.referenceId}`)
  }
}

async function handleMarkAllRead() {
  await markAllNotificationsRead()
  notifications.value.forEach((n) => (n.isRead = true))
  refreshBadge()
}

async function handleDelete(id: number) {
  await deleteNotification(id)
  notifications.value = notifications.value.filter((n) => n.id !== id)
  total.value = Math.max(0, total.value - 1)
  refreshBadge()
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const mins = Math.floor(diff / 60_000)
  if (mins < 1) return 'Just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return d.toLocaleDateString()
}

onMounted(() => fetchNotifications(1))
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.page-header h1 {
  font-family: 'Cinzel', serif;
  font-size: 1.4rem;
  color: var(--accent-gold);
}

.loading-state {
  text-align: center;
  color: var(--text-secondary);
  padding: 3rem 1rem;
}

.empty-state {
  text-align: center;
  padding: 3rem 1.5rem;
  color: var(--text-secondary);
}

.empty-icon {
  color: var(--text-muted);
  margin-bottom: 1rem;
}

.empty-desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  max-width: 360px;
  margin: 0.5rem auto 0;
}

.notification-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.notification-item {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.85rem 1rem;
  cursor: pointer;
  transition: background var(--transition-fast);
  border-left: 3px solid transparent;
}

.notification-item:hover {
  background: var(--bg-card-hover, rgba(255, 255, 255, 0.03));
}

.notification-item.unread {
  border-left-color: var(--accent-gold);
  background: rgba(212, 175, 55, 0.04);
}

.notification-icon {
  flex-shrink: 0;
  color: var(--text-muted);
  margin-top: 2px;
}

.notification-item.unread .notification-icon {
  color: var(--accent-gold);
}

.notification-body {
  flex: 1;
  min-width: 0;
}

.notification-title {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
  margin-bottom: 0.2rem;
}

.notification-message {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.notification-time {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.35rem;
}

.notification-dismiss {
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
}

.notification-dismiss:hover {
  color: #f87171;
  background: rgba(248, 113, 113, 0.1);
}

.load-more {
  text-align: center;
  padding: 1rem;
}
</style>
