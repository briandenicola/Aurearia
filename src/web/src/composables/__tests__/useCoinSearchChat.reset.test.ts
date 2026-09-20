import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CoinCopilotRun } from '@/types'
import { useCoinSearchChat } from '../useCoinSearchChat'

const mocks = vi.hoisted(() => ({
  capability: vi.fn(), stream: vi.fn(), start: vi.fn(), events: vi.fn(), cancel: vi.fn(),
  confirm: vi.fn(), save: vi.fn(),
}))
vi.mock('vue-router', () => ({ useRoute: () => ({ params: {}, fullPath: '/collection' }) }))
vi.mock('@/api/client', () => ({
  getCoinCopilotCapability: mocks.capability, agentChatStream: mocks.stream,
  startCoinCopilotRun: mocks.start, streamCoinCopilotRunEvents: mocks.events,
  cancelCoinCopilotRun: mocks.cancel, saveConversation: mocks.save,
  getAgentStatus: async () => ({ data: { configured: true } }),
  getApiErrorMessage: (error: unknown) => error instanceof Error ? error.message : '',
}))
vi.mock('@/composables/useCoinOptions', () => ({
  useCoinOptions: () => ({ categoryOptions: ref([]), eraOptions: ref([]), loadOptions: vi.fn() }),
}))
vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showAlert: vi.fn(), showConfirm: mocks.confirm }),
}))

function run(status: CoinCopilotRun['status']): CoinCopilotRun {
  return {
    id: 'old-run', threadId: 'old-thread', status, goal: 'Old goal', checkpointVersion: 0,
    lastSeq: 0, attempt: 1, finalAnswer: null, failureCode: null, failureMessage: null,
    resumeDeadline: null, usage: { iterations: 0, toolCalls: 0, inputTokens: 0, outputTokens: 0 },
    createdAt: '', updatedAt: '',
  }
}
function render(loadConversation?: { id: number; title: string; messages: string }) {
  let chat: ReturnType<typeof useCoinSearchChat>
  const wrapper = mount(defineComponent({
    setup() {
      chat = useCoinSearchChat({ messagesEl: ref(), inputBarEl: ref(), onAdded: vi.fn(), loadConversation })
      return () => null
    },
  }))
  return { chat: chat!, wrapper }
}

describe('New Chat', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
    mocks.capability.mockResolvedValue({
      data: { mode: 'legacy', enabled: false, modelToolCallingSupported: false, reason: 'disabled' },
    })
    mocks.confirm.mockResolvedValue(true)
    mocks.save.mockResolvedValue({ data: { id: 9 } })
  })

  it('resets saved-conversation identity and ignores old legacy stream callbacks', async () => {
    const { chat, wrapper } = render({ id: 8, title: 'Old', messages: '[{"role":"user","content":"Old"}]' })
    await flushPromises()
    chat.input.value = 'First'
    await chat.sendMessage()
    const callbacks = mocks.stream.mock.calls[0]!
    expect(await chat.newChat()).toBe(true)
    expect(mocks.confirm).toHaveBeenCalledOnce()
    expect(callbacks[7].aborted).toBe(true)
    chat.input.value = 'Fresh'
    await chat.sendMessage()
    callbacks[2]('Late text')
    callbacks[3]('Old answer', [])
    callbacks[4]('Late error')
    callbacks[5]('Late status')
    expect(chat.messages.value).toEqual([
      { role: 'user', content: 'Fresh' }, { role: 'assistant', content: '', streaming: true },
    ])
    expect(mocks.stream.mock.calls[1]?.[1]).toEqual([])
    expect(chat.conversationId.value).toBeNull()
    await chat.handleSave()
    expect(mocks.save.mock.calls[0]?.[0].id).toBeUndefined()
    wrapper.unmount()
  })

  it('preserves the conversation when cancellation is declined', async () => {
    const { chat, wrapper } = render()
    await flushPromises()
    chat.input.value = 'Working'
    await chat.sendMessage()
    mocks.confirm.mockResolvedValue(false)
    expect(await chat.newChat()).toBe(false)
    expect(chat.messages.value[0]?.content).toBe('Working')
    wrapper.unmount()
  })

  it('cancels an active Copilot run, clears its cursor, and rejects late events', async () => {
    mocks.capability.mockResolvedValue({ data: { mode: 'copilot', enabled: true, modelToolCallingSupported: true } })
    mocks.start.mockResolvedValue({ data: { run: run('running') } })
    let finishStream!: () => void
    mocks.events.mockImplementation(() => new Promise<void>(resolve => { finishStream = resolve }))
    mocks.cancel.mockResolvedValue({ data: { run: run('cancel_requested') } })
    const { chat, wrapper } = render()
    await flushPromises()
    chat.input.value = 'Working'
    const pending = chat.sendMessage()
    await flushPromises()
    const callbacks = mocks.events.mock.calls[0]?.[1]
    expect(await chat.newChat()).toBe(true)
    callbacks.onEnd({ status: 'completed' })
    callbacks.onTruncated()
    finishStream()
    await pending
    expect(mocks.cancel).toHaveBeenCalledWith('old-run')
    expect(chat.messages.value).toEqual([])
    expect(chat.copilotRun.value).toBeNull()
    expect(chat.copilotTruncated.value).toBe(false)
    expect(sessionStorage.getItem('coinCopilot:activeRun')).toBeNull()
    expect(sessionStorage.getItem('coinCopilot:lastThread')).toBeNull()

    mocks.start.mockResolvedValue({ data: { run: { ...run('completed'), id: 'fresh-run', threadId: 'fresh-thread' } } })
    chat.input.value = 'Fresh'
    await chat.sendMessage()
    expect(mocks.start.mock.calls[1]?.[0].threadId).toBeUndefined()
    wrapper.unmount()
  })

  it('does not clear a Copilot run when cancellation fails', async () => {
    mocks.capability.mockResolvedValue({ data: { mode: 'copilot', enabled: true, modelToolCallingSupported: true } })
    const { chat, wrapper } = render()
    await flushPromises()
    chat.copilotRun.value = run('paused')
    chat.messages.value = [{ role: 'user', content: 'Keep this' }]
    mocks.cancel.mockRejectedValue(new Error('Cannot cancel right now'))
    expect(await chat.newChat()).toBe(false)
    expect(chat.messages.value[0]?.content).toBe('Keep this')
    expect(chat.copilotError.value).toContain('Cannot cancel')
    wrapper.unmount()
  })
})
