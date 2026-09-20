import { describe, expect, it } from 'vitest'

import { validateIssuePublicationRequest } from '../contracts'
import { clone, validIssueRequest } from './contract-fixtures'

describe('issue publication contract', () => {
  it('accepts a bounded request with the exact fingerprint marker', () => {
    expect(validateIssuePublicationRequest(validIssueRequest())).toEqual({
      ok: true,
      errors: [],
    })
  })

  it.each([
    ['repository', (issue: Record<string, unknown>) => (issue.repository = 'not-a-repository')],
    ['title limit', (issue: Record<string, unknown>) => (issue.title = 'x'.repeat(161))],
    ['body limit', (issue: Record<string, unknown>) => (issue.body = 'x'.repeat(12001))],
    ['fixed labels', (issue: Record<string, unknown>) => (issue.labels = ['bug'])],
    [
      'exact marker',
      (issue: Record<string, unknown>) =>
        (issue.body =
          '<!-- aurearia-ai-finding:v1:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff -->'),
    ],
    ['unknown fields', (issue: Record<string, unknown>) => (issue.extra = true)],
    ['secret-free payloads', (issue: Record<string, unknown>) => (issue.body = 'token=ghp_exampleSecret123')],
  ])('rejects an invalid %s', (_name, mutate) => {
    const issue = clone(validIssueRequest()) as unknown as Record<string, unknown>
    mutate(issue)
    expect(validateIssuePublicationRequest(issue).ok).toBe(false)
  })
})
