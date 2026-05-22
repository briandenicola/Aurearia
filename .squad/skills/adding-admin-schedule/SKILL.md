# Skill: Adding Admin Schedule Panel

## Context

The Admin Schedules page (`AdminSchedulesSection.vue`) is where system-wide scheduled jobs are configured. Each scheduler (wishlist availability, collection valuation, auction-ending alerts) has its own panel with enable toggle, start time, interval, and save button.

## When to Use

When Cassius or another backend dev adds a new daily/periodic scheduler that needs user configuration (enabled/disabled, start time, interval), add a panel to the Schedules UI following this recipe.

## Recipe

### 1. Identify Setting Keys

The backend scheduler reads three keys from the `AppSettings` table:
- `{Feature}CheckEnabled` — boolean stored as string `'true'` or `'false'`, default `'false'`
- `{Feature}CheckStartTime` — string `"HH:MM"`, default `"08:00"`
- `{Feature}CheckInterval` — integer minutes stored as string, default `"1440"` (24 hours)

**Example:** For auction-ending alerts, the keys are `AuctionEndingCheckEnabled`, `AuctionEndingCheckStartTime`, `AuctionEndingCheckInterval`.

Check `.squad/decisions/inbox/` for backend docs if Cassius chose different names.

### 2. Add UI Panel in AdminSchedulesSection.vue

Open `src/web/src/components/admin/AdminSchedulesSection.vue`.

**Add the panel** after an existing section (e.g., after wishlist, before valuation):

```vue
<hr class="section-divider" />

<!-- {Feature Name} -->
<h3 class="subsection-title">{Panel Title}</h3>
<p class="subsection-desc">{One-line description of what the scheduler does.}</p>
<div class="avail-settings">
  <div class="form-group avail-toggle-row">
    <label class="form-label">Enable Automatic {Action}</label>
    <label class="toggle-switch">
      <input
        type="checkbox"
        :checked="settings.{Feature}CheckEnabled === 'true'"
        @change="settings.{Feature}CheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
      />
      <span class="toggle-slider"></span>
    </label>
  </div>
  <div class="form-group">
    <label class="form-label">Start Time (daily anchor)</label>
    <input
      v-model="settings.{Feature}CheckStartTime"
      class="form-input avail-interval-input"
      type="time"
    />
    <span class="form-hint">The first check runs at this time each day.</span>
  </div>
  <div class="form-group">
    <label class="form-label">Repeat Interval (minutes)</label>
    <input
      v-model="settings.{Feature}CheckInterval"
      class="form-input avail-interval-input"
      type="number"
      min="60"
      step="60"
    />
    <span class="form-hint">How often to repeat after the start time. Default {default interval description}.</span>
  </div>
  <div class="avail-save-row">
    <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
      {{ settingsSaving ? 'Saving...' : 'Save {Feature} Settings' }}
    </button>
    <span v-if="{feature}SettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': {feature}SettingsError }">{{ {feature}SettingsMsg }}</span>
  </div>
</div>
```

**Add props** to the `defineProps` block:

```ts
const props = defineProps<{
  settings: AppSettings
  settingsSaving: boolean
  availSettingsMsg: string
  availSettingsError: boolean
  {feature}SettingsMsg: string  // ADD THIS
  {feature}SettingsError: boolean  // ADD THIS
  valSettingsMsg: string
  valSettingsError: boolean
}>()
```

### 3. Update useAdminConfig.ts Composable

Open `src/web/src/composables/useAdminConfig.ts`.

**Add state refs:**

```ts
// Schedule-tab save messages (cleared alongside main settingsMsg)
const availSettingsMsg = ref('')
const availSettingsError = ref(false)
const {feature}SettingsMsg = ref('')  // ADD THIS
const {feature}SettingsError = ref(false)  // ADD THIS
const valSettingsMsg = ref('')
const valSettingsError = ref(false)
```

**Update loadSettings()** to apply defaults:

```ts
async function loadSettings() {
  try {
    const [settingsRes, defaultsRes] = await Promise.all([
      getAppSettings(),
      getAppSettingDefaults(),
    ])
    settingDefaults.value = { ...settingDefaults.value, ...defaultsRes.data }
    settings.value = { ...settings.value, ...settingsRes.data }

    // Apply defaults for {feature} settings if not set
    if (!settings.value.{Feature}CheckEnabled) {
      settings.value.{Feature}CheckEnabled = 'false'
    }
    if (!settings.value.{Feature}CheckStartTime) {
      settings.value.{Feature}CheckStartTime = '08:00'
    }
    if (!settings.value.{Feature}CheckInterval) {
      settings.value.{Feature}CheckInterval = '1440'
    }

    // ... rest of loadSettings
  } catch { /* use defaults */ }
}
```

