import { describe, expect, it, vi } from 'vitest'
import { useDeepIdentification } from '../useDeepIdentification'
import { createDeepIdentificationJob, getDeepIdentificationJob } from '@/api/client'

vi.mock('@/api/client', () => ({
  createDeepIdentificationJob: vi.fn(),
  getDeepIdentificationJob: vi.fn(),
  getApiErrorMessage: (error: unknown) => {
    if (typeof error === 'object' && error !== null) {
      const maybeResponse = error as { response?: { data?: { error?: unknown } } }
      const message = maybeResponse.response?.data?.error
      if (typeof message === 'string') return message
      return error instanceof Error ? error.message : ''
    }
    return error instanceof Error ? error.message : String(error)
  },
  getApiErrorCode: (error: unknown) => {
    if (typeof error === 'object' && error !== null) {
      const maybeResponse = error as { response?: { data?: { code?: unknown } } }
      const code = maybeResponse.response?.data?.code
      if (typeof code === 'string') return code
    }
    return ''
  },
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

  it('surfaces the job_at_capacity conflict as a specific actionable message, code, and non-loading state', async () => {
    vi.mocked(createDeepIdentificationJob).mockRejectedValue({
      response: {
        status: 409,
        data: { error: 'An analysis is already running. Wait for it to finish or cancel it.', code: 'job_at_capacity' },
      },
    })
    const { start, error, errorCode, starting, job } = useDeepIdentification()

    const result = await start({ obverseImage: null, reverseImage: null })

    // The failed submission must not look like success: no job stored, the
    // spinner state cleared, and a message the user can act on (not a
    // generic "something went wrong").
    expect(result).toBeNull()
    expect(job.value).toBeNull()
    expect(starting.value).toBe(false)
    expect(errorCode.value).toBe('job_at_capacity')
    expect(error.value).toBe('An analysis is already running. Wait for it to finish or cancel it.')
  })

  it('falls back to a local job_at_capacity message when the server sends no message text', async () => {
    vi.mocked(createDeepIdentificationJob).mockRejectedValue({
      response: { status: 409, data: { code: 'job_at_capacity' } },
    })
    const { start, error, errorCode } = useDeepIdentification()

    await start({ obverseImage: null, reverseImage: null })

    expect(errorCode.value).toBe('job_at_capacity')
    expect(error.value).toBe('An analysis is already running. Wait for it to finish or cancel it.')
  })

  it('refreshes an existing job by id', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({ data: { job: { ...baseJob, status: 'running' } } } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    const { refresh, job } = useDeepIdentification()

    const envelope = await refresh(7)

    expect(envelope?.job.status).toBe('running')
    expect(job.value?.status).toBe('running')
  })
})
