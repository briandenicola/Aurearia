// agent endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api, refreshAccessToken, formatAgentServiceError } from '@/api/http'
import type {
  AgentChatAppContext,
  AgentChatMessage,
  ApplyDeepIdentificationProposalInput,
  CoinSuggestion,
  CollectionChatResponse,
  CreateDeepIdentificationJobInput,
  DeepApplyResult,
  DeepIdentificationCapability,
  DeepJobEnvelope,
  DeepJobListResponse,
  DeepProposal,
  DeepProviderId,
  ListDeepIdentificationJobsParams,
  PortfolioSummary,
  UpdateDeepIdentificationProposalInput,
} from '@/types'
import { appendOptionalFormValue } from '@/api/endpoints/_shared'

export const getValuationPrompt = () => api.get<{ prompt: string; default: string }>('/agent/valuation-prompt')

export const getPortfolioSummary = () => api.get<PortfolioSummary>('/agent/portfolio-summary')

// Agent

export async function agentChatStream(
  message: string,
  history: AgentChatMessage[],
  onText: (text: string) => void,
  onDone: (message: string, suggestions: CoinSuggestion[], collection?: CollectionChatResponse) => void,
  onError: (error: string) => void,
  onStatus?: (status: string) => void,
  appContext?: AgentChatAppContext,
) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''

  async function fetchWithAuthRetry(url: string, init: RequestInit): Promise<Response> {
    const firstHeaders = new Headers(init.headers ?? {})
    const token = localStorage.getItem('token')
    if (token) {
      firstHeaders.set('Authorization', `Bearer ${token}`)
    }

    const firstResp = await fetch(url, { ...init, headers: firstHeaders })
    if (firstResp.status !== 401) {
      return firstResp
    }

    const refreshedToken = await refreshAccessToken()
    const retryHeaders = new Headers(init.headers ?? {})
    retryHeaders.set('Authorization', `Bearer ${refreshedToken}`)
    return fetch(url, { ...init, headers: retryHeaders })
  }

  try {
    const resp = await fetchWithAuthRetry(`${baseURL}/api/agent/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ message, history, appContext }),
    })

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: `HTTP ${resp.status}` }))
      onError(formatAgentServiceError(err, `HTTP ${resp.status}`))
      return
    }

    const reader = resp.body?.getReader()
    if (!reader) { onError('No response body'); return }

    const decoder = new TextDecoder()
    let buffer = ''
    let accumulatedText = ''
    let terminalSent = false

    const sendDone = (
      finalMessage?: string,
      suggestions?: CoinSuggestion[],
      collection?: CollectionChatResponse,
    ) => {
      if (terminalSent) return
      terminalSent = true
      onDone(finalMessage || accumulatedText, Array.isArray(suggestions) ? suggestions : [], collection)
    }

    const sendError = (message: string) => {
      if (terminalSent) return
      terminalSent = true
      onError(message)
    }

    const handleDataLine = (line: string) => {
      if (!line.startsWith('data:')) return
      const data = line.replace(/^data:\s*/, '').trim()
      if (!data || data === '[DONE]') return

      try {
        const event = JSON.parse(data)
        if (event.type === 'text' && typeof event.text === 'string') {
          accumulatedText += event.text
          onText(event.text)
        } else if (event.type === 'status' && typeof event.message === 'string') {
          onStatus?.(event.message)
        } else if (event.type === 'done') {
          sendDone(
            typeof event.message === 'string' ? event.message : undefined,
            event.suggestions,
            event.collection,
          )
        } else if (event.type === 'error') {
          sendError(formatAgentServiceError(
            typeof event.message === 'string' ? event.message : '',
            'Agent stream error',
          ))
        }
      } catch {
        // Ignore malformed stream chunks.
      }
    }

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        handleDataLine(line)
      }
    }

    buffer += decoder.decode()
    if (buffer.trim()) {
      handleDataLine(buffer.trim())
    }

    if (!terminalSent) {
      if (accumulatedText.trim()) {
        sendDone(accumulatedText, [])
      } else {
        sendError('Stream ended unexpectedly')
      }
    }
  } catch (err: unknown) {
    onError(err instanceof Error ? err.message : 'Stream failed')
  }
}

export const commitCollectionProposal = (proposalId: string, proposalToken: string) =>
  api.post(`/agent/collection/proposals/${proposalId}/commit`, {
    proposalToken,
    confirm: true,
  })

export const cancelCollectionProposal = (proposalId: string) =>
  api.post(`/agent/collection/proposals/${proposalId}/cancel`, {})

export interface AnthropicModel {
  id: string
  name: string
}

export const getAnthropicModels = () => api.get<AnthropicModel[]>('/agent/models')

export const getCoinSearchPrompt = () => api.get<{ prompt: string; default: string }>('/agent/coin-search-prompt')

export const getCoinShowsPrompt = () => api.get<{ prompt: string; default: string }>('/agent/coin-shows-prompt')

// Agent Conversations
export interface ConversationSummary {
  id: number
  title: string
  createdAt: string
  updatedAt: string
}

export interface SavedConversation {
  id: number
  userId: number
  title: string
  messages: string
  createdAt: string
  updatedAt: string
}

export const listConversations = () => api.get<ConversationSummary[]>('/agent/conversations')

export const getConversation = (id: number) => api.get<SavedConversation>(`/agent/conversations/${id}`)

export const saveConversation = (data: { id?: number; title: string; messages: string }) =>
  api.post<SavedConversation>('/agent/conversations', data)

export const deleteConversation = (id: number) => api.delete(`/agent/conversations/${id}`)

export const getOllamaStatus = () =>
  api.get<{ available: boolean; model: string; url: string; message: string }>('/ollama-status')

export const getAIStatus = () =>
  api.get<{ available: boolean; provider: string; model: string; message: string }>('/ai-status')

// Agent status
export const getAgentStatus = () =>
  api.get<{ provider: string; configured: boolean }>('/agent/status')

// Deep Agentic Coin Identification (344-deep-agentic-coin-identification).
export async function createDeepIdentificationJob(input: CreateDeepIdentificationJobInput) {
  const formData = new FormData()
  if (input.coinId !== undefined) formData.append('coinId', String(input.coinId))
  if (input.obverseImage) formData.append('obverse', input.obverseImage)
  if (input.reverseImage) formData.append('reverse', input.reverseImage)
  for (const hint of input.hintImages ?? []) {
    formData.append('hints', hint)
  }
  appendOptionalFormValue(formData, 'notes', input.notes)
  if (input.providers && input.providers.length > 0) {
    formData.append('providers', input.providers.join(','))
  }
  return api.post<DeepJobEnvelope>('/deep-identification/jobs', formData)
}

export const listDeepIdentificationJobs = (params?: ListDeepIdentificationJobsParams) =>
  api.get<DeepJobListResponse>('/deep-identification/jobs', { params })

export const getDeepIdentificationCapability = () =>
  api.get<DeepIdentificationCapability>('/deep-identification/capability')

export const getDeepIdentificationJob = (id: number) =>
  api.get<DeepJobEnvelope>(`/deep-identification/jobs/${id}`)

export const cancelDeepIdentificationJob = (id: number) =>
  api.post<DeepJobEnvelope>(`/deep-identification/jobs/${id}/cancel`)

export const retryDeepIdentificationJob = (id: number, input?: { notes?: string; providers?: DeepProviderId[] }) =>
  api.post<DeepJobEnvelope>(`/deep-identification/jobs/${id}/retry`, {
    notes: input?.notes,
    providers: input?.providers,
  })

export const patchDeepIdentificationProposal = (id: number, input: UpdateDeepIdentificationProposalInput) =>
  api.patch<DeepProposal>(`/deep-identification/jobs/${id}/proposal`, input)

export const applyDeepIdentificationProposal = (id: number, input: ApplyDeepIdentificationProposalInput) =>
  api.post<DeepApplyResult>(`/deep-identification/jobs/${id}/apply`, input)

// Spec 354 T017/T044: hard-deletes a terminal job (204). Non-terminal → 409,
// non-owner/missing → 404 — surfaced to the caller via getApiErrorMessage.
export const deleteDeepIdentificationJob = (id: number) =>
  api.delete<void>(`/deep-identification/jobs/${id}`)

export default api