**Update saveSettings()** to clear/set new messages:

```ts
async function saveSettings() {
  settingsSaving.value = true
  settingsMsg.value = ''
  settingsError.value = false
  availSettingsMsg.value = ''
  availSettingsError.value = false
  {feature}SettingsMsg.value = ''  // ADD THIS
  {feature}SettingsError.value = false  // ADD THIS
  valSettingsMsg.value = ''
  valSettingsError.value = false
  try {
    const entries = Object.entries(settings.value).map(([key, value]) => ({ key, value: String(value) }))
    await updateAppSettings(entries)
    settingsMsg.value = 'Settings saved'
    availSettingsMsg.value = 'Settings saved'
    {feature}SettingsMsg.value = 'Settings saved'  // ADD THIS
    valSettingsMsg.value = 'Settings saved'
    if (saveTimerId) clearTimeout(saveTimerId)
    saveTimerId = setTimeout(() => { availSettingsMsg.value = ''; {feature}SettingsMsg.value = ''; valSettingsMsg.value = '' }, 3000)
  } catch {
    settingsMsg.value = 'Failed to save settings'
    settingsError.value = true
    availSettingsMsg.value = 'Failed to save settings'
    availSettingsError.value = true
    {feature}SettingsMsg.value = 'Failed to save settings'  // ADD THIS
    {feature}SettingsError.value = true  // ADD THIS
    valSettingsMsg.value = 'Failed to save settings'
    valSettingsError.value = true
  } finally {
    settingsSaving.value = false
  }
}
```

**Update return object:**

```ts
return {
  // ... existing exports
  // Schedule messages
  availSettingsMsg,
  availSettingsError,
  {feature}SettingsMsg,  // ADD THIS
  {feature}SettingsError,  // ADD THIS
  valSettingsMsg,
  valSettingsError,
  // ... rest
}
```

### 4. Update AdminPage.vue Parent

Open `src/web/src/pages/AdminPage.vue`.

**Destructure new refs** from `useAdminConfig()`:

```ts
const {
  settings, settingDefaults, settingsMsg, settingsError, settingsSaving,
  ollamaTesting, ollamaTestResult, ollamaTestOk,
  anthropicTesting, anthropicTestResult, anthropicTestOk, anthropicModels,
  searxngTesting, searxngTestResult, searxngTestOk,
  coinSearchPromptDefault, coinShowsPromptDefault, valuationPromptDefault,
  availSettingsMsg, availSettingsError, {feature}SettingsMsg, {feature}SettingsError, valSettingsMsg, valSettingsError,  // ADD {feature} HERE
  loadSettings, saveSettings,
  testOllamaConnection, testAnthropicConn, testSearxngConn,
  cleanup: cleanupAdminConfig,
} = useAdminConfig()
```

**Update AdminSchedulesSection binding** in template:

```vue
<AdminSchedulesSection
  v-if="activeTab === 'schedules'"
  :settings="settings"
  :settings-saving="settingsSaving"
  :avail-settings-msg="availSettingsMsg"
  :avail-settings-error="availSettingsError"
  :{feature}-settings-msg="{feature}SettingsMsg"
  :{feature}-settings-error="{feature}SettingsError"
  :val-settings-msg="valSettingsMsg"
  :val-settings-error="valSettingsError"
  @save="saveSettings"
  @update:val-settings-msg="valSettingsMsg = $event"
  @update:val-settings-error="valSettingsError = $event"
/>
```

### 5. Run Type-Check

```bash
cd src/web
npx vue-tsc --noEmit
```

Must pass clean before committing.

## Design Guidelines

- **No emojis** in UI text
- **Use global classes** from `main.css` — `.btn`, `.btn-primary`, `.form-label`, `.form-input`, `.form-hint`, `.toggle-switch`, `.avail-settings`, `.subsection-title`
- **Subsection description** should be one sentence, plain English, no marketing language
- **Interval hint** should suggest realistic defaults (e.g., "Default 1440 (daily)" or "e.g. 120 = every 2 hours")
- **Start Time hint** should clarify it's a daily anchor (first run of the day)

## Example — Auction Ending Alerts

See `.squad/decisions/inbox/aurelia-journal-and-auction-schedule.md` for the complete implementation of `AuctionEndingCheckEnabled`, `AuctionEndingCheckStartTime`, `AuctionEndingCheckInterval`.

