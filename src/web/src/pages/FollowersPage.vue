<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container">
      <div class="page-header">
        <h1>Followers</h1>
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

      <div class="followers-layout">
        <!-- Tab Nav -->
        <div class="tab-nav">
          <button
            class="tab-btn"
            :class="{ active: activeTab === 'following' }"
            @click="activeTab = 'following'"
          >
            <UserPlus :size="16" /> Following
            <span v-if="following.length" class="tab-count">{{ following.length }}</span>
          </button>
          <button
            class="tab-btn"
            :class="{ active: activeTab === 'followers' }"
            @click="activeTab = 'followers'"
          >
            <Users :size="16" /> Followers
            <span v-if="followers.length" class="tab-count">{{ followers.length }}</span>
          </button>
        </div>

        <!-- Loading -->
        <div v-if="loading" class="loading-state">
          <div class="spinner" />
          <p>Loading...</p>
        </div>

        <!-- Following Tab -->
        <div v-else-if="activeTab === 'following'">
          <div v-if="following.length === 0" class="empty-state">
            <Users :size="48" />
            <h3>Not following anyone yet</h3>
            <p>Search for users to follow and see their collections.</p>
            <button class="btn btn-primary btn-sm" @click="showSearchModal = true">
              <UserPlus :size="16" /> Find Users
            </button>
          </div>
          <div v-else class="user-grid">
            <div v-for="user in following" :key="user.id" class="user-card card">
              <div class="user-card-body">
                <AuthenticatedImage
                  :media-path="user.avatarPath ? user.avatarPath : '/coin-logo.jpg'"
                  :alt="user.username"
                  class="user-avatar"
                />
                <div class="user-info">
                  <span class="user-name">{{ user.username }}</span>
                  <p v-if="user.bio" class="user-bio">{{ truncate(user.bio, 80) }}</p>
                  <div class="user-meta">
                    <span v-if="user.isPublic && user.coinCount > 0" class="coin-badge">
                      {{ user.coinCount }} coins
                    </span>
                  </div>
                </div>
              </div>
              <div class="user-card-actions">
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
          <div v-if="followers.length === 0" class="empty-state">
            <Users :size="48" />
            <h3>No followers yet</h3>
            <p>When other users follow you, they'll appear here.</p>
          </div>
          <div v-else class="user-grid">
            <div v-for="user in followers" :key="user.id" class="user-card card">
              <div class="user-card-body">
                <AuthenticatedImage
                  :media-path="user.avatarPath ? user.avatarPath : '/coin-logo.jpg'"
                  :alt="user.username"
                  class="user-avatar"
                />
                <div class="user-info">
                  <span class="user-name">{{ user.username }}</span>
                  <p v-if="user.bio" class="user-bio">{{ truncate(user.bio, 80) }}</p>
                  <span v-if="user.status === 'pending'" class="status-badge pending">Pending</span>
                  <span v-else-if="user.status === 'accepted'" class="status-badge accepted">Accepted</span>
                </div>
              </div>
              <div class="user-card-actions">
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

      <!-- Search Modal -->
      <Teleport to="body">
        <div v-if="showSearchModal" class="modal-overlay" @click.self="closeSearchModal">
          <div class="modal-content card">
            <div class="modal-header">
              <h2><Search :size="20" /> Find Users</h2>
              <button class="modal-close" @click="closeSearchModal">
                <X :size="20" />
              </button>
            </div>
            <div class="modal-body">
              <p class="search-hint">Only users with public profiles appear in search results.</p>
              <div class="search-input-wrap">
                <Search :size="16" class="search-icon" />
                <input
                  ref="searchInputRef"
                  v-model="searchQuery"
                  type="text"
                  class="form-input search-input"
                  placeholder="Search by username..."
                  @input="onSearchInput"
                />
              </div>
              <div v-if="searchLoading" class="loading-state compact">
                <div class="spinner" />
              </div>
              <div v-else-if="searchResults.length > 0" class="search-results">
                <div v-for="user in searchResults" :key="user.id" class="user-card compact">
                  <div class="user-card-body">
                    <AuthenticatedImage
                      :media-path="user.avatarPath ? user.avatarPath : '/coin-logo.jpg'"
                      :alt="user.username"
                      class="user-avatar"
                    />
                    <div class="user-info">
                      <span class="user-name">{{ user.username }}</span>
                      <p v-if="user.bio" class="user-bio">{{ truncate(user.bio, 60) }}</p>
                    </div>
                  </div>
                  <div class="user-card-actions">
                    <span v-if="user.followStatus === 'pending'" class="status-badge pending">Pending</span>
                    <span v-else-if="user.followStatus === 'accepted'" class="following-badge">Following</span>
                    <span v-else-if="user.followStatus === 'blocked'" class="status-badge blocked">Blocked</span>
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
              <div v-else-if="searchQuery.length >= 2 && !searchLoading" class="empty-state compact">
                <p>No users found for "{{ searchQuery }}"</p>
              </div>
              <div v-else class="empty-state compact">
                <p>Type at least 2 characters to search</p>
              </div>
            </div>
          </div>
        </div>
      </Teleport>
    </div>
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

