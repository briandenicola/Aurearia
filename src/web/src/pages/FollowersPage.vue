<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container">
      <div class="page-header">
        <h1 class="flex-1">Followers</h1>
        <div v-if="isPwa" class="pwa-actions">
          <button class="pwa-icon-btn" @click="showSearchModal = true" title="Find Users">
            <UserPlus :size="22" />
          </button>
        </div>
        <div v-else class="header-actions">
          <button class="btn btn-primary" @click="showSearchModal = true">
            <UserPlus :size="16" /> Add
          </button>
        </div>
      </div>

      <div class="mx-auto flex max-w-[900px] flex-col gap-6 overflow-x-hidden">
        <!-- Tab Nav -->
        <div class="flex gap-1 rounded-md border border-border-subtle bg-card p-[0.3rem]">
          <button
            class="flex flex-1 items-center justify-center gap-1.5 rounded-sm px-4 py-2.5 text-body font-medium text-text-secondary transition-[color,background-color] hover:text-text-primary"
            :class="activeTab === 'following' ? 'bg-[var(--accent-gold-dim)] text-gold' : ''"
            @click="activeTab = 'following'"
          >
            <UserPlus :size="16" /> Following
            <span
              v-if="following.length"
              class="min-w-[1.3rem] rounded-full bg-border-subtle px-[0.45rem] py-[0.1rem] text-center text-label text-text-secondary"
              :class="activeTab === 'following' ? 'bg-gold text-surface' : ''"
            >
              {{ following.length }}
            </span>
          </button>
          <button
            class="flex flex-1 items-center justify-center gap-1.5 rounded-sm px-4 py-2.5 text-body font-medium text-text-secondary transition-[color,background-color] hover:text-text-primary"
            :class="activeTab === 'followers' ? 'bg-[var(--accent-gold-dim)] text-gold' : ''"
            @click="activeTab = 'followers'"
          >
            <Users :size="16" /> Followers
            <span
              v-if="followers.length"
              class="min-w-[1.3rem] rounded-full bg-border-subtle px-[0.45rem] py-[0.1rem] text-center text-label text-text-secondary"
              :class="activeTab === 'followers' ? 'bg-gold text-surface' : ''"
            >
              {{ followers.length }}
            </span>
          </button>
        </div>

        <!-- Loading -->
        <div v-if="loading" class="flex flex-col items-center gap-3 px-4 py-12 text-text-secondary">
          <div class="spinner" />
          <p>Loading...</p>
        </div>

        <!-- Following Tab -->
        <div v-else-if="activeTab === 'following'">
          <div v-if="following.length === 0" class="empty-state flex flex-col items-center gap-2 !px-4 !py-12">
            <Users :size="48" />
            <h3 class="m-0 text-base text-text-primary">Not following anyone yet</h3>
            <p class="m-0 text-body text-text-muted">Search for users to follow and see their collections.</p>
            <button class="btn btn-primary btn-sm mt-2" @click="showSearchModal = true">
              <UserPlus :size="16" /> Find Users
            </button>
          </div>
          <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div v-for="user in following" :key="user.id" class="card flex flex-col gap-3 overflow-hidden !p-4">
              <div class="flex items-center gap-3">
                <AuthenticatedImage
                  :media-path="user.avatarPath ? user.avatarPath : '/coin-logo.jpg'"
                  :alt="user.username"
                  class="h-12 w-12 shrink-0 rounded-full border-2 border-border-subtle object-cover"
                />
                <div class="min-w-0 flex-1">
                  <span class="block truncate text-[0.95rem] font-semibold text-text-primary">{{ user.username }}</span>
                  <p v-if="user.bio" class="mt-[0.2rem] truncate text-chip leading-[1.3] text-text-muted">{{ truncate(user.bio, 80) }}</p>
                  <div class="mt-[0.3rem]">
                    <span
                      v-if="user.isPublic && user.coinCount > 0"
                      class="inline-flex rounded-full bg-[var(--accent-gold-dim)] px-2 py-[0.15rem] text-label font-medium text-gold"
                    >
                      {{ user.coinCount }} coins
                    </span>
                  </div>
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <router-link
                  :to="`/followers/${user.username}/gallery`"
                  class="btn btn-secondary btn-sm"
                >
                  <Eye :size="14" /> View Collection
                </router-link>
                <button
                  class="btn btn-danger btn-sm"
                  :disabled="actionLoading === user.id"
                  @click="handleUnfollow(user)"
                >
                  Unfollow
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Followers Tab -->
        <div v-else-if="activeTab === 'followers'">
          <div v-if="followers.length === 0" class="empty-state flex flex-col items-center gap-2 !px-4 !py-12">
            <Users :size="48" />
            <h3 class="m-0 text-base text-text-primary">No followers yet</h3>
            <p class="m-0 text-body text-text-muted">When other users follow you, they'll appear here.</p>
          </div>
          <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div v-for="user in followers" :key="user.id" class="card flex flex-col gap-3 overflow-hidden !p-4">
              <div class="flex items-center gap-3">
                <AuthenticatedImage
                  :media-path="user.avatarPath ? user.avatarPath : '/coin-logo.jpg'"
                  :alt="user.username"
                  class="h-12 w-12 shrink-0 rounded-full border-2 border-border-subtle object-cover"
                />
                <div class="min-w-0 flex-1">
                  <span class="block truncate text-[0.95rem] font-semibold text-text-primary">{{ user.username }}</span>
                  <p v-if="user.bio" class="mt-[0.2rem] truncate text-chip leading-[1.3] text-text-muted">{{ truncate(user.bio, 80) }}</p>
                  <span
                    v-if="user.status === 'pending'"
                    class="mt-[0.3rem] inline-flex rounded-full bg-[var(--accent-gold-glow)] px-2 py-[0.15rem] text-label font-medium text-gold"
                  >
                    Pending
                  </span>
                  <span
                    v-else-if="user.status === 'accepted'"
                    class="mt-[0.3rem] inline-flex rounded-full bg-[var(--accent-gold-glow)] px-2 py-[0.15rem] text-label font-medium text-greek"
                  >
                    Accepted
                  </span>
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <button
                  v-if="user.status === 'pending'"
                  class="btn btn-primary btn-sm"
                  :disabled="actionLoading === user.id"
                  @click="handleAccept(user)"
                >
                  <Check :size="14" /> Accept
                </button>
                <button
                  class="btn btn-danger btn-sm"
                  :disabled="actionLoading === user.id"
                  @click="handleBlock(user)"
                >
                  <ShieldOff :size="14" /> Block
                </button>
              </div>
            </div>
          </div>
      </div>
    </div>

    </div>

    <!-- Search Modal -->
    <Teleport to="body">
      <div
        v-if="showSearchModal"
        class="fixed inset-0 z-[1000] flex justify-center overflow-y-auto bg-overlay-full px-4 py-[5vh] md:items-start"
        @click.self="closeSearchModal"
      >
        <div class="card !p-0 flex max-h-[80vh] w-full max-w-[520px] flex-col">
          <div class="flex items-center justify-between border-b border-border-subtle px-5 py-4">
            <h2 class="m-0 flex items-center gap-2 text-base"><Search :size="20" /> Find Users</h2>
            <button class="rounded-sm p-1 text-text-secondary transition-colors hover:text-text-primary" @click="closeSearchModal">
              <X :size="20" />
            </button>
          </div>
          <div class="flex flex-col gap-4 overflow-y-auto px-5 py-4">
            <p class="m-0 text-chip text-text-muted">Only users with public profiles appear in search results.</p>
            <div class="relative">
              <Search :size="16" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
              <input
                ref="searchInputRef"
                v-model="searchQuery"
                type="text"
                class="form-input w-full pl-9"
                placeholder="Search by username..."
                @input="onSearchInput"
              />
            </div>
            <div v-if="searchLoading" class="flex flex-col items-center gap-3 px-4 py-8 text-text-secondary">
              <div class="spinner" />
            </div>
            <div v-else-if="searchResults.length > 0" class="flex flex-col gap-2">
              <div
                v-for="user in searchResults"
                :key="user.id"
                class="flex items-center justify-between gap-3 rounded-md border border-border-subtle bg-card p-3"
              >
                <div class="flex min-w-0 flex-1 items-center gap-3">
                  <AuthenticatedImage
                    :media-path="user.avatarPath ? user.avatarPath : '/coin-logo.jpg'"
                    :alt="user.username"
                    class="h-12 w-12 shrink-0 rounded-full border-2 border-border-subtle object-cover"
                  />
                  <div class="min-w-0 flex-1">
                    <span class="block truncate text-[0.95rem] font-semibold text-text-primary">{{ user.username }}</span>
                    <p v-if="user.bio" class="mt-[0.2rem] truncate text-chip leading-[1.3] text-text-muted">{{ truncate(user.bio, 60) }}</p>
                  </div>
                </div>
                <div class="shrink-0">
                  <span
                    v-if="user.followStatus === 'pending'"
                    class="inline-flex rounded-full bg-[var(--accent-gold-glow)] px-2 py-[0.15rem] text-label font-medium text-gold"
                  >
                    Pending
                  </span>
                  <span
                    v-else-if="user.followStatus === 'accepted'"
                    class="inline-flex rounded-sm bg-[var(--accent-gold-dim)] px-[0.6rem] py-[0.3rem] text-sm font-medium text-gold"
                  >
                    Following
                  </span>
                  <span
                    v-else-if="user.followStatus === 'blocked'"
                    class="inline-flex rounded-full bg-[var(--accent-gold-glow)] px-2 py-[0.15rem] text-label font-medium text-[var(--error-bg)]"
                  >
                    Blocked
                  </span>
                  <button
                    v-else
                    class="btn btn-primary btn-sm"
                    :disabled="actionLoading === user.id"
                    @click="handleFollow(user)"
                  >
                    <UserPlus :size="14" /> Follow
                  </button>
                </div>
              </div>
            </div>
            <div v-else-if="searchQuery.length >= 2 && !searchLoading" class="empty-state !px-4 !py-6">
              <p>No users found for "{{ searchQuery }}"</p>
            </div>
            <div v-else class="empty-state !px-4 !py-6">
              <p>Type at least 2 characters to search</p>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { Users, UserPlus, Search, X, Eye, Check, ShieldOff } from 'lucide-vue-next'
