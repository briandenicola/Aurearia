import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineComponent, h, ref } from 'vue'
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

  it.each([false, true])('finishes replacement once and warns if cleanup is pending (%s)', async cleanupPending => {
    let resolveUpload!: (value: { data: Record<string, never> }) => void
    let resolveDelete!: (value: { data: { cleanupPending: boolean } }) => void
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

    resolveDelete({ data: { cleanupPending } })
    await flushPromises()
    expect(mocks.refresh).toHaveBeenCalledTimes(1)
    expect(mocks.back).toHaveBeenCalledTimes(1)
    expect(mocks.uploadImage).toHaveBeenCalledTimes(1)
    if (cleanupPending) expect(mocks.showAlert).toHaveBeenCalledWith(expect.stringContaining('Old image files are awaiting cleanup'), { title: 'Image cleanup pending' })
    else expect(mocks.showAlert).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it.each(['success', 'ocr-failure', 'upload-failure', 'refresh-failure'])('retains an original card and reports/retries failures (%s)', async outcome => {
    const card = new File(['original card'], 'original.jpg', { type: 'image/jpeg' })
    mocks.uploadImage.mockResolvedValue({ data: {} })
    mocks.extractText.mockResolvedValue({ data: { text: 'Dealer attribution' } })
    mocks.showAlert.mockResolvedValue(true)
    if (outcome === 'ocr-failure') mocks.extractText.mockRejectedValueOnce(new Error('OCR offline'))
    if (outcome === 'upload-failure') mocks.uploadImage.mockRejectedValueOnce(new Error('Upload interrupted'))
    if (outcome === 'refresh-failure') mocks.refresh.mockRejectedValueOnce(new Error('Refresh interrupted'))
    const errorLog = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const CoinFormStub = defineComponent({
      emits: ['submit'],
      setup(_, { emit, expose }) {
        expose({ cardFile: card })
        return () => h('button', { 'aria-label': 'Save coin', onClick: () => emit('submit') }, 'Save')
      },
    })
    const wrapper = mount(EditCoinPage, { global: { stubs: { CoinForm: CoinFormStub } } })
    await flushPromises()
    await wrapper.get('[aria-label="Save coin"]').trigger('click')
    await flushPromises()

    expect(mocks.uploadImage).toHaveBeenCalledWith(42, card, 'card', false)
    if (outcome === 'upload-failure' || outcome === 'refresh-failure') {
      expect(mocks.back).not.toHaveBeenCalled()
      expect(mocks.showAlert).toHaveBeenCalledWith(expect.stringContaining('Coin details were saved'), { title: 'Error' })
      if (outcome === 'upload-failure') expect(mocks.extractText).not.toHaveBeenCalled()
      await wrapper.get('[aria-label="Save coin"]').trigger('click')
      await flushPromises()
    }
    expect(mocks.uploadImage).toHaveBeenCalledTimes(outcome === 'upload-failure' ? 2 : 1)
    expect(mocks.back).toHaveBeenCalledTimes(1)
    if (outcome === 'ocr-failure') {
      expect(mocks.showAlert).toHaveBeenCalledWith(expect.stringContaining('original attribution card was saved privately'), { title: 'Card notes unavailable' })
      expect(errorLog).toHaveBeenCalled()
    } else {
      expect(mocks.updateCoin).toHaveBeenCalledWith(42, { notes: expect.stringContaining('--- Store Card ---\nDealer attribution') })
    }
    expect(mocks.deleteImage).not.toHaveBeenCalled()
    errorLog.mockRestore()
    wrapper.unmount()
  })

  it.each([false, true])('resumes a failed card after both face replacements without repeating completed changes (cleanup pending: %s)', async cleanupPending => {
    const obverseFile = new File(['obverse'], 'obverse.jpg', { type: 'image/jpeg' })
    const reverseFile = new File(['reverse'], 'reverse.jpg', { type: 'image/jpeg' })
    const cardFile = new File(['card'], 'card.jpg', { type: 'image/jpeg' })
    let cardFailed = false
    mocks.uploadImage.mockImplementation(async (_coinId: number, _file: File, imageType: string) => {
      if (imageType === 'card' && !cardFailed) {
        cardFailed = true
        throw new Error('Card interrupted')
      }
      return { data: { id: imageType === 'obverse' ? 101 : 102 } }
    })
    const deleted = new Set<number>()
    mocks.deleteImage.mockImplementation(async (_coinId: number, id: number) => {
      if (deleted.has(id)) throw new Error('Image not found')
      deleted.add(id)
      return { data: { cleanupPending } }
    })
    mocks.extractText.mockResolvedValue({ data: { text: 'Dealer attribution' } })
    const CoinFormStub = defineComponent({
      emits: ['submit'],
      setup(_, { emit, expose }) {
        expose({ obverseFile, reverseFile, cardFile, removedObverseId: 91, removedReverseId: 92 })
        return () => h('button', { onClick: () => emit('submit') }, 'Save')
      },
    })
    const wrapper = mount(EditCoinPage, { global: { stubs: { CoinForm: CoinFormStub } } })
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(mocks.back).not.toHaveBeenCalled()
    expect(mocks.deleteImage.mock.calls.map(call => call[1])).toEqual([91, 92])

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mocks.uploadImage.mock.calls.map(call => call[2])).toEqual(['obverse', 'reverse', 'card', 'card'])
    expect(mocks.deleteImage.mock.calls.map(call => call[1])).toEqual([91, 92])
    expect(mocks.extractText).toHaveBeenCalledExactlyOnceWith(cardFile)
    expect(mocks.back).toHaveBeenCalledTimes(1)
    if (cleanupPending) expect(mocks.showAlert).toHaveBeenCalledWith(expect.stringContaining('Old image files are awaiting cleanup'), { title: 'Image cleanup pending' })
    wrapper.unmount()
  })

  it('resumes a failed superseded-image deletion without uploading the replacement twice', async () => {
    const obverseFile = new File(['obverse'], 'obverse.jpg', { type: 'image/jpeg' })
    const cardFile = new File(['card'], 'card.jpg', { type: 'image/jpeg' })
    mocks.uploadImage.mockResolvedValue({ data: { id: 101 } })
    mocks.deleteImage.mockRejectedValueOnce(new Error('Cleanup interrupted')).mockResolvedValue({ data: {} })
    mocks.extractText.mockResolvedValue({ data: { text: '' } })
    const CoinFormStub = defineComponent({
      emits: ['submit'],
      setup(_, { emit, expose }) {
        expose({ obverseFile, cardFile, removedObverseId: 91 })
        return () => h('button', { onClick: () => emit('submit') }, 'Save')
      },
    })
    const wrapper = mount(EditCoinPage, { global: { stubs: { CoinForm: CoinFormStub } } })
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(mocks.uploadImage.mock.calls.map(call => call[2])).toEqual(['obverse'])

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mocks.uploadImage.mock.calls.map(call => call[2])).toEqual(['obverse', 'card'])
    expect(mocks.deleteImage).toHaveBeenCalledTimes(2)
    expect(mocks.back).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('does not repeat a completed face removal when a card is retried', async () => {
    const cardFile = new File(['card'], 'card.jpg', { type: 'image/jpeg' })
    mocks.uploadImage.mockRejectedValueOnce(new Error('Card interrupted')).mockResolvedValue({ data: { id: 101 } })
    mocks.deleteImage.mockResolvedValueOnce({ data: {} }).mockRejectedValue(new Error('Image not found'))
    mocks.extractText.mockResolvedValue({ data: { text: '' } })
    const CoinFormStub = defineComponent({
      emits: ['submit'],
      setup(_, { emit, expose }) {
        expose({ cardFile, removedObverseId: 91 })
        return () => h('button', { onClick: () => emit('submit') }, 'Save')
      },
    })
    const wrapper = mount(EditCoinPage, { global: { stubs: { CoinForm: CoinFormStub } } })
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mocks.deleteImage).toHaveBeenCalledExactlyOnceWith(42, 91)
    expect(mocks.uploadImage).toHaveBeenCalledTimes(2)
    expect(mocks.back).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it.each(['replace', 'remove'])('uses the saved replacement ID if the face changes after a card failure (%s)', async action => {
    const firstFile = new File(['first'], 'first.jpg', { type: 'image/jpeg' })
    const secondFile = new File(['second'], 'second.jpg', { type: 'image/jpeg' })
    const selected = ref<File | null>(firstFile)
    const cardFile = new File(['card'], 'card.jpg', { type: 'image/jpeg' })
    let cardFailed = false
    let nextImageId = 101
    mocks.uploadImage.mockImplementation(async (_coinId: number, _file: File, imageType: string) => {
      if (imageType === 'card' && !cardFailed) {
        cardFailed = true
        throw new Error('Card interrupted')
      }
      return { data: { id: nextImageId++ } }
    })
    mocks.deleteImage.mockResolvedValue({ data: {} })
    mocks.extractText.mockResolvedValue({ data: { text: '' } })
    const CoinFormStub = defineComponent({
      emits: ['submit'],
      setup(_, { emit, expose }) {
        expose({ obverseFile: selected, cardFile, removedObverseId: 91 })
        return () => h('button', { onClick: () => emit('submit') }, 'Save')
      },
    })
    const wrapper = mount(EditCoinPage, { global: { stubs: { CoinForm: CoinFormStub } } })
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    selected.value = action === 'replace' ? secondFile : null
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mocks.deleteImage.mock.calls.map(call => call[1])).toEqual([91, 101])
    expect(mocks.uploadImage.mock.calls.map(call => call[2])).toEqual(action === 'replace'
      ? ['obverse', 'card', 'obverse', 'card']
      : ['obverse', 'card', 'card'])
    expect(mocks.back).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it.each(['obverse', 'reverse'])('removes a newly uploaded %s after card failure without an original removed-image ID', async side => {
    mocks.getCoin.mockResolvedValue({ data: buildRomanDenariusCore({ id: 42, images: [] }) })
    const selected = ref<File | null>(new File(['face'], `${side}.jpg`, { type: 'image/jpeg' }))
    const cardFile = new File(['card'], 'card.jpg', { type: 'image/jpeg' })
    let cardFailed = false
    mocks.uploadImage.mockImplementation(async (_coinId: number, _file: File, imageType: string) => {
      if (imageType === 'card' && !cardFailed) {
        cardFailed = true
        throw new Error('Card interrupted')
      }
      return { data: { id: 101 } }
    })
    mocks.deleteImage.mockResolvedValue({ data: {} })
    mocks.extractText.mockResolvedValue({ data: { text: '' } })
    const CoinFormStub = defineComponent({
      emits: ['submit'],
      setup(_, { emit, expose }) {
        expose({
          obverseFile: side === 'obverse' ? selected : null,
          reverseFile: side === 'reverse' ? selected : null,
          cardFile, removedObverseId: null, removedReverseId: null,
        })
        return () => h('button', { onClick: () => emit('submit') }, 'Save')
      },
    })
    const wrapper = mount(EditCoinPage, { global: { stubs: { CoinForm: CoinFormStub } } })
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(mocks.deleteImage).not.toHaveBeenCalled()
    expect(mocks.back).not.toHaveBeenCalled()

    selected.value = null
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mocks.deleteImage).toHaveBeenCalledExactlyOnceWith(42, 101)
    expect(mocks.uploadImage.mock.calls.map(call => call[2])).toEqual([side, 'card', 'card'])
    expect(mocks.back).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})
