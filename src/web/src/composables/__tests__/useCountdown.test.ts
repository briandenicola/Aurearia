import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { useCountdown } from '../useCountdown'

describe('useCountdown', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-03T00:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('formats a future target as "Xd HHh MMm SSs left"', () => {
    const target = ref('2026-08-12T04:04:53Z')
    let label!: ReturnType<typeof useCountdown>['label']
    const wrapper = mount(defineComponent({
      setup() {
        ;({ label } = useCountdown(target))
        return () => h('div')
      },
    }))

    expect(label.value).toBe('9d 04h 04m 53s left')
    wrapper.unmount()
  })

  it('drops the day segment once under a day remains', () => {
    const target = ref('2026-08-03T02:00:00Z')
    let label!: ReturnType<typeof useCountdown>['label']
    const wrapper = mount(defineComponent({
      setup() {
        ;({ label } = useCountdown(target))
        return () => h('div')
      },
    }))

    expect(label.value).toBe('02h 00m 00s left')
    wrapper.unmount()
  })

  it('returns null once the target has passed', () => {
    const target = ref('2020-01-01T00:00:00Z')
    let label!: ReturnType<typeof useCountdown>['label']
    const wrapper = mount(defineComponent({
      setup() {
        ;({ label } = useCountdown(target))
        return () => h('div')
      },
    }))

    expect(label.value).toBeNull()
    wrapper.unmount()
  })

  it('returns null when there is no target', () => {
    const target = ref<string | null>(null)
    let label!: ReturnType<typeof useCountdown>['label']
    const wrapper = mount(defineComponent({
      setup() {
        ;({ label } = useCountdown(target))
        return () => h('div')
      },
    }))

    expect(label.value).toBeNull()
    wrapper.unmount()
  })

  it('ticks down as real time advances', () => {
    const target = ref('2026-08-03T00:00:10Z')
    let label!: ReturnType<typeof useCountdown>['label']
    const wrapper = mount(defineComponent({
      setup() {
        ;({ label } = useCountdown(target))
        return () => h('div')
      },
    }))

    expect(label.value).toBe('00h 00m 10s left')
    vi.advanceTimersByTime(3000)
    expect(label.value).toBe('00h 00m 07s left')
    wrapper.unmount()
  })
})
