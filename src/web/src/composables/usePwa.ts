const isPwa =
  (typeof window.matchMedia === 'function'
    && window.matchMedia('(display-mode: standalone)').matches)
  || (window.navigator as { standalone?: boolean }).standalone === true

export function usePwa() {
  return { isPwa }
}
