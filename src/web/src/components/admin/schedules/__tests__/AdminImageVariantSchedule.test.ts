import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AdminImageVariantSchedule from '../AdminImageVariantSchedule.vue'
import type { AppSettings, ImageVariantRun } from '@/types'

const mocks = vi.hoisted(() => ({
  triggerImageVariantRun: vi.fn(),
  getImageVariantRuns: vi.fn(),
  getImageVariantStatus: vi.fn(),
  getApiErrorMessage: vi.fn((err: { response?: { data?: { error?: string } } }) => err?.response?.data?.error ?? ''),
}))

vi.mock('@/api/client', () => mocks)

function buildSettings(overrides: Partial<AppSettings> = {}): AppSettings {
  return {
    AIProvider: 'anthropic',
    OllamaURL: '',
    OllamaModel: '',
    ObversePrompt: '',
    ReversePrompt: '',
    TextExtractionPrompt: '',
    OllamaTimeout: '',
    SearXNGURL: '',
    LogLevel: 'info',
    ImageVariantCheckEnabled: 'true',
    ImageVariantCheckStartTime: '03:17',
    ...overrides,
  }
}

function run(overrides: Partial<ImageVariantRun> = {}): ImageVariantRun {
  return {
    id: 1,
    triggerType: 'scheduled',
    status: 'success',
    checked: 120,
    made: 14,
    failed: 0,
    durationMs: 8300,
    startedAt: '2026-11-18T03:17:00Z',
    completedAt: '2026-11-18T03:17:08Z',
    createdAt: '2026-11-18T03:17:00Z',
    ...overrides,
  }
}

function status(overrides: Record<string, unknown> = {}) {
  return { data: { name: 'Image Variants', enabled: true, isRunning: false, nextRunIn: 3_600_000_000_000 * 2.5, ...overrides } }
}

async function mountComponent(props: { settings?: AppSettings; settingsSaving?: boolean } = {}) {
  const wrapper = mount(AdminImageVariantSchedule, {
    props: { settings: buildSettings(), settingsSaving: false, ...props },
  })
  await flushPromises()
  return wrapper
}

function button(wrapper: Awaited<ReturnType<typeof mountComponent>>, text: string) {
  const found = wrapper.findAll('button').find(b => b.text() === text)
  if (!found) throw new Error(`no "${text}" button; buttons: ${wrapper.findAll('button').map(b => b.text()).join(', ')}`)
  return found
}

