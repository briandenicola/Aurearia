<template>
  <div class="container py-6">
    <div class="page-header !mb-8 !justify-start gap-4">
      <button class="btn btn-ghost !h-10 !w-10 !shrink-0 !justify-center !p-0" @click="router.back()">
        <ArrowLeft :size="20" />
      </button>
      <div v-if="profile" class="flex min-w-0 items-center gap-3">
        <AuthenticatedImage
          :media-path="profile.avatarPath ? profile.avatarPath : '/coin-logo.jpg'"
          alt="Avatar"
          class="h-12 w-12 rounded-full border-2 border-border-accent object-cover"
        />
        <div class="min-w-0">
          <h1 class="truncate text-xl font-semibold text-heading">{{ profile.username }}</h1>
          <p v-if="profile.bio" class="mt-[0.15rem] text-body leading-[1.4] text-text-secondary">
            {{ profile.bio }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p class="text-text-secondary">Loading collection...</p>
    </div>

    <div v-else-if="coins.length" class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="coin in coins"
        :key="coin.id"
        class="card cursor-pointer overflow-hidden hover:-translate-y-1 hover:bg-card-hover"
        @click="router.push(`/followers/${username}/coins/${coin.id}`)"
      >
        <div class="relative aspect-square overflow-hidden bg-[var(--bg-secondary)]">
          <AuthenticatedImage
            v-if="getPrimaryImage(coin)"
            :media-path="getPrimaryImage(coin)"
            :alt="coin.name"
            class="absolute inset-0 h-full w-full object-cover"
          />
          <div v-else class="absolute inset-0 flex items-center justify-center text-text-muted">
            <Coins :size="48" :stroke-width="1" />
          </div>
        </div>
        <div class="space-y-[0.35rem] p-3">
          <h3 class="truncate text-base leading-[1.3] text-text-primary">{{ coin.name }}</h3>
          <div class="flex flex-wrap gap-x-2 gap-y-1">
            <span v-if="coin.ruler" class="text-[0.78rem] text-text-secondary">{{ coin.ruler }}</span>
            <span v-if="coin.era" class="text-[0.78rem] text-text-secondary">{{ coin.era }}</span>
          </div>
          <div v-if="coin.category">
            <span
              class="inline-flex w-fit rounded-full px-2 py-[0.15rem] text-label font-semibold tracking-[0.02em] text-white"
              :style="{ backgroundColor: CATEGORY_COLORS[coin.category] }"
            >
              {{ coin.category }}
            </span>
          </div>
          <div v-if="coin.grade" class="text-sm font-semibold text-gold">{{ coin.grade }}</div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <Coins :size="48" :stroke-width="1" />
      <h3 class="text-text-primary">No coins to show</h3>
      <p class="text-text-secondary">This user hasn't added any coins yet.</p>
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
