<template>
  <section class="card p-6">
    <div class="mb-3 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
      <div>
        <p class="section-label">Sign-in Providers</p>
        <h2 class="m-0 text-xl font-medium text-heading">OIDC Login</h2>
      </div>
      <div class="flex flex-col gap-2 md:flex-row md:flex-wrap md:justify-end">
        <button type="button" class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="openSetupGuide">
          Setup Guide
        </button>
        <button type="button" class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="openCreateForm">
          Add Provider
        </button>
      </div>
    </div>
    <p class="mb-6 text-body leading-6 text-text-secondary">
      Configure Microsoft Entra ID, Pocket ID, or another OpenID Connect provider. Client secrets are write-only and are never shown after saving.
    </p>

    <div v-if="loading" class="flex items-center justify-center p-12">
      <div class="spinner"></div>
    </div>

    <div
      v-else-if="loadError"
      class="flex items-start gap-2 rounded-sm border border-[color-mix(in_srgb,var(--color-negative)_30%,transparent)] bg-[color-mix(in_srgb,var(--color-negative)_10%,transparent)] p-4 text-body text-[var(--color-negative)]"
      role="alert"
    >
      <AlertCircle :size="18" />
      <span>{{ loadError }}</span>
    </div>

    <template v-else>
      <div v-if="providers.length === 0" class="py-8 text-center text-body text-text-muted">
        No OIDC providers configured yet.
      </div>

      <div v-else class="flex flex-col gap-3">
        <article v-for="provider in providers" :key="provider.id" class="grid gap-3 rounded-md border border-border-subtle bg-input p-4">
          <div class="min-w-0">
            <div class="mb-[0.35rem] flex flex-wrap items-center gap-[0.35rem]">
              <h3 class="m-0 text-lg font-medium text-heading">{{ provider.displayName }}</h3>
              <span class="chip-sm">{{ providerTypeLabel(provider.providerType) }}</span>
              <span class="chip-sm" :class="provider.enabled ? 'border-gold bg-[var(--accent-gold-dim)] text-gold' : 'text-text-muted'">
                {{ provider.enabled ? 'Enabled' : 'Disabled' }}
              </span>
            </div>
            <p class="mb-2 text-body text-text-secondary [overflow-wrap:anywhere]">{{ provider.name }} · {{ provider.issuerUrl }}</p>
            <div class="flex flex-wrap gap-x-4 gap-y-2 text-sm text-text-muted">
              <span>Client ID: {{ provider.clientId }}</span>
              <span>Secret: {{ provider.clientSecretConfigured ? 'Configured' : 'Not configured' }}</span>
              <span>Scopes: {{ provider.scopes?.join(' ') ?? 'openid profile email' }}</span>
            </div>
          </div>

          <div
            class="flex flex-col gap-[0.2rem] rounded-sm border bg-card p-[0.8rem] text-body"
            :class="{
              'border-[var(--color-positive)] bg-[color-mix(in_srgb,var(--color-positive)_14%,transparent)]': provider.lastTestStatus === 'ok',
              'border-[var(--color-negative)] bg-[color-mix(in_srgb,var(--color-negative)_14%,transparent)]': provider.lastTestStatus === 'failed',
              'border-border-subtle': provider.lastTestStatus === 'unknown',
            }"
          >
            <span class="font-semibold text-text-primary">{{ statusLabel(provider) }}</span>
            <span class="text-text-secondary">{{ statusMessage(provider) }}</span>
          </div>

          <div
            v-if="testResults[provider.id]"
            class="flex items-start gap-2 rounded-sm border p-[0.8rem] text-body"
            :class="testResults[provider.id]?.available
              ? 'border-[var(--color-positive)] bg-[color-mix(in_srgb,var(--color-positive)_14%,transparent)] text-[var(--color-positive)]'
              : 'border-[var(--color-negative)] bg-[color-mix(in_srgb,var(--color-negative)_14%,transparent)] text-[var(--color-negative)]'"
          >
            <CheckCircle v-if="testResults[provider.id]?.available" :size="16" />
            <AlertCircle v-else :size="16" />
            <div>
              <strong :class="testResults[provider.id]?.available ? 'text-[var(--color-positive)]' : 'text-[var(--color-negative)]'">
                {{ testResults[provider.id]?.available ? 'Discovery succeeded' : 'Discovery failed' }}
              </strong>
              <p class="mt-[0.15rem] text-text-secondary">{{ testResults[provider.id]?.message }}</p>
            </div>
          </div>

          <div class="flex flex-col gap-[0.35rem] md:flex-row md:flex-wrap md:justify-end">
            <button type="button" class="btn btn-secondary btn-xs focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="testingProviderId === provider.id" @click="testProvider(provider.id)">
              {{ testingProviderId === provider.id ? 'Testing discovery...' : 'Test Discovery' }}
            </button>
            <button type="button" class="btn btn-ghost btn-xs focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="savingProviderId === provider.id" @click="toggleProvider(provider)">
              {{ provider.enabled ? 'Disable' : 'Enable' }}
            </button>
            <button type="button" class="btn btn-ghost btn-xs focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="openEditForm(provider)">
              Edit
            </button>
            <button type="button" class="btn btn-danger btn-xs focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="deletingProviderId === provider.id" @click="deleteProvider(provider)">
              {{ deletingProviderId === provider.id ? 'Deleting...' : 'Delete' }}
            </button>
          </div>

          <div class="flex items-start gap-2 rounded-sm border border-border-accent bg-[var(--accent-gold-glow)] p-[0.8rem] text-sm text-text-secondary">
            <AlertCircle :size="16" />
            <span>Discovery tests do not validate the client secret. Entra verifies the secret only when a user completes sign-in or account linking.</span>
          </div>
        </article>
      </div>
    </template>

    <div v-if="showForm" class="fixed inset-0 z-[200] flex items-center justify-center bg-[rgba(0,0,0,0.6)] p-4" @click.self="closeForm">
      <div class="max-h-[90vh] w-full max-w-[760px] overflow-auto rounded-md border border-border-subtle bg-card shadow-[var(--shadow-card)]">
        <div class="flex items-center justify-between gap-4 border-b border-border-subtle px-6 py-4">
          <h3 class="m-0 text-lg font-medium text-heading">{{ editingProvider ? 'Edit OIDC Provider' : 'Add OIDC Provider' }}</h3>
          <button
            type="button"
            class="inline-flex items-center justify-center rounded-sm p-1 text-text-muted transition-colors hover:text-text-primary focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
            aria-label="Close"
            @click="closeForm"
          >
            <X :size="20" />
          </button>
        </div>

        <form class="p-6" @submit.prevent="saveProvider">
          <div class="grid gap-4 md:grid-cols-2">
            <div class="form-group">
              <label class="form-label" for="oidc-name">Provider Key</label>
              <input
                id="oidc-name"
                v-model.trim="form.name"
                class="form-input"
                required
                placeholder="entra-work"
                autocomplete="off"
              />
              <span class="mt-1 block text-sm text-text-muted">Stable admin key used by the API. Use lowercase letters, numbers, and hyphens.</span>
            </div>

            <div class="form-group">
              <label class="form-label" for="oidc-display-name">Display Name</label>
              <input
                id="oidc-display-name"
                v-model.trim="form.displayName"
                class="form-input"
                required
                placeholder="Microsoft"
                autocomplete="off"
              />
            </div>

            <div class="form-group">
              <label class="form-label" for="oidc-provider-type">Provider Type</label>
              <select id="oidc-provider-type" v-model="form.providerType" class="form-select">
                <option value="entra">Microsoft Entra ID</option>
                <option value="pocket_id">Pocket ID</option>
                <option value="generic">Generic OIDC</option>
              </select>
            </div>

            <div class="form-group flex items-center justify-between gap-4">
              <div>
                <label class="form-label" for="oidc-enabled">Enabled</label>
                <span class="mt-1 block text-sm text-text-muted">Only enabled providers appear on the login page.</span>
              </div>
              <label class="relative inline-block h-7 w-[50px] shrink-0 rounded-full focus-within:outline-2 focus-within:outline-gold focus-within:outline-offset-2">
                <input id="oidc-enabled" v-model="form.enabled" type="checkbox" class="peer sr-only" />
                <span class="absolute inset-0 rounded-full border border-border-subtle bg-input transition-colors peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] after:absolute after:bottom-[2px] after:left-[2px] after:h-[22px] after:w-[22px] after:rounded-full after:bg-text-secondary after:content-[''] after:transition-transform peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"></span>
              </label>
            </div>
          </div>

          <div v-if="form.providerType === 'entra'" class="form-group">
            <label class="form-label" for="oidc-tenant-id">Tenant ID</label>
            <input
              id="oidc-tenant-id"
              v-model.trim="form.tenantId"
              class="form-input"
              required
              placeholder="00000000-0000-0000-0000-000000000000"
              autocomplete="off"
            />
            <span class="mt-1 block text-sm text-text-muted">
              Derived issuer URL:
              <code v-if="derivedEntraIssuerUrl" class="font-mono text-sm text-text-primary [overflow-wrap:anywhere]">{{ derivedEntraIssuerUrl }}</code>
              <span v-else>enter a tenant ID to generate the Microsoft issuer URL.</span>
            </span>
          </div>

          <div v-else class="form-group">
            <label class="form-label" for="oidc-issuer-url">Issuer URL</label>
            <input
              id="oidc-issuer-url"
              v-model.trim="form.issuerUrl"
              class="form-input"
              required
              type="url"
              placeholder="https://login.microsoftonline.com/{tenant}/v2.0"
            />
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="form-group">
              <label class="form-label" for="oidc-client-id">Client ID</label>
              <input
                id="oidc-client-id"
                v-model.trim="form.clientId"
                class="form-input"
                required
                autocomplete="off"
              />
            </div>

            <div class="form-group">
              <label class="form-label" for="oidc-client-secret">Client Secret</label>
              <input
                id="oidc-client-secret"
                v-model="form.clientSecret"
                class="form-input"
                type="password"
                :placeholder="secretPlaceholder"
                autocomplete="new-password"
              />
              <span class="mt-1 block text-sm text-text-muted">{{ secretHint }}</span>
            </div>
          </div>

          <div class="mb-4 flex items-start gap-2 rounded-sm border border-border-accent bg-[var(--accent-gold-glow)] p-[0.8rem] text-sm text-text-secondary">
            <AlertCircle :size="16" />
            <span>Discovery tests do not validate the client secret. Entra verifies the secret only when a user completes sign-in or account linking.</span>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="form-group">
              <label class="form-label" for="oidc-scopes">Scopes</label>
              <input
                id="oidc-scopes"
                v-model.trim="form.scopesInput"
                class="form-input"
                required
                placeholder="openid profile email"
              />
              <span class="mt-1 block text-sm text-text-muted">Space or comma separated. Must include openid.</span>
            </div>

            <div class="form-group">
              <label class="form-label" for="oidc-callback-path">Callback Path</label>
              <input
                id="oidc-callback-path"
                v-model.trim="form.callbackPath"
                class="form-input"
                placeholder="/auth/oidc/callback/1"
                autocomplete="off"
              />
              <span class="mt-1 block text-sm text-text-muted">Login and linking use branded frontend callbacks. Leave blank unless you need a fallback override.</span>
            </div>
          </div>

          <div class="form-group flex items-center justify-between gap-4">
            <div>
              <label class="form-label" for="oidc-verified-email">Require Verified Email</label>
              <span class="mt-1 block text-sm text-text-muted">Recommended for matching account emails safely.</span>
            </div>
            <label class="relative inline-block h-7 w-[50px] shrink-0 rounded-full focus-within:outline-2 focus-within:outline-gold focus-within:outline-offset-2">
              <input id="oidc-verified-email" v-model="form.requireVerifiedEmail" type="checkbox" class="peer sr-only" />
              <span class="absolute inset-0 rounded-full border border-border-subtle bg-input transition-colors peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] after:absolute after:bottom-[2px] after:left-[2px] after:h-[22px] after:w-[22px] after:rounded-full after:bg-text-secondary after:content-[''] after:transition-transform peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"></span>
            </label>
          </div>

          <div
            v-if="formError"
            class="mb-4 rounded-sm border border-[color-mix(in_srgb,var(--color-negative)_30%,transparent)] bg-[color-mix(in_srgb,var(--color-negative)_10%,transparent)] p-3 text-body text-[var(--color-negative)]"
            role="alert"
          >
            <div class="flex items-center gap-2">
              <AlertCircle :size="16" />
              <span>{{ formError }}</span>
            </div>
          </div>

          <div class="mt-6 flex flex-col gap-2 md:flex-row md:justify-end">
            <button type="button" class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="closeForm">Cancel</button>
            <button type="submit" class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="formSaving">
              {{ formSaving ? 'Saving...' : 'Save Provider' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { AlertCircle, CheckCircle, X } from 'lucide-vue-next'
import {
  createAdminOIDCProvider,
  deleteAdminOIDCProvider,
  getAdminOIDCProviders,
  getApiErrorMessage,
  testAdminOIDCProvider,
  updateAdminOIDCProvider,
} from '@/api/client'
import type {
  OIDCAdminProvider,
  OIDCAdminProviderInput,
  OIDCAdminProviderUpdate,
  OIDCProviderTestResponse,
  OIDCProviderType,
} from '@/types'
import { useDialog } from '@/composables/useDialog'

type ProviderForm = {
  name: string
  displayName: string
  providerType: OIDCProviderType
  enabled: boolean
  tenantId: string
  issuerUrl: string
  clientId: string
  clientSecret: string
  scopesInput: string
  callbackPath: string
  requireVerifiedEmail: boolean
}

const REDACTED_SECRET_VALUES = new Set([
  'configured',
  'redacted',
  '[configured]',
  '[redacted]',
  '<configured>',
  '<redacted>',
  '********',
  '••••••••',
])

const { showAlert, showConfirm } = useDialog()
const router = useRouter()

const providers = ref<OIDCAdminProvider[]>([])
const loading = ref(true)
const loadError = ref('')
const showForm = ref(false)
const formSaving = ref(false)
const formError = ref('')
const editingProvider = ref<OIDCAdminProvider | null>(null)
const testingProviderId = ref<number | null>(null)
const savingProviderId = ref<number | null>(null)
const deletingProviderId = ref<number | null>(null)
const testResults = ref<Record<number, OIDCProviderTestResponse>>({})

const form = reactive<ProviderForm>({
  name: '',
  displayName: '',
  providerType: 'entra',
  enabled: false,
  tenantId: '',
  issuerUrl: '',
  clientId: '',
  clientSecret: '',
  scopesInput: 'openid profile email',
  callbackPath: '',
  requireVerifiedEmail: true,
})

const secretPlaceholder = computed(() =>
  editingProvider.value?.clientSecretConfigured
    ? 'Configured; leave blank to preserve'
    : 'Enter client secret'
)

const secretHint = computed(() =>
  editingProvider.value
    ? 'Leave blank to keep the existing secret. Enter a new value only when rotating the secret.'
    : 'Stored by the API and never returned to the browser.'
)

const derivedEntraIssuerUrl = computed(() => {
  const tenantId = normalizedTenantId()
  return tenantId ? `https://login.microsoftonline.com/${tenantId}/v2.0` : ''
})

async function loadProviders() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await getAdminOIDCProviders()
    providers.value = response.data.providers ?? []
  } catch (error: unknown) {
    loadError.value = getApiErrorMessage(error) || 'Failed to load OIDC providers'
  } finally {
    loading.value = false
  }
}

