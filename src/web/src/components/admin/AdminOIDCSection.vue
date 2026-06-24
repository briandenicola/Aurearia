<template>
  <section class="admin-section card">
    <div class="section-heading">
      <div>
        <p class="section-label">Sign-in Providers</p>
        <h2>OIDC Login</h2>
      </div>
      <div class="section-actions">
        <button type="button" class="btn btn-secondary btn-sm" @click="openSetupGuide">
          Setup Guide
        </button>
        <button type="button" class="btn btn-primary btn-sm" @click="openCreateForm">
          Add Provider
        </button>
      </div>
    </div>
    <p class="section-description">
      Configure Microsoft Entra ID, Pocket ID, or another OpenID Connect provider. Client secrets are write-only and are never shown after saving.
    </p>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="loadError" class="status-message error" role="alert">
      <AlertCircle :size="18" />
      <span>{{ loadError }}</span>
    </div>

    <template v-else>
      <div v-if="providers.length === 0" class="empty-state-copy">
        No OIDC providers configured yet.
      </div>

      <div v-else class="provider-list">
        <article v-for="provider in providers" :key="provider.id" class="provider-card">
          <div class="provider-summary">
            <div class="provider-title">
              <h3>{{ provider.displayName }}</h3>
              <span class="chip-sm">{{ providerTypeLabel(provider.providerType) }}</span>
              <span class="chip-sm" :class="provider.enabled ? 'enabled-chip' : 'disabled-chip'">
                {{ provider.enabled ? 'Enabled' : 'Disabled' }}
              </span>
            </div>
            <p class="provider-meta">{{ provider.name }} · {{ provider.issuerUrl }}</p>
            <div class="provider-details">
              <span>Client ID: {{ provider.clientId }}</span>
              <span>Secret: {{ provider.clientSecretConfigured ? 'Configured' : 'Not configured' }}</span>
              <span>Scopes: {{ provider.scopes?.join(' ') ?? 'openid profile email' }}</span>
            </div>
          </div>

          <div class="provider-status" :class="statusClass(provider)">
            <span class="status-label">{{ statusLabel(provider) }}</span>
            <span class="status-message-text">{{ statusMessage(provider) }}</span>
          </div>

          <div v-if="testResults[provider.id]" class="test-result" :class="{ success: testResults[provider.id]?.available, error: !testResults[provider.id]?.available }">
            <CheckCircle v-if="testResults[provider.id]?.available" :size="16" />
            <AlertCircle v-else :size="16" />
            <div>
              <strong>{{ testResults[provider.id]?.available ? 'Discovery succeeded' : 'Discovery failed' }}</strong>
              <p>{{ testResults[provider.id]?.message }}</p>
            </div>
          </div>

          <div class="provider-actions">
            <button type="button" class="btn btn-secondary btn-xs" :disabled="testingProviderId === provider.id" @click="testProvider(provider.id)">
              {{ testingProviderId === provider.id ? 'Testing...' : 'Test' }}
            </button>
            <button type="button" class="btn btn-ghost btn-xs" :disabled="savingProviderId === provider.id" @click="toggleProvider(provider)">
              {{ provider.enabled ? 'Disable' : 'Enable' }}
            </button>
            <button type="button" class="btn btn-ghost btn-xs" @click="openEditForm(provider)">
              Edit
            </button>
            <button type="button" class="btn btn-danger btn-xs" :disabled="deletingProviderId === provider.id" @click="deleteProvider(provider)">
              {{ deletingProviderId === provider.id ? 'Deleting...' : 'Delete' }}
            </button>
          </div>
        </article>
      </div>
    </template>

    <div v-if="showForm" class="modal-overlay" @click.self="closeForm">
      <div class="modal-content">
        <div class="modal-header">
          <h3>{{ editingProvider ? 'Edit OIDC Provider' : 'Add OIDC Provider' }}</h3>
          <button type="button" class="modal-close" aria-label="Close" @click="closeForm">
            <X :size="20" />
          </button>
        </div>

        <form class="modal-body" @submit.prevent="saveProvider">
          <div class="form-grid">
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
              <span class="form-hint">Stable admin key used by the API. Use lowercase letters, numbers, and hyphens.</span>
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

            <div class="form-group toggle-row">
              <div>
                <label class="form-label" for="oidc-enabled">Enabled</label>
                <span class="form-hint">Only enabled providers appear on the login page.</span>
              </div>
              <label class="toggle">
                <input id="oidc-enabled" v-model="form.enabled" type="checkbox" />
                <span class="toggle-slider"></span>
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
            <span class="form-hint">
              Derived issuer URL:
              <code v-if="derivedEntraIssuerUrl">{{ derivedEntraIssuerUrl }}</code>
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

          <div class="form-grid">
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
              <span class="form-hint">{{ secretHint }}</span>
            </div>
          </div>

          <div class="form-grid">
            <div class="form-group">
              <label class="form-label" for="oidc-scopes">Scopes</label>
              <input
                id="oidc-scopes"
                v-model.trim="form.scopesInput"
                class="form-input"
                required
                placeholder="openid profile email"
              />
              <span class="form-hint">Space or comma separated. Must include openid.</span>
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
              <span class="form-hint">Login and linking use branded frontend callbacks. Leave blank unless you need a fallback override.</span>
            </div>
          </div>

          <div class="form-group toggle-row">
            <div>
              <label class="form-label" for="oidc-verified-email">Require Verified Email</label>
              <span class="form-hint">Recommended for matching account emails safely.</span>
            </div>
            <label class="toggle">
              <input id="oidc-verified-email" v-model="form.requireVerifiedEmail" type="checkbox" />
              <span class="toggle-slider"></span>
            </label>
          </div>

          <div v-if="formError" class="status-message error form-error" role="alert">
            <AlertCircle :size="16" />
            <span>{{ formError }}</span>
          </div>

          <div class="modal-footer">
            <button type="button" class="btn btn-secondary btn-sm" @click="closeForm">Cancel</button>
            <button type="submit" class="btn btn-primary btn-sm" :disabled="formSaving">
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
  if (provider.lastTestStatus === 'ok') return 'Test passed'
  if (provider.lastTestStatus === 'failed') return 'Test failed'
  return 'Not tested'
}

