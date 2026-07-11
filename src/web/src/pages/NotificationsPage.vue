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

    <div
      v-if="loading && notifications.length === 0"
      class="py-12 text-center text-text-secondary"
    >
      Loading notifications...
    </div>

    <div
      v-else-if="notifications.length === 0"
      class="empty-state card !px-6 !py-12 text-text-secondary"
    >
      <BellOff :size="48" class="mb-4 text-text-muted" />
      <p>No notifications yet</p>
      <p class="mx-auto mt-2 max-w-[360px] text-body text-text-muted">
        You will be notified about follower requests, wishlist changes, and new coins from users you follow.
      </p>
    </div>

    <div v-else class="flex flex-col gap-2">
      <div
        v-for="n in notifications"
        :key="n.id"
        class="card !p-0 flex cursor-pointer items-start gap-3 border-l-[3px] border-l-transparent !px-4 !py-[0.85rem] transition-colors hover:bg-card-hover"
        :class="!n.isRead ? 'border-l-gold bg-[rgba(201,168,76,0.04)]' : ''"
        @click="handleClick(n)"
      >
        <div
          class="mt-[2px] shrink-0"
          :class="!n.isRead ? 'text-gold' : 'text-text-muted'"
        >
          <AlertTriangle v-if="n.type === 'wishlist_unavailable'" :size="20" />
          <UserPlus v-else-if="n.type === 'friend_new_coin' || n.type === 'follow_request'" :size="20" />
          <Sparkles v-else-if="n.type === 'coin_of_day'" :size="20" />
          <Key v-else-if="n.type === 'api_key_rotation_required'" :size="20" />
          <FolderOpen v-else-if="n.type === 'set_milestone'" :size="20" />
          <Bell v-else :size="20" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="mb-[0.2rem] text-base font-semibold text-text-primary">{{ n.title }}</div>
          <div class="text-body leading-[1.4] text-text-secondary">{{ n.message }}</div>
          <div class="mt-[0.35rem] text-sm text-text-muted">{{ formatTime(n.createdAt) }}</div>
        </div>
        <button
          class="shrink-0 rounded-sm p-1 text-text-muted transition-colors hover:bg-[rgba(248,113,113,0.1)] hover:text-[rgb(248,113,113)]"
          title="Delete"
          @click.stop="handleDelete(n.id)"
        >
          <X :size="16" />
        </button>
      </div>

      <div v-if="hasMore" class="py-4 text-center">
        <button class="btn btn-secondary btn-sm" @click="loadMore" :disabled="loading">
          {{ loading ? 'Loading...' : 'Load more' }}
        </button>
      </div>
    </div>

    <FeaturedCoinModal
      v-if="featuredCoinModalId !== null"
      :featured-coin-id="featuredCoinModalId"
      @close="featuredCoinModalId = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, BellOff, AlertTriangle, UserPlus, Sparkles, Key, X, FolderOpen } from 'lucide-vue-next'
import {
  getNotifications,
  markNotificationRead,
  markAllNotificationsRead,
  deleteNotification,
} from '@/api/client'
import { useNotifications } from '@/composables/useNotifications'
import FeaturedCoinModal from '@/components/coin/FeaturedCoinModal.vue'
import type { Notification } from '@/types'

const router = useRouter()
const { refresh: refreshBadge } = useNotifications()
const notifications = ref<Notification[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const featuredCoinModalId = ref<number | null>(null)

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

  if (n.type === 'coin_of_day' && n.referenceId) {
    featuredCoinModalId.value = n.referenceId
    return
  }
  if (n.type === 'follow_request') {
    router.push('/followers')
    return
  }
  if (n.type === 'wishlist_unavailable' && n.referenceId) {
    router.push(`/coin/${n.referenceId}`)
  } else if (n.type === 'friend_new_coin' && n.referenceId) {
    router.push(`/coin/${n.referenceId}`)
  } else if (n.type === 'api_key_rotation_required') {
    router.push('/settings')
  } else if (n.referenceUrl && n.referenceUrl.startsWith('/')) {
    router.push(n.referenceUrl)
  } else if (n.type === 'set_milestone' && n.referenceId) {
    router.push(`/sets/${n.referenceId}`)
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