function providerTypeLabel(type: OIDCProviderType) {
  if (type === 'entra') return 'Entra ID'
  if (type === 'pocket_id') return 'Pocket ID'
  return 'Generic'
}

function statusLabel(provider: OIDCAdminProvider) {
  if (provider.lastTestStatus === 'ok') return 'Discovery passed'
  if (provider.lastTestStatus === 'failed') return 'Discovery failed'
  return 'Discovery not tested'
}

function statusMessage(provider: OIDCAdminProvider) {
  return provider.lastTestMessage || 'Run a discovery test to verify issuer metadata. Client secrets are verified only during sign-in or account linking.'
}

function statusClass(provider: OIDCAdminProvider) {
  return {
    success: provider.lastTestStatus === 'ok',
    error: provider.lastTestStatus === 'failed',
    unknown: provider.lastTestStatus === 'unknown',
  }
}

function resetForm() {
  form.name = ''
  form.displayName = ''
  form.providerType = 'entra'
  form.enabled = false
  form.tenantId = ''
  form.issuerUrl = ''
  form.clientId = ''
  form.clientSecret = ''
  form.scopesInput = 'openid profile email'
  form.callbackPath = ''
  form.requireVerifiedEmail = true
  formError.value = ''
}

function openCreateForm() {
  editingProvider.value = null
  resetForm()
  showForm.value = true
}

