import { computed, onMounted, onUnmounted, ref, type ComputedRef, type Ref } from 'vue'

const now = ref(Date.now())
let activeConsumers = 0
let intervalId: ReturnType<typeof setInterval> | null = null

function formatCountdown(targetMs: number, nowMs: number): string | null {
  const diff = targetMs - nowMs
  if (diff <= 0) return null
  const totalSeconds = Math.floor(diff / 1000)
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  if (days > 0) return `${days}d ${pad(hours)}h ${pad(minutes)}m ${pad(seconds)}s left`
  return `${pad(hours)}h ${pad(minutes)}m ${pad(seconds)}s left`
}

/**
 * Live-ticking "Xd HHh MMm SSs left" label for a target ISO date/time. Backs the countdown
 * onto a single shared per-second clock (ref-counted across mounted consumers) rather than one
 * setInterval per instance, since this is used inside auction lot grids with many cards.
 */
export function useCountdown(target: Ref<string | null | undefined> | ComputedRef<string | null | undefined>) {
  onMounted(() => {
    activeConsumers++
    now.value = Date.now()
    if (intervalId === null) {
      intervalId = setInterval(() => { now.value = Date.now() }, 1000)
    }
  })

  onUnmounted(() => {
    activeConsumers = Math.max(0, activeConsumers - 1)
    if (activeConsumers === 0 && intervalId !== null) {
      clearInterval(intervalId)
      intervalId = null
    }
  })

  const label = computed(() => {
    if (!target.value) return null
    const targetMs = new Date(target.value).getTime()
    if (Number.isNaN(targetMs)) return null
    return formatCountdown(targetMs, now.value)
  })

  return { label }
}
