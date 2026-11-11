import { onScopeDispose, ref, watch, type Ref } from 'vue'
import { getApiErrorMessage, resolvePaperlessInvoiceTask } from '@/api/client'

export const PAPERLESS_WATCH_INTERVAL_MS = 4000
// Matches the server's 10-minute background watch.
export const PAPERLESS_WATCH_MAX_ATTEMPTS = 150

/**
 * Watches a coin's pending paperless-ngx upload from any page that shows the
 * coin. The server links the coin on its own; this only asks it to re-check so
 * the page updates without the user having to open the Edit page. `onSettled`
 * runs once the task is linked or has failed, so the caller can reload the coin.
 */
export function usePaperlessTaskWatch(
  coinId: Ref<number | null | undefined>,
  taskId: Ref<string | null | undefined>,
  enabled: Ref<boolean>,
  onSettled: () => void | Promise<void>,
) {
  const error = ref('')
  let run = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  function stop() {
    run++
    if (timer) clearTimeout(timer)
    timer = undefined
  }

  function start(id: number, task: string) {
    stop()
    const current = run
    let attempts = 0
    const tick = async () => {
      if (current !== run) return
      attempts++
      try {
        const res = await resolvePaperlessInvoiceTask(id, task)
        if (current !== run) return
        if (res.data.status === 'linked') {
          await onSettled()
          return
        }
      } catch (err) {
        if (current !== run) return
        error.value = getApiErrorMessage(err) || 'paperless-ngx could not process the invoice.'
        await onSettled()
        return
      }
      if (attempts < PAPERLESS_WATCH_MAX_ATTEMPTS) timer = setTimeout(tick, PAPERLESS_WATCH_INTERVAL_MS)
    }
    timer = setTimeout(tick, PAPERLESS_WATCH_INTERVAL_MS)
  }

  watch(
    [coinId, taskId, enabled],
    ([id, task, on]) => {
      if (on && id && task) start(id, task)
      else stop()
    },
    { immediate: true },
  )

  onScopeDispose(stop)

  return { error }
}
