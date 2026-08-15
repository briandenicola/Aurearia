import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { useDeepIdentificationCapability } from '../useDeepIdentificationCapability'
import { getDeepIdentificationCapability } from '@/api/client'

vi.mock('@/api/client', () => ({
  getDeepIdentificationCapability: vi.fn(),
}))

function mountCapability() {
  const state: { enabled: boolean; loaded: boolean } = { enabled: false, loaded: false }
  const host = defineComponent({
    setup() {
      const cap = useDeepIdentificationCapability()
      return () => {
        state.enabled = cap.enabled.value
        state.loaded = cap.loaded.value
        return h('div')
      }
    },
  })
  const wrapper = mount(host)
  return { wrapper, state }
}

describe('useDeepIdentificationCapability', () => {
  it('enables the entry point when the backend reports enabled', async () => {
    vi.mocked(getDeepIdentificationCapability).mockResolvedValue({
      data: { enabled: true },
    } as Awaited<ReturnType<typeof getDeepIdentificationCapability>>)

    const { state } = mountCapability()
    await flushPromises()
    await nextTick()

    expect(state.enabled).toBe(true)
    expect(state.loaded).toBe(true)
  })

  it('hides the entry point when the backend reports disabled', async () => {
    vi.mocked(getDeepIdentificationCapability).mockResolvedValue({
      data: { enabled: false },
    } as Awaited<ReturnType<typeof getDeepIdentificationCapability>>)

    const { state } = mountCapability()
    await flushPromises()
    await nextTick()

    expect(state.enabled).toBe(false)
    expect(state.loaded).toBe(true)
  })

  it('fails closed when the capability probe is unavailable', async () => {
    vi.mocked(getDeepIdentificationCapability).mockRejectedValue(new Error('network down'))

    const { state } = mountCapability()
    await flushPromises()
    await nextTick()

    expect(state.enabled).toBe(false)
    expect(state.loaded).toBe(true)
  })
})
