<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-[radial-gradient(ellipse_at_top,var(--bg-secondary)_0%,var(--bg-primary)_70%)]">
    <div class="w-full max-w-[420px] text-center card">
      <img
        :src="coinLogoSrc"
        alt="Aurearia - Coin Collection"
        class="w-20 h-20 rounded-full object-cover border border-gold-dim mb-4 shadow-[var(--shadow-glow)] mx-auto block"
      />
      <div
        class="w-12 h-12 mx-auto mb-4 flex items-center justify-center rounded-full border bg-input"
        :class="{
          'border-border-subtle text-gold': status === 'loading',
          'border-[var(--color-positive)] text-[var(--color-positive)]': status === 'success',
          'border-[var(--color-negative)] text-[var(--color-negative)]': status === 'error',
        }"
      >
        <LoaderCircle v-if="status === 'loading'" :size="28" aria-hidden="true" class="animate-spin" />
        <CheckCircle v-else-if="status === 'success'" :size="28" aria-hidden="true" />
        <AlertTriangle v-else :size="28" aria-hidden="true" />
      </div>

      <h1 class="mb-1">{{ title }}</h1>
      <p class="text-text-secondary mb-6 text-base">{{ subtitle }}</p>

      <div
        v-if="identity"
        class="grid gap-3 p-4 mb-4 border border-border-subtle rounded-sm bg-input text-left"
      >
        <div class="flex items-center justify-between gap-3">
          <span class="section-label">Provider</span>
          <strong class="text-gold text-base overflow-wrap-anywhere text-right">{{ identity.providerDisplayName }}</strong>
        </div>
        <div v-if="identity.email" class="flex items-center justify-between gap-3">
          <span class="section-label">Email</span>
          <strong class="text-gold text-base overflow-wrap-anywhere text-right">{{ identity.email }}</strong>
        </div>
      </div>

      <p
        v-if="message"
        class="text-body mb-4"
        :class="status === 'error' ? 'text-[var(--color-negative)]' : 'text-text-secondary'"
      >
        {{ message }}
      </p>

      <router-link
        class="btn btn-primary w-full justify-center gap-2 py-3"
        to="/settings?tab=account"
      >
        <ArrowLeft :size="18" aria-hidden="true" />
        Back to Account Settings
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { completeOIDCLinkCallback, getApiErrorMessage } from '@/api/client'
import type { OIDCLinkedIdentity } from '@/types'
import { AlertTriangle, ArrowLeft, CheckCircle, LoaderCircle } from 'lucide-vue-next'

type CallbackStatus = 'loading' | 'success' | 'error'

const route = useRoute()
const router = useRouter()

const status = ref<CallbackStatus>('loading')
const identity = ref<OIDCLinkedIdentity | null>(null)
const message = ref('')
const coinLogoSrc = '/coin-logo.jpg'

const title = computed(() => {
  if (status.value === 'loading') return 'Linking Sign-in Provider'
  if (status.value === 'success') return 'Provider Linked'
  return 'Linking Failed'
})

const subtitle = computed(() => {
  if (status.value === 'loading') return 'Finishing the secure provider handshake...'
  if (status.value === 'success') return 'Your external sign-in provider is now connected to this account.'
  return 'The provider could not be linked to your account.'
})

onMounted(() => {
  void completeCallback()
})

async function completeCallback() {
  const providerId = Number(firstParamValue(route.params.providerId))
  const code = firstQueryValue('code')
  const state = firstQueryValue('state')
  const providerError = firstQueryValue('error') || firstQueryValue('error_description')

  void router.replace({ name: 'oidc-link-callback', params: { providerId: route.params.providerId } })

  if (!Number.isInteger(providerId) || providerId <= 0) {
    setError('The provider callback was missing a valid provider. Start linking again from Account Settings.')
    return
  }

  if (providerError) {
    setError(mapProviderError(providerError))
    return
  }

  if (!code || !state) {
    setError('The provider callback was incomplete. Start linking again from Account Settings.')
    return
  }

  try {
    const response = await completeOIDCLinkCallback(providerId, code, state)
    identity.value = response.data.identity
    message.value = response.data.message || 'OIDC identity linked.'
    status.value = 'success'
  } catch (error: unknown) {
    setError(mapCallbackError(error))
  }
}

function setError(text: string) {
  status.value = 'error'
  message.value = text
}

function firstQueryValue(name: string) {
  const value = route.query[name]
  if (Array.isArray(value)) return value[0] ?? ''
  return value ?? ''
}

function firstParamValue(value: string | string[] | undefined) {
  if (Array.isArray(value)) return value[0] ?? ''
  return value
}

function mapProviderError(error: string) {
  const normalized = error.toLowerCase()
  if (normalized.includes('access_denied') || normalized.includes('cancel') || normalized.includes('denied')) {
    return 'Linking was cancelled or denied at the provider. You can try again from Account Settings.'
  }
  return 'The provider returned an error before linking completed. Try again or ask an administrator to review the provider setup.'
}

function mapCallbackError(error: unknown) {
  const response = getErrorResponse(error)
  const messageText = getApiErrorMessage(error)
  const detailText = getErrorDetail(error)
  const normalized = `${messageText} ${detailText}`.toLowerCase()

  if (response?.status === 409) {
    if (normalized.includes('another user') || normalized.includes('already linked')) {
      return 'This provider account is already linked to another user. Sign in with a different provider account or ask an administrator for help.'
    }
    return 'This provider account cannot be linked automatically. Sign in locally with the intended account, then try linking again.'
  }

  if (normalized.includes('redirect uri') || normalized.includes('client secret') || normalized.includes('configuration') || normalized.includes('discovery') || response?.status === 500) {
    return providerConfigurationMessage(detailText)
  }

  if (normalized.includes('state') || normalized.includes('claims') || response?.status === 400 || response?.status === 401) {
    return 'The provider response could not be validated. Start the linking flow again from Account Settings.'
  }

  return messageText || 'The provider could not be linked. Start the linking flow again from Account Settings.'
}

function getErrorResponse(error: unknown): { status?: number } | null {
  if (typeof error !== 'object' || error === null || !('response' in error)) return null
  const response = (error as { response?: unknown }).response
  if (typeof response !== 'object' || response === null) return null
  return response as { status?: number }
}

function getErrorDetail(error: unknown) {
  if (typeof error !== 'object' || error === null || !('response' in error)) return ''
  const response = (error as { response?: { data?: { detail?: unknown } } }).response
  const detail = response?.data?.detail
  return typeof detail === 'string' ? detail : ''
}

function providerConfigurationMessage(detail: string) {
  const safeDetail = detail.trim()
  if (safeDetail) {
    return `The sign-in provider is not configured correctly: ${safeDetail}. Ask an administrator to review the provider settings.`
  }
  return 'The sign-in provider is not configured correctly. Ask an administrator to review the provider settings.'
}
</script>
