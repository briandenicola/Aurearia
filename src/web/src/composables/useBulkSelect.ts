import { ref } from 'vue'

const bulkSelectActive = ref(false)

export function useBulkSelect() {
  return { bulkSelectActive }
}
