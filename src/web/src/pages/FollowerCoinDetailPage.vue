<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Star, Send, Trash2, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { getPublicProfile, getFollowingCoinDetail, addComment, deleteComment, rateCoin } from '@/api/client'
import type { LimitedCoin, CoinComment, CoinRating } from '@/types'
import { CATEGORY_COLORS } from '@/types'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const route = useRoute()
const router = useRouter()

const coin = ref<LimitedCoin | null>(null)
const comments = ref<CoinComment[]>([])
const rating = ref<CoinRating>({ average: 0, count: 0, userRating: 0 })
const loading = ref(true)
const error = ref('')
const currentImageIndex = ref(0)

const newComment = ref('')
const newCommentRating = ref(0)
const hoverRating = ref(0)
const hoverUserRating = ref(0)
const submitting = ref(false)

const username = computed(() => route.params['username'] as string)
const coinId = computed(() => Number(route.params['coinId']))

const sortedImages = computed(() => {
  if (!coin.value?.images?.length) return []
  return [...coin.value.images].sort((a, b) => (b.isPrimary ? 1 : 0) - (a.isPrimary ? 1 : 0))
})

const currentImage = computed(() => sortedImages.value[currentImageIndex.value])

const categoryColor = computed(() => {
  if (!coin.value) return '#888'
  return CATEGORY_COLORS[coin.value.category] || '#888'
})

function prevImage() {
  if (sortedImages.value.length <= 1) return
  currentImageIndex.value = (currentImageIndex.value - 1 + sortedImages.value.length) % sortedImages.value.length
}

function nextImage() {
  if (sortedImages.value.length <= 1) return
  currentImageIndex.value = (currentImageIndex.value + 1) % sortedImages.value.length
}