function openSetupGuide() {
  router.push({ path: '/settings', query: { tab: 'help', section: 'oidc' } })
}

function openEditForm(provider: OIDCAdminProvider) {
  editingProvider.value = provider
  form.name = provider.name
  form.displayName = provider.displayName
  form.providerType = provider.providerType
  form.enabled = provider.enabled
  form.tenantId = provider.providerType === 'entra' ? inferEntraTenantId(provider.issuerUrl) : ''
  form.issuerUrl = provider.issuerUrl
  form.clientId = provider.clientId
  form.clientSecret = ''
  form.scopesInput = provider.scopes?.join(' ') ?? 'openid profile email'
  form.callbackPath = provider.callbackPath ?? ''
  form.requireVerifiedEmail = provider.requireVerifiedEmail ?? true
  formError.value = ''
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editingProvider.value = null
  resetForm()
}

function parseScopes() {
  return form.scopesInput
    .split(/[\s,]+/)
    .map(scope => scope.trim())
    .filter(Boolean)
}

function sanitizedSecret() {
  const secret = form.clientSecret.trim()
  if (!secret) return ''
  if (REDACTED_SECRET_VALUES.has(secret.toLowerCase())) return ''
  return secret
}

function normalizedTenantId() {
  return form.tenantId.trim()
}

