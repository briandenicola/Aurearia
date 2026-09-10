import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EditCoinPage from '../EditCoinPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const editCoinPagePath = path.resolve(__dirname, '../EditCoinPage.vue')
const mocks = vi.hoisted(() => ({
  getCoin: vi.fn(),
  updateCoin: vi.fn(),
  uploadImage: vi.fn(),
  deleteImage: vi.fn(),
  extractText: vi.fn(),
  refresh: vi.fn(),
  back: vi.fn(),
  showAlert: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  getCoin: mocks.getCoin,
  updateCoin: mocks.updateCoin,
  uploadImage: mocks.uploadImage,
  deleteImage: mocks.deleteImage,
  extractText: mocks.extractText,
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '42' } }),
  useRouter: () => ({ back: mocks.back }),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showAlert: mocks.showAlert }),
}))

vi.mock('@/composables/useQuickAccess', () => ({
  useQuickAccess: () => ({ refresh: mocks.refresh }),
}))

describe('EditCoinPage', () => {
  beforeEach(() => {
    Object.values(mocks).forEach(mock => mock.mockReset())
    mocks.getCoin.mockResolvedValue({ data: buildRomanDenariusCore({ id: 42 }) })
    mocks.updateCoin.mockResolvedValue({ data: {} })
    mocks.refresh.mockResolvedValue(undefined)
  })

  it('does not clear legacy or custom era values when loading an existing coin', () => {
    const source = fs.readFileSync(editCoinPagePath, 'utf8')

    expect(source).not.toContain('COIN_ERAS')
    expect(source).not.toContain('form.era = \'\'')
  })

  it('keeps promoted Quick Capture coins on the existing edit and image-upload contract', () => {
    const source = fs.readFileSync(editCoinPagePath, 'utf8')

    expect(source).toContain('getCoin')
    expect(source).toContain('updateCoin')
    expect(source).toContain('uploadImage')
    expect(source).toContain('deleteImage')
    expect(source).not.toContain('getQuickCaptureDraft')
    expect(source).not.toContain('updateQuickCaptureDraft')
  })

  it('refreshes Quick Access once only after the replacement upload and deletion finish', async () => {
    let resolveUpload!: (value: { data: Record<string, never> }) => void
    let resolveDelete!: (value: { data: Record<string, never> }) => void
    mocks.uploadImage.mockReturnValue(new Promise(resolve => { resolveUpload = resolve }))
    mocks.deleteImage.mockReturnValue(new Promise(resolve => { resolveDelete = resolve }))
    const obverseFile = new File(['obverse'], 'obverse.jpg', { type: 'image/jpeg' })
    const CoinFormStub = defineComponent({
      emits: ['submit'],
      setup(_, { emit, expose }) {
        expose({
          obverseFile,
          reverseFile: null,
          cardFile: null,
          removedObverseId: 91,
          removedReverseId: null,
        })
        return () => h('button', { 'aria-label': 'Save coin', onClick: () => emit('submit') }, 'Save')
      },
    })
    const wrapper = mount(EditCoinPage, {
      global: { stubs: { CoinForm: CoinFormStub } },
    })
    await flushPromises()

    await wrapper.get('button[aria-label="Save coin"]').trigger('click')
    await flushPromises()
    expect(mocks.updateCoin).toHaveBeenCalledTimes(1)
    expect(mocks.uploadImage).toHaveBeenCalledWith(42, obverseFile, 'obverse', true)
    expect(mocks.refresh).not.toHaveBeenCalled()

    resolveUpload({ data: {} })
    await flushPromises()
    expect(mocks.deleteImage).toHaveBeenCalledWith(42, 91)
    expect(mocks.refresh).not.toHaveBeenCalled()

    resolveDelete({ data: {} })
    await flushPromises()
    expect(mocks.refresh).toHaveBeenCalledTimes(1)
    expect(mocks.back).toHaveBeenCalledTimes(1)
  })
})