**Panel title:** "Auction Ending Alerts"  
**Description:** "Sends a Pushover alert each day for auction lots you are bidding on that end today."  
**Defaults:** `false`, `"08:00"`, `"1440"`

## Manual Run + Recent Runs Pattern

Valuation, Wishlist (Availability), and Auction Ending now all support manual "Run Now" buttons and run history logs. This pattern is **handled entirely in AdminSchedulesSection.vue**, not in the composable.

### Adding Manual Run + Run Log to a Scheduler

If the backend implements:
- Manual trigger endpoint (e.g., `POST /api/admin/{feature}/run`)
- Runs history endpoint (e.g., `GET /api/admin/{feature}/runs?page=N&limit=N`)

Then add the UI as follows:

#### 1. Add TypeScript types in `src/web/src/types/index.ts`

```typescript
export interface {Feature}Result {
  id: number
  runId: number
  // ...other fields specific to this scheduler
  checkedAt: string
}

export interface {Feature}Run {
  id: number
  userId: number
  triggerType: string          // "manual" | "scheduled"
  triggerUserId: number | null
  // ...metrics (e.g., lotsChecked, alertsSent, errors)
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage?: string
  results?: {Feature}Result[]
  createdAt: string
}
```

#### 2. Add API client functions in `src/web/src/api/client.ts`

```typescript
// {Feature} Runs
export const get{Feature}Runs = (page = 1, limit = 20) =>
  api.get<{ runs: {Feature}Run[]; total: number }>('/admin/{feature}/runs', { params: { page, limit } })
export const get{Feature}RunDetail = (runId: number) =>
  api.get<{Feature}Run>(`/admin/{feature}/runs/${runId}`)
export const trigger{Feature}Check = () =>
  api.post<{ message: string; users: number }>('/admin/{feature}/run')
```

Don't forget to add the type to the import list at the top of `client.ts`.

#### 3. Update `AdminSchedulesSection.vue`

**Add imports:**
```typescript
import { get{Feature}Runs, get{Feature}RunDetail, trigger{Feature}Check } from '@/api/client'
import type { {Feature}Run } from '@/types'
```

**Add emits:**
```typescript
const emit = defineEmits<{
  save: []
  'update:{feature}SettingsMsg': [val: string]
  'update:{feature}SettingsError': [val: boolean]
}>()
```

**Add state after existing scheduler sections:**
```typescript
// {Feature}
const {feature}Runs = ref<{Feature}Run[]>([])
const {feature}Total = ref(0)
const {feature}Page = ref(1)
const {feature}Loading = ref(false)
const {feature}TriggerLoading = ref(false)
const {feature}ExpandedRunId = ref<number | null>(null)
const {feature}ExpandedResults = ref<{Feature}Run['results']>(undefined)
const {feature}ExpandedLoading = ref(false)
const {feature}Colspan = computed(() => isMobile.value ? 4 : 6) // adjust colspan to match table columns
```

**Add functions:**
```typescript
async function load{Feature}Runs() {
  {feature}Loading.value = true
  try {
    const res = await get{Feature}Runs({feature}Page.value, 5)
    {feature}Runs.value = res.data.runs ?? []
    {feature}Total.value = res.data.total ?? 0
  } catch { /* ignore */ } finally {
    {feature}Loading.value = false
  }
}

async function toggle{Feature}RunDetail(runId: number) {
  if ({feature}ExpandedRunId.value === runId) {
    {feature}ExpandedRunId.value = null
    {feature}ExpandedResults.value = undefined
    return
  }
  {feature}ExpandedRunId.value = runId
  {feature}ExpandedResults.value = []
  {feature}ExpandedLoading.value = true
  try {
    const res = await get{Feature}RunDetail(runId)
    {feature}ExpandedResults.value = res.data.results ?? []
  } catch {
    {feature}ExpandedResults.value = []
  } finally {
    {feature}ExpandedLoading.value = false
  }
}

async function triggerManual{Feature}Check() {
  {feature}TriggerLoading.value = true
  emit('update:{feature}SettingsMsg', '')
  emit('update:{feature}SettingsError', false)
  try {
    await trigger{Feature}Check()
    emit('update:{feature}SettingsMsg', '{Feature} check started')
    timers.push(setTimeout(() => { emit('update:{feature}SettingsMsg', '') }, 10000))
    timers.push(setTimeout(() => { load{Feature}Runs() }, 2000))
  } catch {
    emit('update:{feature}SettingsMsg', 'Failed to trigger {feature} check')
    emit('update:{feature}SettingsError', true)
  } finally {
    {feature}TriggerLoading.value = false
  }
}
```

