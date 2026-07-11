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

      <p
        v-if="message"
        class="text-body mb-4"
        :class="status === 'error' ? 'text-[var(--color-negative)]' : 'text-text-secondary'"
      >
        {{ message }}
      </p>

      <button
        v-if="status === 'success'"
        type="button"
        class="btn btn-primary w-full justify-center gap-2 py-3"
        @click="continueToApp"
      >
        <ArrowRight :size="18" aria-hidden="true" />
        Continue to Collection
      </button>
      <router-link
        v-else-if="status === 'error'"
        class="btn btn-secondary w-full justify-center gap-2 py-3"
        to="/login"
      >
        <ArrowLeft :size="18" aria-hidden="true" />
        Back to Sign In
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { completeOIDCLoginCallback, getApiErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { AlertTriangle, ArrowLeft, ArrowRight, CheckCircle, LoaderCircle } from 'lucide-vue-next'

type CallbackStatus = 'loading' | 'success' | 'error'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const status = ref<CallbackStatus>('loading')
const message = ref('')
const coinLogoSrc = '/coin-logo.jpg'

const title = computed(() => {
  if (status.value === 'loading') return 'Signing You In'
  if (status.value === 'success') return 'Sign In Complete'
  return 'Sign In Failed'
})

const subtitle = computed(() => {
  if (status.value === 'loading') return 'Finishing the secure provider handshake...'
  if (status.value === 'success') return 'Your secure session is ready.'
  return 'The provider sign-in could not be completed.'
})

onMounted(() => {
  void completeCallback()
})

async function completeCallback() {
  const providerIdParam = firstParamValue(route.params.providerId)
  const providerId = Number(providerIdParam)
  const code = firstQueryValue('code')
  const state = firstQueryValue('state')
  const providerError = firstQueryValue('error') || firstQueryValue('error_description')

  void router.replace({ name: 'oidc-login-callback', params: { providerId: providerIdParam ?? '' } })

  if (!Number.isInteger(providerId) || providerId <= 0) {
    setError('The provider callback was missing a valid provider. Start sign-in again.')
    return
  }

  if (providerError) {
    setError(mapProviderError(providerError))
    return
  }

  if (!code || !state) {
    setError('The provider callback was incomplete. Start sign-in again.')
    return
  }

  try {
    const response = await completeOIDCLoginCallback(providerId, code, state)
    await auth.applyAuthResponse(response.data)
    message.value = 'You are signed in. Continue to your collection.'
    status.value = 'success'
  } catch (error: unknown) {
    setError(mapCallbackError(error))
  }
}

function continueToApp() {
  void router.replace('/')
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
    return 'Sign-in was cancelled or denied at the provider. You can try again or use your local password.'
  }
  return 'The provider returned an error before sign-in completed. Try again or ask an administrator to review the provider setup.'
}

function mapCallbackError(error: unknown) {
  const response = getErrorResponse(error)
  const messageText = getApiErrorMessage(error)
  const detailText = getErrorDetail(error)
  const normalized = `${messageText} ${detailText}`.toLowerCase()

  if (response?.status === 409) {
    return 'This provider account matches an existing local account. Sign in locally, then link the provider from Account Settings.'
  }

  if (normalized.includes('redirect uri') || normalized.includes('client secret') || normalized.includes('configuration') || normalized.includes('discovery') || response?.status === 500) {
    return providerConfigurationMessage(detailText)
  }

  if (normalized.includes('state') || normalized.includes('claims') || response?.status === 400 || response?.status === 401) {
    return 'The provider response could not be validated. Start sign-in again.'
  }

  return messageText || 'OIDC sign-in failed. Try again or use your local password.'
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