function cycleImage() {
  nextImage()
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function loadCoin() {
  loading.value = true
  error.value = ''
  try {
    const profile = await getPublicProfile(username.value)
    const result = await getFollowingCoinDetail(profile.data.id, coinId.value)
    coin.value = result.data
    comments.value = (result.data as { comments?: CoinComment[] }).comments || []
    rating.value = (result.data as { rating?: CoinRating }).rating || { average: 0, count: 0, userRating: 0 }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'Failed to load coin'
    if (typeof e === 'object' && e !== null && 'response' in e) {
      const axiosErr = e as { response?: { data?: { error?: string } } }
      error.value = axiosErr.response?.data?.error || msg
    } else {
      error.value = msg
    }
  } finally {
    loading.value = false
  }
}

async function handleRate(stars: number) {
  try {
    const updated = await rateCoin(coinId.value, stars)
    rating.value = updated.data
  } catch (e: unknown) {
    console.error('Failed to rate coin', e)
  }
}

async function handleAddComment() {
  if (!newComment.value.trim()) return
  submitting.value = true
  try {
    const comment = await addComment(coinId.value, newComment.value.trim(), newCommentRating.value || undefined)
    comments.value.push(comment.data)
    newComment.value = ''
    newCommentRating.value = 0
  } catch (e: unknown) {
    console.error('Failed to add comment', e)
  }finally {
    submitting.value = false
  }
}

async function handleDeleteComment(commentId: number) {
  try {
    await deleteComment(coinId.value, commentId)
    comments.value = comments.value.filter(c => c.id !== commentId)
  } catch (e: unknown) {
    console.error('Failed to delete comment', e)
  }
}

function goBack() {
  router.push(`/followers/${username.value}/gallery`)
}

onMounted(loadCoin)
</script>

<template>
  <div class="mx-auto w-full max-w-[1100px] px-4 py-8 md:px-6">
    <div v-if="loading" class="flex min-h-[40vh] flex-col items-center justify-center gap-4 text-text-secondary">
      <div class="spinner" />
      <p>Loading coin details…</p>
    </div>

    <div v-else-if="error" class="flex min-h-[40vh] flex-col items-center justify-center gap-4 text-text-secondary">
      <p>{{ error }}</p>
      <button class="btn btn-secondary btn-sm" @click="goBack">
        <ArrowLeft :size="18" /> Go Back
      </button>
    </div>

    <template v-else-if="coin">
      <header class="mb-6 flex flex-col items-start gap-2 md:flex-row md:items-center md:gap-4">
        <button class="btn btn-secondary btn-sm" @click="goBack">
          <ArrowLeft :size="18" />
          <span>Back to Gallery</span>
        </button>
        <h1 class="m-0 text-xl font-semibold text-heading">{{ coin.name }}</h1>
      </header>

      <div class="grid items-start gap-6 md:grid-cols-2">
        <section class="md:sticky md:top-6">
          <div
            v-if="sortedImages.length"
            class="relative cursor-pointer overflow-hidden rounded-md border border-border-subtle bg-card"
            @click="cycleImage"
          >
            <AuthenticatedImage
              :media-path="currentImage?.filePath"
              :alt="coin.name"
              class="block max-h-[500px] w-full object-contain"
            />
            <div v-if="sortedImages.length > 1" class="absolute inset-x-0 bottom-0 flex items-center justify-between bg-gradient-to-t from-black/70 to-transparent p-2">
              <button
                class="flex h-8 w-8 items-center justify-center rounded-full bg-black/50 text-white transition hover:bg-[var(--accent-gold)] hover:text-[var(--bg-primary)]"
                @click.stop="prevImage"
              >
                <ChevronLeft :size="20" />
              </button>
              <span class="text-chip text-white/80">{{ currentImageIndex + 1 }} / {{ sortedImages.length }}</span>
              <button
                class="flex h-8 w-8 items-center justify-center rounded-full bg-black/50 text-white transition hover:bg-[var(--accent-gold)] hover:text-[var(--bg-primary)]"
                @click.stop="nextImage"
              >
                <ChevronRight :size="20" />
              </button>
            </div>
          </div>
          <div v-else class="rounded-md border border-border-subtle bg-card px-8 py-16 text-center text-text-muted">
            No images available
          </div>
        </section>

        <div class="space-y-4">
          <section class="card p-5">
            <h2 class="mb-4 flex items-center gap-2 text-lg font-medium text-heading">Details</h2>
            <div class="grid gap-3 md:grid-cols-2">
              <div class="flex flex-col gap-1">
                <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Category</span>
                <span class="inline-flex w-fit rounded-sm px-[0.6rem] py-[0.2rem] text-chip font-medium text-white" :style="{ background: categoryColor }">
                  {{ coin.category }}
                </span>
              </div>
              <div v-if="coin.ruler" class="flex flex-col gap-1">
                <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Ruler</span>
                <span class="text-base text-text-primary">{{ coin.ruler }}</span>
              </div>
              <div v-if="coin.era" class="flex flex-col gap-1">
                <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Era</span>
                <span class="text-base text-text-primary">{{ coin.era }}</span>
              </div>
              <div v-if="coin.denomination" class="flex flex-col gap-1">
                <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Denomination</span>
                <span class="text-base text-text-primary">{{ coin.denomination }}</span>
              </div>
              <div v-if="coin.material" class="flex flex-col gap-1">
                <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Material</span>
                <span
                  :class="[
                    'text-base font-medium',
                    {
                      'text-[var(--mat-gold)]': coin.material.toLowerCase() === 'gold',
                      'text-[var(--mat-silver)]': coin.material.toLowerCase() === 'silver',
                      'text-[var(--mat-bronze)]': coin.material.toLowerCase() === 'bronze',
                      'text-[var(--mat-copper)]': coin.material.toLowerCase() === 'copper',
                      'text-[var(--mat-electrum)]': coin.material.toLowerCase() === 'electrum',
                    },
                  ]"
                >
                  {{ coin.material }}
                </span>
              </div>
              <div v-if="coin.grade" class="flex flex-col gap-1">
                <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Grade</span>
                <span class="text-base text-text-primary">{{ coin.grade }}</span>
              </div>
            </div>
          </section>

          <section class="card p-5">
            <h2 class="mb-4 flex items-center gap-2 text-lg font-medium text-heading">Rating</h2>
            <div class="flex flex-col gap-4">
              <div class="flex flex-col gap-2 md:flex-row md:items-center md:gap-3">
                <div class="flex gap-[2px]">
                  <Star
                    v-for="i in 5"
                    :key="'avg-' + i"
                    :size="20"
                    :fill="i <= Math.round(rating.average) ? '#c9a84c' : 'none'"
                    :stroke="i <= Math.round(rating.average) ? '#c9a84c' : 'var(--text-muted)'"
                  />
                </div>
                <span class="text-body text-text-secondary">
                  {{ rating.average.toFixed(1) }} avg · {{ rating.count }} {{ rating.count === 1 ? 'rating' : 'ratings' }}
                </span>
              </div>

              <div class="flex flex-col gap-[0.35rem]">
                <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Your Rating</span>
                <div class="flex gap-[2px]">
                  <Star
                    v-for="i in 5"
                    :key="'user-' + i"
                    :size="24"
                    class="cursor-pointer transition-transform duration-150 hover:scale-110"
                    :fill="i <= (hoverUserRating || rating.userRating) ? '#c9a84c' : 'none'"
                    :stroke="i <= (hoverUserRating || rating.userRating) ? '#c9a84c' : 'var(--text-muted)'"
                    @mouseenter="hoverUserRating = i"
                    @mouseleave="hoverUserRating = 0"
                    @click="handleRate(i)"
                  />
                </div>
              </div>
            </div>
          </section>

          <section class="card p-5">
            <h2 class="mb-4 flex items-center gap-2 text-lg font-medium text-heading">
              Comments
              <span class="inline-flex min-w-[1.75rem] items-center justify-center rounded-full bg-[var(--accent-gold-dim)] px-2 py-px text-sm font-semibold text-gold">
                {{ comments.length }}
              </span>
            </h2>

            <div v-if="comments.length === 0" class="py-6 text-center text-base text-text-muted">
              No comments yet. Be the first!
            </div>

            <div v-else class="mb-5 flex flex-col gap-3">
              <div v-for="comment in comments" :key="comment.id" class="rounded-sm border border-border-subtle bg-[var(--bg-secondary)] p-3">
                <div class="mb-2 flex items-start gap-2">
                  <AuthenticatedImage
                    v-if="comment.avatarPath"
                    :media-path="comment.avatarPath"
                    class="h-8 w-8 shrink-0 rounded-full object-cover"
                    alt=""
                  />
                  <div v-else class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[var(--accent-gold-dim)] text-chip font-semibold text-gold">
                    {{ comment.username.charAt(0).toUpperCase() }}
                  </div>
                  <div class="flex min-w-0 flex-1 flex-col">
                    <span class="text-body font-medium text-text-primary">{{ comment.username }}</span>
                    <span class="text-label tracking-normal text-text-muted">{{ formatDate(comment.createdAt) }}</span>
                  </div>
                  <div v-if="comment.rating" class="ml-auto flex gap-px pt-[0.125rem]">
                    <Star
                      v-for="i in 5"
                      :key="'c-' + comment.id + '-' + i"
                      :size="14"
                      :fill="i <= comment.rating ? '#c9a84c' : 'none'"
                      :stroke="i <= comment.rating ? '#c9a84c' : 'var(--text-muted)'"
                    />
                  </div>
                  <button
                    class="flex shrink-0 items-center rounded-[4px] p-1 text-text-muted transition hover:bg-[rgba(231,76,60,0.1)] hover:text-[rgb(231,76,60)]"
                    title="Delete comment"
                    @click="handleDeleteComment(comment.id)"
                  >
                    <Trash2 :size="14" />
                  </button>
                </div>
                <p class="m-0 break-words text-body leading-6 text-text-secondary">{{ comment.comment }}</p>
              </div>
            </div>

            <div class="mt-2 border-t border-border-subtle pt-4">
              <h3 class="mb-3 text-body font-medium text-text-secondary">Add a Comment</h3>
              <textarea
                v-model="newComment"
                class="min-h-[5.5rem] w-full resize-y rounded-sm border border-border-subtle bg-input px-3 py-2.5 text-body text-text-primary transition placeholder:text-text-muted focus:border-[var(--accent-gold)] focus:outline-none"
                placeholder="Share your thoughts on this coin…"
                rows="3"
              />
              <div class="mt-3 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
                <div class="flex flex-col gap-1">
                  <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Rating (optional)</span>
                  <div class="flex gap-px">
                    <Star
                      v-for="i in 5"
                      :key="'new-' + i"
                      :size="18"
                      class="cursor-pointer transition-transform duration-150 hover:scale-110"
                      :fill="i <= (hoverRating || newCommentRating) ? '#c9a84c' : 'none'"
                      :stroke="i <= (hoverRating || newCommentRating) ? '#c9a84c' : 'var(--text-muted)'"
                      @mouseenter="hoverRating = i"
                      @mouseleave="hoverRating = 0"
                      @click="newCommentRating = newCommentRating === i ? 0 : i"
                    />
                  </div>
                </div>
                <button
                  class="btn btn-primary btn-sm w-full justify-center whitespace-nowrap disabled:cursor-not-allowed disabled:opacity-50 md:w-auto"
                  :disabled="!newComment.trim() || submitting"
                  @click="handleAddComment"
                >
                  <Send :size="16" />
                  {{ submitting ? 'Posting…' : 'Post Comment' }}
                </button>
              </div>
            </div>
          </section>
        </div>
      </div>
    </template>
  </div>
</template>
