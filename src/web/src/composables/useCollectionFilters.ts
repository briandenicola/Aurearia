import { ref, onBeforeUnmount, onMounted, watch } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import { getTags } from '@/api/client'
import { usePwa } from '@/composables/usePwa'
import type { Tag } from '@/types'

const RANDOM_SEED_KEY = 'coins:randomSeed'
const PWA_RESUME_THRESHOLD_MS = 30_000

function generateRandomSeed(): number {
  return Math.floor(Math.random() * 1_000_000) + 1
}

// Module-level flag: true once any consumer of this composable has mounted in
// the current app lifetime. Resets when JS modules reload (i.e., PWA relaunch
// or full page reload).
let appLifecycleStarted = false

export function useCollectionFilters() {
  const store = useCoinsStore()
  const { isPwa } = usePwa()

  // On the first composable mount of a PWA's lifetime, treat it as a fresh
  // launch and clear any cached random seed so the next random fetch reshuffles.
  if (isPwa && !appLifecycleStarted) {
    sessionStorage.removeItem(RANDOM_SEED_KEY)
    store.galleryIndex = 0
  }
  appLifecycleStarted = true

  const selectedCategory = store.selectedCategory !== undefined ? ref(store.selectedCategory) : ref('')
  const search = ref(store.searchQuery)
  const page = ref(1)
  const sortKey = ref(store.activeSortKey || localStorage.getItem('defaultSort') || 'updated_at_desc')
  const selectedTag = ref('')
  const userTags = ref<Tag[]>([])

  let debounceTimer: ReturnType<typeof setTimeout>
  let hiddenAt = 0

  async function fetchUserTags() {
    try {
      const res = await getTags()
      userTags.value = res.data?.tags ?? []
    } catch { /* ignore */ }
  }

  function loadCoins() {
    const [sort, order] = sortKey.value.split('_').length === 3
      ? [sortKey.value.split('_').slice(0, 2).join('_'), sortKey.value.split('_')[2]]
      : [sortKey.value.split('_')[0], sortKey.value.split('_')[1]]
    store.selectedCategory = selectedCategory.value
    store.searchQuery = search.value

    // For random sort, generate a per-session seed (stable across pagination within a session)
    let seed: number | undefined
    if (sort === 'random') {
      const cached = sessionStorage.getItem(RANDOM_SEED_KEY)
      if (cached) {
        seed = parseInt(cached, 10)
      } else {
        seed = generateRandomSeed()
        sessionStorage.setItem(RANDOM_SEED_KEY, String(seed))
      }
    }

    store.fetchCoins({
      category: selectedCategory.value || undefined,
      search: search.value || undefined,
      tag: selectedTag.value || undefined,
      wishlist: 'false',
      sold: 'false',
      page: page.value,
      sort,
      order,
      seed,
    })
  }

  watch(selectedCategory, () => {
    page.value = 1
    loadCoins()
  })

  watch(selectedTag, () => {
    page.value = 1
    loadCoins()
  })

  watch(search, () => {
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      page.value = 1
      loadCoins()
    }, 300)
  })

  watch(page, loadCoins)
  watch(sortKey, () => {
    store.activeSortKey = sortKey.value
    page.value = 1
    // Reset the random seed when the user re-selects Random so the order shuffles.
    const sort = sortKey.value.split('_').slice(0, -1).join('_')
    if (sort === 'random') {
      sessionStorage.setItem(RANDOM_SEED_KEY, String(generateRandomSeed()))
    }
    loadCoins()
  })

  // In PWA mode, when the app resumes after being backgrounded for a while,
  // treat it as a relaunch: reshuffle if currently sorted by Random.
  function onVisibilityChange() {
    if (document.visibilityState === 'hidden') {
      hiddenAt = Date.now()
      return
    }
    if (document.visibilityState !== 'visible' || hiddenAt === 0) return
    const hiddenFor = Date.now() - hiddenAt
    hiddenAt = 0
    if (!isPwa || hiddenFor < PWA_RESUME_THRESHOLD_MS) return
    const sort = sortKey.value.split('_').slice(0, -1).join('_')
    if (sort !== 'random') return
    sessionStorage.setItem(RANDOM_SEED_KEY, String(generateRandomSeed()))
    store.galleryIndex = 0
    page.value = 1
    loadCoins()
  }

  if (isPwa) {
    onMounted(() => {
      document.addEventListener('visibilitychange', onVisibilityChange)
    })
  }

  onBeforeUnmount(() => {
    clearTimeout(debounceTimer)
    if (isPwa) {
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  })

  return {
    selectedCategory,
    search,
    page,
    sortKey,
    selectedTag,
    userTags,
    fetchUserTags,
    loadCoins,
  }
}
