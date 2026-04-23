<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Star, Send, Trash2, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { getPublicProfile, getFollowingCoinDetail, addComment, deleteComment, rateCoin } from '@/api/client'
import type { LimitedCoin, CoinComment, CoinRating } from '@/types'
import { CATEGORY_COLORS } from '@/types'

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
  <div class="follower-coin-detail">
    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner" />
      <p>Loading coin details…</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-state">
      <p>{{ error }}</p>
      <button class="btn-back" @click="goBack">
        <ArrowLeft :size="18" /> Go Back
      </button>
    </div>

    <!-- Coin Detail -->
    <template v-else-if="coin">
      <!-- Header -->
      <header class="detail-header">
        <button class="btn-back" @click="goBack">
          <ArrowLeft :size="18" />
          <span>Back to Gallery</span>
        </button>
        <h1 class="coin-title">{{ coin.name }}</h1>
      </header>

      <div class="detail-layout">
        <!-- Image Gallery -->
        <section class="image-gallery">
          <div v-if="sortedImages.length" class="gallery-container" @click="cycleImage">
            <img
              :src="`/uploads/${currentImage?.filePath}`"
              :alt="coin.name"
              class="gallery-image"
            />
            <div v-if="sortedImages.length > 1" class="gallery-controls">
              <button class="gallery-btn" @click.stop="prevImage">
                <ChevronLeft :size="20" />
              </button>
              <span class="image-counter">{{ currentImageIndex + 1 }} / {{ sortedImages.length }}</span>
              <button class="gallery-btn" @click.stop="nextImage">
                <ChevronRight :size="20" />
              </button>
            </div>
          </div>
          <div v-else class="no-images">No images available</div>
        </section>

        <!-- Info + Interactions -->
        <div class="detail-content">
          <!-- Coin Details -->
          <section class="info-card">
            <h2 class="section-title">Details</h2>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Category</span>
                <span class="category-badge" :style="{ background: categoryColor }">
                  {{ coin.category }}
                </span>
              </div>
              <div class="info-item" v-if="coin.ruler">
                <span class="info-label">Ruler</span>
                <span class="info-value">{{ coin.ruler }}</span>
              </div>
              <div class="info-item" v-if="coin.era">
                <span class="info-label">Era</span>
                <span class="info-value">{{ coin.era }}</span>
              </div>
              <div class="info-item" v-if="coin.denomination">
                <span class="info-label">Denomination</span>
                <span class="info-value">{{ coin.denomination }}</span>
              </div>
              <div class="info-item" v-if="coin.material">
                <span class="info-label">Material</span>
                <span class="info-value material-tag" :class="`mat-${coin.material.toLowerCase()}`">
                  {{ coin.material }}
                </span>
              </div>
              <div class="info-item" v-if="coin.grade">
                <span class="info-label">Grade</span>
                <span class="info-value">{{ coin.grade }}</span>
              </div>
            </div>
          </section>

          <!-- Rating Section -->
          <section class="info-card">
            <h2 class="section-title">Rating</h2>
            <div class="rating-display">
              <div class="aggregate-rating">
                <div class="stars-display">
                  <Star
                    v-for="i in 5"
                    :key="'avg-' + i"
                    :size="20"
                    :fill="i <= Math.round(rating.average) ? '#c9a84c' : 'none'"
                    :stroke="i <= Math.round(rating.average) ? '#c9a84c' : 'var(--text-muted)'"
                  />
                </div>
                <span class="rating-text">
                  {{ rating.average.toFixed(1) }} avg · {{ rating.count }} {{ rating.count === 1 ? 'rating' : 'ratings' }}
                </span>
              </div>

              <div class="user-rating">
                <span class="info-label">Your Rating</span>
                <div class="stars-interactive">
                  <Star
                    v-for="i in 5"
                    :key="'user-' + i"
                    :size="24"
                    class="star-btn"
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

          <!-- Comments Section -->
          <section class="info-card comments-section">
            <h2 class="section-title">
              Comments
              <span class="comment-count">{{ comments.length }}</span>
            </h2>

            <div v-if="comments.length === 0" class="no-comments">
              No comments yet. Be the first!
            </div>

            <div v-else class="comments-list">
              <div v-for="comment in comments" :key="comment.id" class="comment-card">
                <div class="comment-header">
                  <img
                    v-if="comment.avatarPath"
                    :src="`/uploads/${comment.avatarPath}`"
                    class="comment-avatar"
                    alt=""
                  />
                  <div v-else class="comment-avatar placeholder-avatar">
                    {{ comment.username.charAt(0).toUpperCase() }}
                  </div>
                  <div class="comment-meta">
                    <span class="comment-username">{{ comment.username }}</span>
                    <span class="comment-time">{{ formatDate(comment.createdAt) }}</span>
                  </div>
                  <div v-if="comment.rating" class="comment-stars">
                    <Star
                      v-for="i in 5"
                      :key="'c-' + comment.id + '-' + i"
                      :size="14"
                      :fill="i <= comment.rating ? '#c9a84c' : 'none'"
                      :stroke="i <= comment.rating ? '#c9a84c' : 'var(--text-muted)'"
                    />
                  </div>
                  <button class="btn-delete-comment" @click="handleDeleteComment(comment.id)" title="Delete comment">
                    <Trash2 :size="14" />
                  </button>
                </div>
                <p class="comment-text">{{ comment.comment }}</p>
              </div>
            </div>

            <!-- Add Comment Form -->
            <div class="add-comment-form">
              <h3 class="form-title">Add a Comment</h3>
              <textarea
                v-model="newComment"
                class="comment-input"
                placeholder="Share your thoughts on this coin…"
                rows="3"
              />
              <div class="form-actions">
                <div class="comment-rating-picker">
                  <span class="info-label">Rating (optional)</span>
                  <div class="stars-interactive small">
                    <Star
                      v-for="i in 5"
                      :key="'new-' + i"
                      :size="18"
                      class="star-btn"
                      :fill="i <= (hoverRating || newCommentRating) ? '#c9a84c' : 'none'"
                      :stroke="i <= (hoverRating || newCommentRating) ? '#c9a84c' : 'var(--text-muted)'"
                      @mouseenter="hoverRating = i"
                      @mouseleave="hoverRating = 0"
                      @click="newCommentRating = newCommentRating === i ? 0 : i"
                    />
                  </div>
                </div>
                <button
                  class="btn-submit"
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

