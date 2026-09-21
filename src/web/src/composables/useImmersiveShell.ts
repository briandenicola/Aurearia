import { computed, onBeforeUnmount, onMounted, readonly, ref } from 'vue'

/**
 * Reference count of mounted full-bleed surfaces. Counted rather than a plain
 * boolean because a route change mounts the next page's shell before the
 * previous one unmounts - a boolean would be cleared by the outgoing shell and
 * leave the app chrome visible underneath the incoming one.
 */
const claimCount = ref(0)

const immersive = computed(() => claimCount.value > 0)

/**
 * Read-only view for app chrome (nav bar, floating agent button) that has to
 * step aside while a full-bleed surface owns the viewport.
 */
export function useImmersiveShell() {
  return { immersive: readonly(immersive) }
}

/**
 * Claimed by a component that covers the whole viewport - currently the PWA
 * coin capture shell. The claim is released automatically on unmount.
 */
export function useImmersiveShellClaim() {
  let claimed = false

  function claim() {
    if (claimed) return
    claimed = true
    claimCount.value += 1
  }

  function release() {
    if (!claimed) return
    claimed = false
    claimCount.value -= 1
  }

  onMounted(claim)
  onBeforeUnmount(release)

  return { claim, release }
}