**Update onMounted:**
```typescript
onMounted(() => {
  window.addEventListener('resize', onResize)
  loadAvailRuns()
  load{Feature}Runs()  // ADD THIS
  loadValRuns()
})
```

**Add UI in template (after the settings panel, before the next divider):**
```vue
<hr class="section-divider" />
<h3 class="subsection-title">{Feature} Run History</h3>

<div v-if="{feature}Loading" class="loading-overlay"><div class="spinner"></div></div>
<div v-else-if="{feature}Runs.length === 0" class="logs-empty">No {feature} runs recorded yet.</div>
<template v-else>
  <table class="users-table avail-table">
    <thead>
      <tr>
        <th>Date</th>
        <th class="hide-mobile">Trigger</th>
        <!-- Add columns for your metrics -->
        <th>Duration</th>
      </tr>
    </thead>
    <tbody>
      <template v-for="run in {feature}Runs" :key="run.id">
        <tr class="avail-row" :class="{ 'avail-row-expanded': {feature}ExpandedRunId === run.id }" @click="toggle{Feature}RunDetail(run.id)">
          <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
          <td class="hide-mobile">{{ run.triggerType }}</td>
          <!-- Add metric cells -->
          <td>{{ formatDuration(run.durationMs) }}</td>
        </tr>
        <tr v-if="{feature}ExpandedRunId === run.id && {feature}ExpandedResults" class="avail-detail-row">
          <td :colspan="{feature}Colspan">
            <div v-if="{feature}ExpandedLoading" class="loading-overlay"><div class="spinner"></div></div>
            <table v-else-if="{feature}ExpandedResults.length" class="avail-detail-table {feature}-detail-table">
              <thead>
                <tr>
                  <!-- Add detail columns -->
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in {feature}ExpandedResults" :key="r.id">
                  <!-- Add detail cells -->
                </tr>
              </tbody>
            </table>
            <p v-else class="logs-empty">No results for this run.</p>
          </td>
        </tr>
      </template>
    </tbody>
  </table>

  <div class="avail-pagination">
    <button class="btn btn-secondary btn-sm" :disabled="{feature}Page <= 1" @click="{feature}Page--; load{Feature}Runs()">Prev</button>
    <span class="avail-page-info">Page {{ {feature}Page }}</span>
    <button class="btn btn-secondary btn-sm" :disabled="{feature}Runs.length < 5" @click="{feature}Page++; load{Feature}Runs()">Next</button>
  </div>
</template>
```

**Add "Run Now" button to settings panel:**
```vue
<div class="avail-save-row">
  <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
    {{ settingsSaving ? 'Saving...' : 'Save {Feature} Settings' }}
  </button>
  <button class="btn btn-secondary btn-sm" :disabled="{feature}TriggerLoading" @click="triggerManual{Feature}Check()">
    {{ {feature}TriggerLoading ? 'Starting...' : 'Run Now' }}
  </button>
  <span v-if="{feature}SettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': {feature}SettingsError }">{{ {feature}SettingsMsg }}</span>
</div>
```

**Add CSS for detail table columns (if unique column structure):**
```css
.{feature}-detail-table th:nth-child(1),
.{feature}-detail-table td:nth-child(1) { width: 20%; }
/* ... adjust widths to match your columns */
```

#### 4. Update `AdminPage.vue` parent

**Add emit handlers to `AdminSchedulesSection` binding:**
```vue
<AdminSchedulesSection
  v-if="activeTab === 'schedules'"
  ...
  :{feature}-settings-msg="{feature}SettingsMsg"
  :{feature}-settings-error="{feature}SettingsError"
  @update:{feature}-settings-msg="{feature}SettingsMsg = $event"
  @update:{feature}-settings-error="{feature}SettingsError = $event"
/>
```

No changes needed to `useAdminConfig.ts` — the composable only manages settings save state, not run history.

#### 5. Verify

```bash
cd src/web
npm run type-check
npm run build
```

Both must pass clean.

## Files Touched

1. `src/web/src/components/admin/AdminSchedulesSection.vue` — add panel and props
2. `src/web/src/composables/useAdminConfig.ts` — add state, defaults, clear/set in save, expose in return
3. `src/web/src/pages/AdminPage.vue` — destructure and pass props

## Notes

- The `AppSettings` interface in `src/web/src/types/index.ts` has an index signature `[key: string]: string`, so new keys don't require type changes — they're dynamically accessible.
- If the backend hasn't implemented the scheduler yet, the settings will still save/load — they just won't have any effect until the backend reads them.
- Always check for backend decision docs in `.squad/decisions/inbox/cassius-*.md` to confirm the exact setting key names before implementing.
