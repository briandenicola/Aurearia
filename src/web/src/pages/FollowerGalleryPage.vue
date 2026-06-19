<template>
  <div class="container">
    <div class="page-header">
      <button class="btn-back" @click="router.back()">
        <ArrowLeft :size="20" />
      </button>
      <div v-if="profile" class="profile-info">
        <AuthenticatedImage
          :media-path="profile.avatarPath ? profile.avatarPath : '/coin-logo.jpg'"
          alt="Avatar"
          class="profile-avatar"
        />
        <div class="profile-text">
          <h1 class="profile-username">{{ profile.username }}</h1>
          <p v-if="profile.bio" class="profile-bio">{{ profile.bio }}</p>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading collection...</p>
    </div>

    <div v-else-if="coins.length" class="coins-grid">
      <div
        v-for="coin in coins"
        :key="coin.id"
        class="coin-card card"
        @click="router.push(`/followers/${username}/coins/${coin.id}`)"
      >
        <div class="card-image-container">
          <AuthenticatedImage
            v-if="getPrimaryImage(coin)"
            :media-path="getPrimaryImage(coin)"
            :alt="coin.name"
            class="card-image"
          />
          <div v-else class="card-image-placeholder">
            <Coins :size="48" :stroke-width="1" />
          </div>
        </div>
        <div class="card-body">
          <h3 class="card-title">{{ coin.name }}</h3>
          <div class="card-meta">
            <span v-if="coin.ruler" class="meta-item">{{ coin.ruler }}</span>
            <span v-if="coin.era" class="meta-item">{{ coin.era }}</span>
          </div>
          <div class="card-details">
            <span
              v-if="coin.category"
              class="category-badge"
              :style="{ backgroundColor: CATEGORY_COLORS[coin.category] }"
            >
              {{ coin.category }}
            </span>
          </div>
          <div v-if="coin.grade" class="card-grade">{{ coin.grade }}</div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <Coins :size="48" :stroke-width="1" />
      <h3>No coins to show</h3>
      <p>This user hasn't added any coins yet.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Coins } from 'lucide-vue-next'
import { getPublicProfile, getFollowingCoins } from '@/api/client'
import type { LimitedCoin, PublicProfile } from '@/types'
import { CATEGORY_COLORS } from '@/types'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const route = useRoute()
const router = useRouter()
const username = route.params.username as string

const profile = ref<PublicProfile | null>(null)
const coins = ref<LimitedCoin[]>([])
const loading = ref(true)

function getPrimaryImage(coin: LimitedCoin): string | null {
  if (!coin.images || coin.images.length === 0) return null
  const primary = coin.images.find((img) => img.isPrimary)
  const img = primary ?? coin.images[0]
  return img ? img.filePath : null
}

onMounted(async () => {
  try {
    const profileRes = await getPublicProfile(username)
    profile.value = profileRes.data

    const coinsRes = await getFollowingCoins(profile.value.id)
    coins.value = coinsRes.data.coins
  } catch (err) {
    console.error('Failed to load follower gallery', err)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1.5rem;
}

.page-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
}

.btn-back {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.btn-back:hover {
  background: var(--bg-card-hover);
  color: var(--accent-gold);
  border-color: var(--border-accent);
}

.profile-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.profile-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-full);
  object-fit: cover;
  border: 2px solid var(--border-accent);
}

.profile-text {
  display: flex;
  flex-direction: column;
}

.profile-username {
  font-size: 1.25rem;
  color: var(--text-heading);
  margin: 0;
  line-height: 1.3;
}

.profile-bio {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0.15rem 0 0;
  line-height: 1.4;
}

/* Loading */
.loading-overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  gap: 1rem;
}

.loading-overlay p {
  color: var(--text-secondary);
}

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Coin Grid */
.coins-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.coin-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: all var(--transition-fast);
  box-shadow: var(--shadow-card);
}

.coin-card:hover {
  transform: translateY(-4px);
  border-color: var(--border-accent);
  box-shadow: var(--shadow-card), var(--shadow-glow);
  background: var(--bg-card-hover);
}

.card-image-container {
  position: relative;
  width: 100%;
  padding-top: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
}

.card-image {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.card-image-placeholder {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.card-body {
  padding: 0.75rem;
}

.card-title {
  font-size: 0.9rem;
  color: var(--text-primary);
  margin: 0 0 0.35rem;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem 0.5rem;
  margin-bottom: 0.35rem;
}

.meta-item {
  font-size: 0.78rem;
  color: var(--text-secondary);
}

.card-details {
  margin-bottom: 0.35rem;
}

.category-badge {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
  color: #fff;
  letter-spacing: 0.02em;
}

.card-grade {
  font-size: 0.75rem;
  color: var(--accent-gold);
  font-weight: 600;
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 4rem 1rem;
  color: var(--text-muted);
}

.empty-state h3 {
  color: var(--text-primary);
  margin: 1rem 0 0.5rem;
}

.empty-state p {
  color: var(--text-secondary);
}

/* Responsive */
@media (max-width: 1024px) {
  .coins-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .coins-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    gap: 0.75rem;
  }

  .profile-username {
    font-size: 1.1rem;
  }
}
</style>