function inferEntraTenantId(issuerUrl: string) {
  try {
    const parsed = new URL(issuerUrl)
    const pathParts = parsed.pathname.split('/').filter(Boolean)
    const tenant = pathParts[0] ?? ''
    const version = pathParts[1] ?? ''
    if (parsed.hostname.toLowerCase() === 'login.microsoftonline.com' && tenant && version.toLowerCase() === 'v2.0') {
      return decodeURIComponent(tenant)
    }
  } catch {
    // Fall through to the regex parser for partial issuer strings.
  }

  return issuerUrl.match(/^https:\/\/login\.microsoftonline\.com\/([^/]+)\/v2\.0\/?$/i)?.[1] ?? ''
}

function issuerUrlForPayload() {
  if (form.providerType !== 'entra') {
    return form.issuerUrl
  }

  const tenantId = normalizedTenantId()
  if (!tenantId) {
    throw new Error('Tenant ID is required for Microsoft Entra ID.')
  }
  if (/[\s/\\]/.test(tenantId)) {
    throw new Error('Tenant ID must not contain spaces or slashes.')
  }

  return `https://login.microsoftonline.com/${tenantId}/v2.0`
}

function buildPayload(): OIDCAdminProviderInput {
  const scopes = parseScopes()
  if (!scopes.includes('openid')) {
    throw new Error('Scopes must include openid.')
  }

  const payload: OIDCAdminProviderInput = {
    name: form.name,
    displayName: form.displayName,
    providerType: form.providerType,
    enabled: form.enabled,
    issuerUrl: issuerUrlForPayload(),
    clientId: form.clientId,
    scopes,
    requireVerifiedEmail: form.requireVerifiedEmail,
  }

  if (form.callbackPath) {
    payload.callbackPath = form.callbackPath
  }

  const clientSecret = sanitizedSecret()
  if (clientSecret) {
    payload.clientSecret = clientSecret
  }

  return payload
}

