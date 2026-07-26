import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import CoinSearchChat from '../CoinSearchChat.vue'

const mockCreateNote = vi.fn()
const mockShowAlert = vi.fn()

vi.mock('@/api/client', () => ({
  createNote: (note: { title: string; body: string }) => mockCreateNote(note),
  getApiErrorMessage: () => '',
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showAlert: mockShowAlert }),
}))

vi.mock('@/composables/useCoinSearchChat', () => ({
  useCoinSearchChat: () => ({
    messages: ref([
      { role: 'user', content: 'Which coins are missing an era?' },
      { role: 'assistant', content: '# Missing Era Report\n\n- Coin #1 needs an era.' },
    ]),
    input: ref(''),
    loading: ref(false),
    addingIdx: ref(null),
    addedSet: ref(new Set()),
    savedShows: ref(new Set()),
    savingShow: ref(null),
    conversationId: ref(null),
    saving: ref(false),
    saveLabel: ref('Save'),
    providerConfigured: ref(true),
    categoryEraConfirmRequest: ref(null),
    chooseCategoryEraConfirmation: vi.fn(),
    cancelCategoryEraConfirmation: vi.fn(),
    sendMessage: vi.fn(),
    sendExample: vi.fn(),
    sendPortfolioAnalysis: vi.fn(),
    handleSave: vi.fn(),
    addToWishlist: vi.fn(),
    confirmCollectionProposal: vi.fn(),
    cancelCollectionProposalMessage: vi.fn(),
    pickDisambiguationCandidate: vi.fn(),
    formatMessage: (message: string) => message,
    isCoinShowResults: () => false,
    saveShowToCalendar: vi.fn(),
  }),
}))

describe('CoinSearchChat note saving', () => {
  it('lets the user explicitly save an assistant answer to Notes', async () => {
    mockCreateNote.mockResolvedValue({ data: { id: 7 } })
    mockShowAlert.mockResolvedValue(true)

    const wrapper = mount(CoinSearchChat, {
      global: {
        stubs: {
          CoinSuggestionGrid: true,
          CoinShowResultsGrid: true,
          CategoryEraConfirmModal: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Coin Agent')

    const saveToNotesButton = wrapper.findAll('button').find((button) => button.text() === 'Save to Notes')
    expect(saveToNotesButton).toBeTruthy()
    await saveToNotesButton!.trigger('click')
    expect(wrapper.text()).toContain('Review Note')
    expect((wrapper.get('#noteTitle').element as HTMLInputElement).value).toBe('Missing Era Report')

    await wrapper.get('#noteTitle').setValue('Collection cleanup')
    await wrapper.get('#noteBody').setValue('## Missing era\n\n- Coin #1 needs an era.')
    const noteForm = wrapper.findAll('form').at(-1)
    expect(noteForm).toBeTruthy()
    await noteForm!.trigger('submit')
    await flushPromises()

    expect(mockCreateNote).toHaveBeenCalledWith({
      title: 'Collection cleanup',
      body: '## Missing era\n\n- Coin #1 needs an era.',
    })
    expect(mockShowAlert).toHaveBeenCalledWith('The agenda result was saved to Notes.', { title: 'Note Created' })
  })
})