describe('AdminImageVariantSchedule', () => {
  beforeEach(() => {
    Object.values(mocks).forEach(mock => mock.mockClear())
    mocks.getImageVariantStatus.mockResolvedValue(status())
    mocks.getImageVariantRuns.mockResolvedValue({ data: { runs: [], total: 0, page: 1, limit: 5 } })
    mocks.triggerImageVariantRun.mockResolvedValue({ data: { message: 'Image variant pass started. Its result will appear in the run history when it finishes.' } })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('settings', () => {
    it('renders the heading, description and labelled controls', async () => {
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Image Variant Check')
      expect(wrapper.text()).toContain('Makes the small and medium copies')
      expect(wrapper.find('label[for="image-variant-check-enabled"]').text()).toContain('Enable Image Variant Check')
      expect(wrapper.find('#image-variant-check-enabled').attributes('type')).toBe('checkbox')
      const time = wrapper.find('#image-variant-check-start-time')
      expect(time.attributes('type')).toBe('time')
      expect(time.attributes('aria-describedby')).toBe('image-variant-check-start-time-hint')
      expect(wrapper.find('#image-variant-check-start-time-hint').exists()).toBe(true)
    })

    it('reflects and updates the enabled setting', async () => {
      const settings = buildSettings({ ImageVariantCheckEnabled: 'false' })
      const wrapper = await mountComponent({ settings })
      const checkbox = wrapper.find<HTMLInputElement>('#image-variant-check-enabled')
      expect(checkbox.element.checked).toBe(false)
      checkbox.element.checked = true
      await checkbox.trigger('change')
      expect(settings.ImageVariantCheckEnabled).toBe('true')
    })

    it('reflects and updates the start time', async () => {
      const settings = buildSettings({ ImageVariantCheckStartTime: '02:45' })
      const wrapper = await mountComponent({ settings })
      const input = wrapper.find<HTMLInputElement>('#image-variant-check-start-time')
      expect(input.element.value).toBe('02:45')
      await input.setValue('04:10')
      expect(settings.ImageVariantCheckStartTime).toBe('04:10')
    })

    it('emits save, and shows saving state', async () => {
      const wrapper = await mountComponent()
      await button(wrapper, 'Save Image Variant Settings').trigger('click')
      expect(wrapper.emitted('save')).toHaveLength(1)

      const saving = await mountComponent({ settingsSaving: true })
      expect(button(saving, 'Saving...').attributes('disabled')).toBeDefined()
    })

    it('reloads the server status after a save finishes', async () => {
      const wrapper = await mountComponent({ settingsSaving: true })
      mocks.getImageVariantStatus.mockClear()
      mocks.getImageVariantStatus.mockResolvedValue(status({ enabled: false }))
      await wrapper.setProps({ settingsSaving: false })
      await flushPromises()
      expect(mocks.getImageVariantStatus).toHaveBeenCalledTimes(1)
      expect(wrapper.text()).toContain('Server status: Disabled')
    })
  })

  describe('server status', () => {
    it('shows enabled with the time to the next run', async () => {
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Server status: Enabled — next run in 2h 30m')
    })

    it('shows a running pass and disables Run Now', async () => {
      mocks.getImageVariantStatus.mockResolvedValue(status({ isRunning: true }))
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Running a pass now')
      expect(button(wrapper, 'Running...').attributes('disabled')).toBeDefined()
    })

    it('says so when the status cannot be loaded', async () => {
      mocks.getImageVariantStatus.mockRejectedValue({ response: { data: { error: 'Forbidden' } } })
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Could not load the server status: Forbidden.')
    })
  })

  describe('run now', () => {
    it('starts a pass, shows the server message and refreshes the history', async () => {
      const wrapper = await mountComponent()
      mocks.getImageVariantRuns.mockClear()
      await button(wrapper, 'Run Now').trigger('click')
      await flushPromises()
      expect(mocks.triggerImageVariantRun).toHaveBeenCalledTimes(1)
      expect(wrapper.text()).toContain('Image variant pass started.')
      expect(mocks.getImageVariantRuns).toHaveBeenCalledWith(1, 5)
    })

    it('shows the server reason when a pass is already running', async () => {
      mocks.triggerImageVariantRun.mockRejectedValue({ response: { status: 409, data: { error: 'An image variant pass is already running.' } } })
      const wrapper = await mountComponent()
      await button(wrapper, 'Run Now').trigger('click')
      await flushPromises()
      const msg = wrapper.find('[role="status"]')
      expect(msg.text()).toBe('An image variant pass is already running.')
      expect(msg.classes()).toContain('text-loss')
    })

    it('ignores a second click while starting', async () => {
      let resolve: (v: unknown) => void = () => {}
      mocks.triggerImageVariantRun.mockReturnValue(new Promise(r => { resolve = r }))
      const wrapper = await mountComponent()
      await button(wrapper, 'Run Now').trigger('click')
      expect(button(wrapper, 'Starting...').attributes('disabled')).toBeDefined()
      await button(wrapper, 'Starting...').trigger('click')
      resolve({ data: { message: 'started' } })
      await flushPromises()
      expect(mocks.triggerImageVariantRun).toHaveBeenCalledTimes(1)
    })

    it('keeps refreshing while a pass runs and stops once it finishes', async () => {
      vi.useFakeTimers()
      mocks.getImageVariantStatus.mockResolvedValue(status({ isRunning: true }))
      mocks.getImageVariantRuns.mockResolvedValue({ data: { runs: [run({ status: 'running', triggerType: 'manual' })], total: 1 } })
      const wrapper = await mountComponent()
      expect(mocks.getImageVariantRuns).toHaveBeenCalledTimes(1)

      mocks.getImageVariantStatus.mockResolvedValue(status())
      mocks.getImageVariantRuns.mockResolvedValue({ data: { runs: [run({ triggerType: 'manual', made: 3 })], total: 1 } })
      await vi.advanceTimersByTimeAsync(5000)
      await flushPromises()
      expect(mocks.getImageVariantRuns).toHaveBeenCalledTimes(2)
      expect(wrapper.text()).toContain('success')

      await vi.advanceTimersByTimeAsync(15000)
      expect(mocks.getImageVariantRuns).toHaveBeenCalledTimes(2)
      wrapper.unmount()
    })

    it('stops refreshing when unmounted', async () => {
      vi.useFakeTimers()
      mocks.getImageVariantStatus.mockResolvedValue(status({ isRunning: true }))
      const wrapper = await mountComponent()
      wrapper.unmount()
      await vi.advanceTimersByTimeAsync(20000)
      expect(mocks.getImageVariantStatus).toHaveBeenCalledTimes(1)
    })
  })

  describe('run history', () => {
    it('says when there are no runs yet', async () => {
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('No image variant passes recorded yet.')
    })

    it('shows a failed load as an error, not as an empty history', async () => {
      mocks.getImageVariantRuns.mockRejectedValue({ response: { data: { error: 'Failed to list image variant runs' } } })
      const wrapper = await mountComponent()
      expect(wrapper.text()).toContain('Could not load the run history: Failed to list image variant runs.')
      expect(wrapper.text()).not.toContain('No image variant passes recorded yet.')
    })

    it('lists runs with counts, and the photos that could not be resized', async () => {
      mocks.getImageVariantRuns.mockResolvedValue({
        data: {
          runs: [
            run({ failed: 12, failures: 'image 7 (coin 3, coin-3/a.png): corrupt\nimage 9 (coin 4, coin-4/b.png): too large' }),
            run({ id: 2, status: 'stopped', errorMessage: 'The API stopped during this pass. The next pass carries on.', durationMs: 125000 }),
          ],
          total: 2,
        },
      })
      const wrapper = await mountComponent()
      const text = wrapper.text()
      expect(text).toContain('120')
      expect(text).toContain('8.3s')
      expect(text).toContain('2m 5s')
      expect(text).toContain('Could not resize (first 2 of 12):')
      expect(wrapper.findAll('li').map(li => li.text())).toEqual([
        'image 7 (coin 3, coin-3/a.png): corrupt',
        'image 9 (coin 4, coin-4/b.png): too large',
      ])
      expect(text).toContain('The API stopped during this pass.')
      expect(wrapper.find('.image-variant-run-detail td').attributes('colspan')).toBe('7')
    })

    it('pages through the history and disables Next on the last page', async () => {
      mocks.getImageVariantRuns.mockResolvedValue({ data: { runs: [run()], total: 7 } })
      const wrapper = await mountComponent()
      expect(button(wrapper, 'Prev').attributes('disabled')).toBeDefined()
      await button(wrapper, 'Next').trigger('click')
      await flushPromises()
      expect(mocks.getImageVariantRuns).toHaveBeenLastCalledWith(2, 5)
      expect(wrapper.text()).toContain('Page 2')
      expect(button(wrapper, 'Next').attributes('disabled')).toBeDefined()
    })

    it('keeps the newest history when an earlier request answers late', async () => {
      mocks.getImageVariantRuns.mockResolvedValue({ data: { runs: [run()], total: 7 } })
      const wrapper = await mountComponent()
      let slow: (v: unknown) => void = () => {}
      mocks.getImageVariantRuns
        .mockReturnValueOnce(new Promise(r => { slow = r }))
        .mockResolvedValueOnce({ data: { runs: [run({ id: 9, checked: 999 })], total: 7 } })
      // Next asks for page 2, which answers slowly; Run Now then reloads page 1, which answers at once.
      await button(wrapper, 'Next').trigger('click')
      await button(wrapper, 'Run Now').trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('999')
      slow({ data: { runs: [run({ id: 8, checked: 111 })], total: 7 } })
      await flushPromises()
      expect(wrapper.text()).toContain('999')
      expect(wrapper.text()).not.toContain('111')
    })
  })
})