<style scoped>
.page-header h1 {
  flex: 1;
}

.followers-layout {
  max-width: 900px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  overflow-x: hidden;
}

/* Tabs */
.tab-nav {
  display: flex;
  gap: 0.25rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 0.3rem;
}

.tab-btn {
  flex: 1;
  padding: 0.6rem 1rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
}

.tab-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tab-btn:hover:not(.active) {
  color: var(--text-primary);
}

.tab-count {
  background: var(--border-subtle);
  color: var(--text-secondary);
  font-size: 0.7rem;
  padding: 0.1rem 0.45rem;
  border-radius: var(--radius-full);
  min-width: 1.3rem;
  text-align: center;
}

.tab-btn.active .tab-count {
  background: var(--accent-gold);
  color: var(--bg-primary);
}

/* User Grid */
.user-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(340px, 100%), 1fr));
  gap: 1rem;
}

.user-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  overflow: hidden;
}

.user-card-body {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.user-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-full);
  object-fit: cover;
  border: 2px solid var(--border-subtle);
  flex-shrink: 0;
}

.user-info {
  min-width: 0;
  flex: 1;
}

.user-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-bio {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin: 0.2rem 0 0;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-meta {
  margin-top: 0.3rem;
}

.coin-badge {
  font-size: 0.7rem;
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.user-card-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
}

/* Loading */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 1rem;
  color: var(--text-secondary);
}

.loading-state.compact {
  padding: 2rem 1rem;
}

.spinner {
  width: 28px;
  height: 28px;
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 3rem 1rem;
  color: var(--text-secondary);
  text-align: center;
}

.empty-state.compact {
  padding: 1.5rem 1rem;
}

.empty-state h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-primary);
}

.empty-state p {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-muted);
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 5vh 1rem;
  z-index: 1000;
  overflow-y: auto;
}

.modal-content {
  width: 100%;
  max-width: 520px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-header h2 {
  font-size: 1rem;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast);
}

.modal-close:hover {
  color: var(--text-primary);
}

.modal-body {
  padding: 1rem 1.25rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.search-input-wrap {
  position: relative;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-muted);
  pointer-events: none;
}

.search-input {
  padding-left: 2.25rem;
  width: 100%;
}

.search-results {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.user-card.compact {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.user-card.compact .user-card-body {
  flex: 1;
  min-width: 0;
}

.user-card.compact .user-card-actions {
  flex-shrink: 0;
}

.following-badge {
  font-size: 0.75rem;
  color: var(--accent-gold);
  font-weight: 500;
  padding: 0.3rem 0.6rem;
  background: var(--accent-gold-dim);
  border-radius: var(--radius-sm);
}

.status-badge {
  font-size: 0.7rem;
  font-weight: 500;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
}

.status-badge.pending {
  background: var(--accent-gold-glow);
  color: var(--accent-gold);
}

.status-badge.accepted {
  background: var(--accent-gold-glow);
  color: var(--cat-greek);
}

.status-badge.blocked {
  background: var(--accent-gold-glow);
  color: var(--error-bg);
}

.search-hint {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin: 0;
}

/* Responsive */
@media (max-width: 640px) {
  .user-grid {
    grid-template-columns: 1fr;
  }

  .page-header h1 {
    font-size: 1.2rem;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: row;
    align-items: center;
  }
}
</style>