import PullToRefresh from '@/components/PullToRefresh.vue'
import {
  getFollowers, getFollowing, searchUsers, followUser, unfollowUser,
  acceptFollower, blockFollower,
} from '@/api/client'
import type { FollowUser } from '@/types'
import { usePwa } from '@/composables/usePwa'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const activeTab = ref<'following' | 'followers'>('following')
const { isPwa } = usePwa()
const loading = ref(true)
const actionLoading = ref<number | null>(null)

const following = ref<FollowUser[]>([])
const followers = ref<FollowUser[]>([])

// Search modal
const showSearchModal = ref(false)
const searchQuery = ref('')
const searchResults = ref<FollowUser[]>([])
const searchLoading = ref(false)
const searchInputRef = ref<HTMLInputElement | null>(null)
let searchTimeout: ReturnType<typeof setTimeout> | null = null

function truncate(text: string, max: number): string {
  return text.length > max ? text.slice(0, max) + '…' : text
}

async function loadData() {
  loading.value = true
  try {
    const [followersRes, followingRes] = await Promise.all([
      getFollowers(),
      getFollowing(),
    ])
    followers.value = followersRes.data.followers
    following.value = followingRes.data.following
  } catch {
    // silently handle – lists stay empty
  } finally {
    loading.value = false
  }
}