function statusMessage(provider: OIDCAdminProvider) {
  return provider.lastTestMessage || 'Run a provider test to verify discovery metadata.'
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

<style scoped>
.admin-section {
  padding: 1.5rem;
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.section-heading h2 {
  margin: 0;
}

.section-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.5rem;
}

.section-description {
  color: var(--text-secondary);
  font-size: 0.85rem;
  line-height: 1.5;
  margin: 0 0 1.5rem;
}

.loading-overlay {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 3rem;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: var(--radius-full);
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state-copy {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-size: 0.85rem;
}

.provider-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.provider-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0.75rem;
  padding: 1rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.provider-summary {
  min-width: 0;
}

.provider-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.35rem;
  margin-bottom: 0.35rem;
}

.provider-title h3 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--text-heading);
}

.enabled-chip {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
  background: var(--accent-gold-dim);
}

.disabled-chip {
  color: var(--text-muted);
}

.provider-meta {
  margin: 0 0 0.5rem;
  color: var(--text-secondary);
  font-size: 0.85rem;
  overflow-wrap: anywhere;
}

.provider-details {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1rem;
  color: var(--text-muted);
  font-size: 0.8rem;
}

.provider-status,
.test-result,
.status-message {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.6rem 0.8rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
}

.provider-status {
  flex-direction: column;
  gap: 0.2rem;
  background: var(--bg-card);
}

.status-label {
  font-weight: 600;
  color: var(--text-primary);
}

.status-message-text {
  color: var(--text-secondary);
}

.provider-status.success,
.test-result.success {
  border-color: var(--color-positive);
  background: color-mix(in srgb, var(--color-positive) 14%, transparent);
}

.provider-status.error,
.test-result.error,
.status-message.error {
  border-color: var(--color-negative);
  background: color-mix(in srgb, var(--color-negative) 14%, transparent);
}

.provider-status.unknown {
  border-color: var(--border-subtle);
}

.test-result.success,
.test-result.success strong {
  color: var(--color-positive);
}

.test-result.error,
.test-result.error strong,
.status-message.error {
  color: var(--color-negative);
}

.test-result p {
  margin: 0.15rem 0 0;
  color: var(--text-secondary);
}

.provider-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  justify-content: flex-end;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: var(--overlay-dark);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  width: min(760px, 100%);
  max-height: 90vh;
  overflow: auto;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-header h3 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--text-heading);
}

.modal-close {
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: color var(--transition-fast);
}

.modal-close:hover {
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.form-hint {
  display: block;
  color: var(--text-muted);
  font-size: 0.75rem;
  margin-top: 0.25rem;
}

.toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.toggle {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 28px;
  flex-shrink: 0;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  transition: background var(--transition-fast);
}

.toggle-slider::before {
  content: '';
  position: absolute;
  width: 22px;
  height: 22px;
  left: 2px;
  bottom: 2px;
  background: var(--text-secondary);
  border-radius: var(--radius-full);
  transition: transform var(--transition-fast);
}

.toggle input:checked + .toggle-slider {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
}

.toggle input:checked + .toggle-slider::before {
  transform: translateX(22px);
  background: var(--accent-gold);
}

.form-error {
  margin-bottom: 1rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.5rem;
}

@media (max-width: 768px) {
  .section-heading,
  .section-actions,
  .provider-actions,
  .modal-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .form-grid {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
