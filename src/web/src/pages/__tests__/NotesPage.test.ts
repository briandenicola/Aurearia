import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import NotesPage from '../NotesPage.vue'
import { createNote, deleteNote, getNotes, updateNote } from '@/api/client'
import type { UserNote } from '@/types'

const dialogMocks = vi.hoisted(() => ({
  showConfirm: vi.fn(),
  showAlert: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  getNotes: vi.fn(),
  createNote: vi.fn(),
  updateNote: vi.fn(),
  deleteNote: vi.fn(),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => dialogMocks,
}))

function buildNote(overrides: Partial<UserNote> = {}): UserNote {
  return {
    id: 1,
    userId: 1,
    title: 'Research leads',
    body: '**Bold lead**\n\n<script>alert("x")</script>\n\n[Dealer](https://example.com)',
    createdAt: '2026-06-10T10:00:00Z',
    updatedAt: '2026-06-11T10:00:00Z',
    ...overrides,
  }
}

describe('NotesPage', () => {
  beforeEach(() => {
    vi.mocked(getNotes).mockReset()
    vi.mocked(createNote).mockReset()
    vi.mocked(updateNote).mockReset()
    vi.mocked(deleteNote).mockReset()
    dialogMocks.showConfirm.mockReset()
    dialogMocks.showAlert.mockReset()
  })

  it('lists notes and renders Markdown display safely', async () => {
    vi.mocked(getNotes).mockResolvedValue({ data: { notes: [buildNote()] } } as Awaited<ReturnType<typeof getNotes>>)

    const wrapper = mount(NotesPage)
    await flushPromises()

    expect(wrapper.text()).toContain('Research leads')
    expect(wrapper.find('.markdown-rendered strong').text()).toBe('Bold lead')
    expect(wrapper.find('.markdown-rendered a').attributes('href')).toBe('https://example.com')
    expect(wrapper.find('.markdown-rendered script').exists()).toBe(false)
  })

  it('creates a note from the empty state', async () => {
    vi.mocked(getNotes).mockResolvedValue({ data: { notes: [] } } as Awaited<ReturnType<typeof getNotes>>)
    vi.mocked(createNote).mockResolvedValue({
      data: buildNote({ id: 2, title: 'Show ideas', body: '- Visit Chicago show' }),
    } as Awaited<ReturnType<typeof createNote>>)

    const wrapper = mount(NotesPage)
    await flushPromises()

    await wrapper.find('.page-header .btn-primary').trigger('click')
    await wrapper.find('input[type="text"]').setValue('Show ideas')
    await wrapper.find('textarea').setValue('- Visit Chicago show')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(createNote).toHaveBeenCalledWith({ title: 'Show ideas', body: '- Visit Chicago show' })
    expect(wrapper.text()).toContain('Show ideas')
    expect(wrapper.find('form').exists()).toBe(false)
  })

  it('edits and deletes the selected note', async () => {
    vi.mocked(getNotes).mockResolvedValue({ data: { notes: [buildNote()] } } as Awaited<ReturnType<typeof getNotes>>)
    vi.mocked(updateNote).mockResolvedValue({
      data: buildNote({ title: 'Updated research', body: 'Saved update' }),
    } as Awaited<ReturnType<typeof updateNote>>)
    vi.mocked(deleteNote).mockResolvedValue({ data: {} } as Awaited<ReturnType<typeof deleteNote>>)
    dialogMocks.showConfirm.mockResolvedValue(true)

    const wrapper = mount(NotesPage)
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('Edit'))?.trigger('click')
    await wrapper.find('input[type="text"]').setValue('Updated research')
    await wrapper.find('textarea').setValue('Saved update')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(updateNote).toHaveBeenCalledWith(1, { title: 'Updated research', body: 'Saved update' })
    expect(wrapper.text()).toContain('Updated research')

    await wrapper.findAll('button').find(button => button.text().includes('Delete'))?.trigger('click')
    await flushPromises()

    expect(dialogMocks.showConfirm).toHaveBeenCalled()
    expect(deleteNote).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('Select or create a note')
  })
})