async function saveProvider() {
  formSaving.value = true
  formError.value = ''
  try {
    const payload = buildPayload()
    if (editingProvider.value) {
      await updateAdminOIDCProvider(editingProvider.value.id, payload as OIDCAdminProviderUpdate)
    } else {
      await createAdminOIDCProvider(payload)
    }
    await loadProviders()
    closeForm()
  } catch (error: unknown) {
    formError.value = getApiErrorMessage(error) || (error instanceof Error ? error.message : 'Failed to save OIDC provider')
  } finally {
    formSaving.value = false
  }
}

async function toggleProvider(provider: OIDCAdminProvider) {
  savingProviderId.value = provider.id
  try {
    await updateAdminOIDCProvider(provider.id, { enabled: !provider.enabled })
    await loadProviders()
  } catch (error: unknown) {
    await showAlert(getApiErrorMessage(error) || 'Failed to update provider status', { title: 'Provider Update Failed' })
  } finally {
    savingProviderId.value = null
  }
}

async function testProvider(providerId: number) {
  testingProviderId.value = providerId
  try {
    const response = await testAdminOIDCProvider(providerId)
    testResults.value = {
      ...testResults.value,
      [providerId]: response.data,
    }
    await loadProviders()
  } catch (error: unknown) {
    testResults.value = {
      ...testResults.value,
      [providerId]: {
        available: false,
        message: getApiErrorMessage(error) || 'Provider discovery failed',
        issuer: '',
        authorizationEndpoint: '',
        tokenEndpoint: '',
      },
    }
  } finally {
    testingProviderId.value = null
  }
}

async function deleteProvider(provider: OIDCAdminProvider) {
  const confirmed = await showConfirm(
    `Delete ${provider.displayName}? This will fail if any user identities are linked to this provider.`,
    { title: 'Delete OIDC Provider', variant: 'danger' },
  )
  if (!confirmed) return

  deletingProviderId.value = provider.id
  try {
    await deleteAdminOIDCProvider(provider.id)
    await loadProviders()
  } catch (error: unknown) {
    await showAlert(getApiErrorMessage(error) || 'Failed to delete OIDC provider', { title: 'Delete Failed' })
  } finally {
    deletingProviderId.value = null
  }
}

onMounted(() => {
  void loadProviders()
})
</script>