async function handleFollow(user: FollowUser) {
  actionLoading.value = user.id
  try {
    await followUser(user.id)
    user.followStatus = 'pending'
  } catch {
    // ignore
  } finally {
    actionLoading.value = null
  }
}

async function handleUnfollow(user: FollowUser) {
  actionLoading.value = user.id
  try {
    await unfollowUser(user.id)
    following.value = following.value.filter(u => u.id !== user.id)
  } catch {
    // ignore
  } finally {
    actionLoading.value = null
  }
}

async function handleAccept(user: FollowUser) {
  actionLoading.value = user.id
  try {
    await acceptFollower(user.id)
    user.status = 'accepted'
  } catch {
    // ignore
  } finally {
    actionLoading.value = null
  }
}

async function handleBlock(user: FollowUser) {
  actionLoading.value = user.id
  try {
    await blockFollower(user.id)
    followers.value = followers.value.filter(u => u.id !== user.id)
  } catch {
    // ignore
  } finally {
    actionLoading.value = null
  }
}

function onSearchInput() {
  if (searchTimeout) clearTimeout(searchTimeout)
  if (searchQuery.value.length < 2) {
    searchResults.value = []
    return
  }
  searchLoading.value = true
  searchTimeout = setTimeout(async () => {
    try {
      const res = await searchUsers(searchQuery.value)
      searchResults.value = res.data.users
    } catch {
      searchResults.value = []
    } finally {
      searchLoading.value = false
    }
  }, 300)
}

function closeSearchModal() {
  showSearchModal.value = false
  searchQuery.value = ''
  searchResults.value = []
  loadData()
}

watch(showSearchModal, (open) => {
  if (open) {
    nextTick(() => searchInputRef.value?.focus())
  }
})

async function handleRefresh() {
  await loadData()
}

onMounted(() => {
  loadData()
})

onBeforeUnmount(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
})
</script>