<style scoped>
.follower-coin-detail {
  max-width: 1100px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

/* Loading / Error */
.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 40vh;
  color: var(--text-secondary);
  gap: 1rem;
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

/* Header */
.detail-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.btn-back {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.85rem;
  transition: var(--transition-fast);
}

.btn-back:hover {
  color: var(--accent-gold);
  border-color: var(--accent-gold);
}

.coin-title {
  font-size: 1.5rem;
  color: var(--text-heading);
  font-weight: 600;
  margin: 0;
}

/* Layout */
.detail-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  align-items: start;
}

/* Image Gallery */
.image-gallery {
  position: sticky;
  top: 1.5rem;
}

.gallery-container {
  position: relative;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
}

.gallery-image {
  width: 100%;
  max-height: 500px;
  object-fit: contain;
  display: block;
}

.gallery-controls {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.7));
}

.gallery-btn {
  background: rgba(0, 0, 0, 0.5);
  border: none;
  color: #fff;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: var(--transition-fast);
}

.gallery-btn:hover {
  background: var(--accent-gold);
  color: var(--bg-primary);
}

.image-counter {
  color: rgba(255, 255, 255, 0.8);
  font-size: 0.8rem;
}

.no-images {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 4rem 2rem;
  text-align: center;
  color: var(--text-muted);
}

/* Info Card */
.info-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.25rem;
  margin-bottom: 1rem;
}

.section-title {
  font-size: 1rem;
  color: var(--text-heading);
  margin: 0 0 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Info Grid */
.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.info-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-value {
  color: var(--text-primary);
  font-size: 0.9rem;
}

.category-badge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-sm);
  color: #fff;
  font-size: 0.8rem;
  font-weight: 500;
  width: fit-content;
}

.material-tag {
  font-weight: 500;
}

.mat-gold { color: var(--mat-gold); }
.mat-silver { color: var(--mat-silver); }
.mat-bronze { color: var(--mat-bronze); }
.mat-copper { color: var(--mat-copper); }
.mat-electrum { color: var(--mat-electrum); }

/* Rating */
.rating-display {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.aggregate-rating {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.stars-display {
  display: flex;
  gap: 2px;
}

.rating-text {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.user-rating {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.stars-interactive {
  display: flex;
  gap: 2px;
}

.star-btn {
  cursor: pointer;
  transition: transform 0.15s ease;
}

.star-btn:hover {
  transform: scale(1.15);
}

/* Comments */
.comment-count {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
  font-size: 0.75rem;
  padding: 0.1rem 0.5rem;
  border-radius: var(--radius-full);
  font-weight: 600;
}

.no-comments {
  color: var(--text-muted);
  text-align: center;
  padding: 1.5rem 0;
  font-size: 0.9rem;
}

.comments-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
}

.comment-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.4rem;
}

.comment-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.placeholder-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
  font-size: 0.8rem;
  font-weight: 600;
}

.comment-meta {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.comment-username {
  color: var(--text-primary);
  font-size: 0.85rem;
  font-weight: 500;
}

.comment-time {
  color: var(--text-muted);
  font-size: 0.7rem;
}

.comment-stars {
  display: flex;
  gap: 1px;
  margin-left: auto;
}

.comment-text {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin: 0;
  line-height: 1.5;
  word-break: break-word;
}

.btn-delete-comment {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 4px;
  display: flex;
  align-items: center;
  transition: var(--transition-fast);
  flex-shrink: 0;
}

.btn-delete-comment:hover {
  color: #e74c3c;
  background: rgba(231, 76, 60, 0.1);
}

/* Add Comment Form */
.add-comment-form {
  border-top: 1px solid var(--border-subtle);
  padding-top: 1rem;
  margin-top: 0.5rem;
}

.form-title {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0 0 0.75rem;
  font-weight: 500;
}

.comment-input {
  width: 100%;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  padding: 0.6rem 0.75rem;
  font-size: 0.85rem;
  resize: vertical;
  font-family: inherit;
  box-sizing: border-box;
  transition: border-color var(--transition-fast);
}

.comment-input::placeholder {
  color: var(--text-muted);
}

.comment-input:focus {
  outline: none;
  border-color: var(--accent-gold);
}

.form-actions {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-top: 0.75rem;
  gap: 1rem;
}

.comment-rating-picker {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stars-interactive.small {
  gap: 1px;
}

.btn-submit {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--accent-gold);
  color: var(--bg-primary);
  border: none;
  padding: 0.55rem 1.2rem;
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: var(--transition-fast);
  white-space: nowrap;
}

.btn-submit:hover:not(:disabled) {
  filter: brightness(1.1);
  box-shadow: 0 0 12px var(--accent-gold-dim);
}

.btn-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Responsive */
@media (max-width: 768px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }

  .image-gallery {
    position: static;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .detail-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .form-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .btn-submit {
    justify-content: center;
  }
}
</style>
