import { ref, onBeforeUnmount, watch } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import { getTags } from '@/api/client'
import type { Tag } from '@/types'

export function useCollectionFilters() {
  const store = useCoinsStore()

  const selectedCategory = store.selectedCategory !== undefined ? ref(store.selectedCategory) : ref('')
  const search = ref(store.searchQuery)
  const page = ref(1)
  const sortKey = ref(store.activeSortKey || localStorage.getItem('defaultSort') || 'updated_at_desc')
  const selectedTag = ref('')
  const userTags = ref<Tag[]>([])

  let debounceTimer: ReturnType<typeof setTimeout>

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
      const key = 'coins:randomSeed'
      const cached = sessionStorage.getItem(key)
      if (cached) {
        seed = parseInt(cached, 10)
      } else {
        seed = Math.floor(Math.random() * 1_000_000) + 1
        sessionStorage.setItem(key, String(seed))
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
      sessionStorage.setItem('coins:randomSeed', String(Math.floor(Math.random() * 1_000_000) + 1))
    }
    loadCoins()
  })

  onBeforeUnmount(() => {
    clearTimeout(debounceTimer)
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
