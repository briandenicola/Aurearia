import { ref } from 'vue'

type RunHistoryResponse<T> = {
  runs?: T[]
  total?: number
}

export function useRunHistoryPagination<T>(
  fetchRuns: (page: number, limit: number) => Promise<RunHistoryResponse<T>>,
  limit = 5,
) {
  const runs = ref<T[]>([])
  const total = ref(0)
  const page = ref(1)
  const loading = ref(false)

  async function loadRuns() {
    loading.value = true
    try {
      const data = await fetchRuns(page.value, limit)
      runs.value = data.runs ?? []
      total.value = data.total ?? 0
    } catch {
      runs.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  async function nextPage() {
    page.value++
    await loadRuns()
  }

  async function prevPage() {
    page.value = Math.max(1, page.value - 1)
    await loadRuns()
  }

  return {
    runs,
    total,
    page,
    loading,
    loadRuns,
    nextPage,
    prevPage,
  }
}
