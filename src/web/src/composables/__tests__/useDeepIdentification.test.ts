import { describe, expect, it, vi } from 'vitest'
import { useDeepIdentification } from '../useDeepIdentification'
import { createDeepIdentificationJob, getDeepIdentificationJob } from '@/api/client'

vi.mock('@/api/client', () => ({
  createDeepIdentificationJob: vi.fn(),
  getDeepIdentificationJob: vi.fn(),
  getApiErrorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

const baseJob = {
  id: 7,
  source: 'intake' as const,
  status: 'queued' as const,
  partialSuccess: false,
  cancelRequested: false,
  lastSeq: 0,
  eventsAvailable: false,
  expiresAt: '2030-01-01T00:00:00Z',
  createdAt: '2030-01-01T00:00:00Z',
}

describe('useDeepIdentification', () => {
  it('starts a job and stores it as current job', async () => {
    vi.mocked(createDeepIdentificationJob).mockResolvedValue({ data: { job: baseJob } } as Awaited<ReturnType<typeof createDeepIdentificationJob>>)
    const { start, job, starting } = useDeepIdentification()

    const promise = start({ obverseImage: null, reverseImage: null })
    expect(starting.value).toBe(true)
    const result = await promise

    expect(result?.id).toBe(7)
    expect(job.value?.id).toBe(7)
    expect(starting.value).toBe(false)
  })

  it('captures an error message when starting fails', async () => {
    vi.mocked(createDeepIdentificationJob).mockRejectedValue(new Error('boom'))
    const { start, error } = useDeepIdentification()

    const result = await start({ obverseImage: null, reverseImage: null })

    expect(result).toBeNull()
    expect(error.value).toBe('boom')
  })

  it('refreshes an existing job by id', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({ data: { job: { ...baseJob, status: 'running' } } } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    const { refresh, job } = useDeepIdentification()

    const envelope = await refresh(7)

    expect(envelope?.job.status).toBe('running')
    expect(job.value?.status).toBe('running')
  })
})
