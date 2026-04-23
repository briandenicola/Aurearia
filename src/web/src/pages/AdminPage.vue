<template>
  <div class="container">
    <div class="page-header">
      <h1>Admin</h1>
    </div>

    <div v-if="!auth.isAdmin" class="empty-state">
      <h3>Access Denied</h3>
      <p>Admin privileges required</p>
    </div>

    <div v-else class="admin-layout">
      <!-- Tab Nav -->
      <div class="tab-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
        ><component :is="tabIcons[tab.id]" :size="16" /> {{ tab.label }}</button>
      </div>

      <!-- Users Tab -->
      <section v-if="activeTab === 'users'" class="admin-section card">
        <h2>User Management</h2>
        <div v-if="usersLoading" class="loading-overlay"><div class="spinner"></div></div>
        <table v-else class="users-table">
          <thead>
            <tr>
              <th>Username</th>
              <th>Role</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id">
              <td>
                <span class="username">{{ user.username }}</span>
                <span v-if="user.id === auth.user?.id" class="you-badge">(you)</span>
              </td>
              <td>
                <span class="badge" :class="`badge-${user.role === 'admin' ? 'roman' : 'modern'}`">
                  {{ user.role }}
                </span>
              </td>
              <td class="date-cell">{{ formatDate(user.createdAt) }}</td>
              <td>
                <div v-if="user.id !== auth.user?.id" class="action-btns">
                  <button class="btn btn-secondary btn-sm" @click="openResetModal(user)">
                    Reset
                  </button>
                  <button class="btn btn-danger btn-sm" @click="handleDeleteUser(user)">
                    Delete
                  </button>
                </div>
                <span v-else class="text-muted">—</span>
              </td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- AI Tab -->
      <section v-if="activeTab === 'ai'" class="admin-section card">
        <h2>AI Configuration</h2>
        <form @submit.prevent="saveSettings">
          <!-- Provider Selection -->
          <div class="form-group">
            <label class="form-label">AI Provider</label>
            <div class="provider-toggle">
              <label class="provider-option" :class="{ active: settings.AIProvider === 'anthropic' }">
                <input type="radio" v-model="settings.AIProvider" value="anthropic" />
                <span class="provider-label">Anthropic (Recommended)</span>
                <span class="provider-desc">Claude models with built-in web search</span>
              </label>
              <label class="provider-option" :class="{ active: settings.AIProvider === 'ollama' }">
                <input type="radio" v-model="settings.AIProvider" value="ollama" />
                <span class="provider-label">Ollama</span>
                <span class="provider-desc">Self-hosted models. Requires SearXNG for web search.</span>
              </label>
            </div>
            <p v-if="!settings.AIProvider" class="provider-warning">
              Please select an AI provider to enable agent features.
            </p>
          </div>

          <!-- Anthropic Settings -->
          <template v-if="settings.AIProvider === 'anthropic'">
            <div class="form-group">
              <label class="form-label">Anthropic API Key</label>
              <input v-model="settings.AnthropicAPIKey" class="form-input" type="password" placeholder="Enter your Anthropic API key" />
              <span class="form-hint">Get a key at <a href="https://console.anthropic.com/" target="_blank" rel="noopener">console.anthropic.com</a></span>
            </div>
            <div class="form-group">
              <label class="form-label">Anthropic Model</label>
              <select v-model="settings.AnthropicModel" class="form-input">
                <option v-for="m in anthropicModels" :key="m.id" :value="m.id">{{ m.name }}</option>
              </select>
            </div>
            <div class="connectivity-actions">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="anthropicTesting" @click="testAnthropicConn">
                {{ anthropicTesting ? 'Testing...' : 'Test Anthropic API' }}
              </button>
              <div v-if="anthropicTestResult" class="connectivity-result" :class="{ success: anthropicTestOk, error: !anthropicTestOk }">
                <span class="connectivity-icon">{{ anthropicTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
                {{ anthropicTestResult }}
              </div>
            </div>
          </template>

          <!-- Ollama Settings -->
          <template v-if="settings.AIProvider === 'ollama'">
            <div class="form-group">
              <label class="form-label">Ollama URL</label>
              <input v-model="settings.OllamaURL" class="form-input" placeholder="http://localhost:11434" />
            </div>
            <div class="form-group">
              <label class="form-label">Vision Model</label>
              <input v-model="settings.OllamaModel" class="form-input" placeholder="llava" />
              <span class="form-hint">e.g. llava, llama3.2-vision, bakllava</span>
            </div>
            <div class="form-group">
              <label class="form-label">Request Timeout (seconds)</label>
              <input v-model="settings.OllamaTimeout" class="form-input" type="number" min="10" max="1800" step="10" />
              <span class="form-hint">Time limit for AI analysis calls. Default: 300 (5 minutes)</span>
            </div>
            <div class="form-group">
              <label class="form-label">SearXNG URL</label>
              <input v-model="settings.SearXNGURL" class="form-input" placeholder="http://localhost:8888" />
              <span class="form-hint">Required for web search features (coin search, coin shows, valuations).</span>
              <p v-if="settings.AIProvider === 'ollama' && !settings.SearXNGURL" class="provider-warning">
                Web search features require a SearXNG instance. Configure the URL or switch to Anthropic.
              </p>
            </div>
            <div class="connectivity-actions">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="ollamaTesting" @click="testOllamaConnection">
                {{ ollamaTesting ? 'Testing...' : 'Test Ollama' }}
              </button>
              <button v-if="settings.SearXNGURL" type="button" class="btn btn-secondary btn-sm" :disabled="searxngTesting" @click="testSearxngConn">
                {{ searxngTesting ? 'Testing...' : 'Test SearXNG' }}
              </button>
            </div>
            <div v-if="ollamaTestResult" class="connectivity-result" :class="{ success: ollamaTestOk, error: !ollamaTestOk }">
              <span class="connectivity-icon">{{ ollamaTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
              {{ ollamaTestResult }}
            </div>
            <div v-if="searxngTestResult" class="connectivity-result" :class="{ success: searxngTestOk, error: !searxngTestOk }">
              <span class="connectivity-icon">{{ searxngTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
              {{ searxngTestResult }}
            </div>
          </template>

          <!-- Shared Prompt Settings (visible when a provider is selected) -->
          <template v-if="settings.AIProvider">
            <hr class="section-divider" />
            <h3 class="subsection-title">Agent Prompts</h3>
            <div class="form-group">
              <div class="prompt-header">
                <label class="form-label">Coin Search Prompt</label>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="settings.CoinSearchPrompt === coinSearchPromptDefault"
                  @click="settings.CoinSearchPrompt = coinSearchPromptDefault"
                >Revert to Default</button>
              </div>
              <textarea
                v-model="settings.CoinSearchPrompt"
                class="form-textarea"
                rows="8"
              />
              <span class="form-hint">Search instructions for the coin search agent (Team 1). Controls which dealer sites to search, availability rules, and search strategy.</span>
            </div>
            <div class="form-group">
              <div class="prompt-header">
                <label class="form-label">Coin Shows Prompt</label>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="settings.CoinShowsPrompt === coinShowsPromptDefault"
                  @click="settings.CoinShowsPrompt = coinShowsPromptDefault"
                >Revert to Default</button>
              </div>
              <textarea
                v-model="settings.CoinShowsPrompt"
                class="form-textarea"
                rows="8"
              />
              <span class="form-hint">Search instructions for the coin shows agent (Team 2). Controls which show directories and organizations to search.</span>
            </div>
            <div class="form-group">
              <div class="prompt-header">
                <label class="form-label">Value Estimator Prompt</label>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="settings.ValuationPrompt === valuationPromptDefault"
                  @click="settings.ValuationPrompt = valuationPromptDefault"
                >Revert to Default</button>
              </div>
              <textarea
                v-model="settings.ValuationPrompt"
                class="form-textarea"
                rows="8"
              />
              <span class="form-hint">System prompt for the AI value estimator. Controls how it researches and estimates coin values.</span>
            </div>
            <h3 class="subsection-title">Analysis Prompts</h3>
            <div class="form-group">
              <div class="prompt-header">
                <label class="form-label">Obverse Analysis Prompt</label>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="settings.ObversePrompt === settingDefaults.ObversePrompt"
                  @click="settings.ObversePrompt = settingDefaults.ObversePrompt"
                >Revert to Default</button>
              </div>
              <textarea
                v-model="settings.ObversePrompt"
                class="form-textarea"
                rows="6"
              />
              <span class="form-hint">Prompt for obverse image analysis. Coin context is appended automatically.</span>
            </div>
            <div class="form-group">
              <div class="prompt-header">
                <label class="form-label">Reverse Analysis Prompt</label>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="settings.ReversePrompt === settingDefaults.ReversePrompt"
                  @click="settings.ReversePrompt = settingDefaults.ReversePrompt"
                >Revert to Default</button>
              </div>
              <textarea
                v-model="settings.ReversePrompt"
                class="form-textarea"
                rows="6"
              />
              <span class="form-hint">Prompt for reverse image analysis. Coin context is appended automatically.</span>
            </div>
            <div class="form-group">
              <div class="prompt-header">
                <label class="form-label">Text Extraction Prompt</label>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="settings.TextExtractionPrompt === settingDefaults.TextExtractionPrompt"
                  @click="settings.TextExtractionPrompt = settingDefaults.TextExtractionPrompt"
                >Revert to Default</button>
              </div>
              <textarea
                v-model="settings.TextExtractionPrompt"
                class="form-textarea"
                rows="6"
              />
              <span class="form-hint">Prompt for extracting text from store card images.</span>
            </div>
          </template>

          <p v-if="settingsMsg" class="msg" :class="{ error: settingsError }">{{ settingsMsg }}</p>
          <div class="ai-actions">
            <button type="submit" class="btn btn-primary btn-sm" :disabled="settingsSaving">
              {{ settingsSaving ? 'Saving...' : 'Save AI Settings' }}
            </button>
          </div>
        </form>
      </section>

      <!-- System Tab -->
      <section v-if="activeTab === 'system'" class="admin-section card">
        <h2>System Settings</h2>
        <form @submit.prevent="saveSettings">
          <div class="form-group">
            <label class="form-label">Numista API Key</label>
            <input v-model="settings.NumistaAPIKey" class="form-input" type="password" placeholder="Enter your Numista API key" />
            <span class="form-hint">Get a free key at <a href="https://en.numista.com/api/" target="_blank" rel="noopener">numista.com/api</a> (2,000 requests/month free)</span>
          </div>
          <div class="form-group">
            <label class="form-label">Log Level</label>
            <select v-model="settings.LogLevel" class="form-select">
              <option v-for="level in LOG_LEVELS" :key="level" :value="level">{{ level }}</option>
            </select>
          </div>
          <p v-if="settingsMsg" class="msg" :class="{ error: settingsError }">{{ settingsMsg }}</p>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="settingsSaving">
            {{ settingsSaving ? 'Saving...' : 'Save System Settings' }}
          </button>
        </form>
        <div class="version-info">
          <span class="version-label">Version</span>
          <span class="version-value">{{ appVersion }}</span>
          <span v-if="buildDate" class="version-date">Built {{ buildDate }}</span>
        </div>
      </section>

      <!-- Logs Tab -->
      <section v-if="activeTab === 'logs'" class="admin-section card">
        <h2>Application Logs</h2>
        <div class="logs-toolbar">
          <select v-model="logsFilter" class="form-select logs-filter" @change="loadLogs">
            <option value="">All Levels</option>
            <option v-for="level in ['TRACE','DEBUG','INFO','WARN','ERROR']" :key="level" :value="level">{{ level }}</option>
          </select>
          <button class="btn btn-secondary btn-sm" @click="loadLogs" :disabled="logsLoading">
            {{ logsLoading ? 'Loading...' : 'Refresh' }}
          </button>
          <button
            class="btn btn-sm"
            :class="logsAutoRefresh ? 'btn-primary' : 'btn-secondary'"
            @click="toggleAutoRefresh"
          >
            {{ logsAutoRefresh ? 'Auto ●' : 'Auto ○' }}
          </button>
          <button
            class="btn btn-secondary btn-sm"
            @click="exportLogs"
            :disabled="logs.length === 0"
            title="Export logs as text file"
          >
            <Download :size="14" /> Export
          </button>
        </div>
        <div class="logs-container">
          <div v-if="logs.length === 0 && !logsLoading" class="logs-empty">
            No log entries. Click Refresh to load.
          </div>
          <div
            v-for="(entry, i) in logs"
            :key="i"
            class="log-entry"
            :class="logLevelClass(entry.level)"
          >
            <span class="log-time">{{ entry.timestamp.substring(11, 19) }}</span>
            <span class="log-level-badge">{{ entry.level }}</span>
            <span class="log-msg">{{ entry.message }}</span>
          </div>
        </div>
      </section>

      <!-- Schedules Tab -->
      <section v-if="activeTab === 'schedules'" class="admin-section card">
        <h2>Schedules</h2>

        <!-- Wishlist Availability Check -->
        <h3 class="subsection-title">Wishlist Availability Check</h3>
        <div class="avail-settings">
          <div class="form-group avail-toggle-row">
            <label class="form-label">Enable Automatic Checks</label>
            <label class="toggle-switch">
              <input
                type="checkbox"
                :checked="settings.WishlistCheckEnabled === 'true'"
                @change="settings.WishlistCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
              />
              <span class="toggle-slider"></span>
            </label>
          </div>
          <div class="form-group">
            <label class="form-label">Start Time (daily anchor)</label>
            <input
              v-model="settings.WishlistCheckStartTime"
              class="form-input avail-interval-input"
              type="time"
            />
            <span class="form-hint">The first check runs at this time each day. Subsequent checks repeat at the interval below.</span>
          </div>
          <div class="form-group">
            <label class="form-label">Repeat Interval (minutes)</label>
            <input
              v-model="settings.WishlistCheckInterval"
              class="form-input avail-interval-input"
              type="number"
              min="5"
              step="5"
            />
            <span class="form-hint">How often to repeat after the start time (e.g. 120 = every 2 hours).</span>
          </div>
          <div class="avail-save-row">
            <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="saveSettings()">
              {{ settingsSaving ? 'Saving...' : 'Save Schedule Settings' }}
            </button>
            <span v-if="availSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': availSettingsError }">{{ availSettingsMsg }}</span>
          </div>
        </div>

        <hr class="section-divider" />
        <h3 class="subsection-title">Availability Run History</h3>

        <div v-if="availLoading" class="loading-overlay"><div class="spinner"></div></div>
        <div v-else-if="availRuns.length === 0" class="logs-empty">No availability runs recorded yet.</div>
        <template v-else>
          <table class="users-table avail-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Trigger</th>
                <th>Checked</th>
                <th>Avail</th>
                <th>Unavail</th>
                <th>Unknown</th>
                <th>Errors</th>
                <th>Duration</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="run in availRuns" :key="run.id">
                <tr class="avail-row" :class="{ 'avail-row-expanded': expandedRunId === run.id }" @click="toggleRunDetail(run.id)">
                  <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
                  <td>{{ run.triggerType }}</td>
                  <td>{{ run.coinsChecked }}</td>
                  <td class="avail-count-available">{{ run.available }}</td>
                  <td class="avail-count-unavailable">{{ run.unavailable }}</td>
                  <td class="avail-count-unknown">{{ run.unknown }}</td>
                  <td>{{ run.errors }}</td>
                  <td>{{ formatDuration(run.durationMs) }}</td>
                </tr>
                <tr v-if="expandedRunId === run.id && expandedResults" class="avail-detail-row">
                  <td colspan="8">
                    <div v-if="expandedLoading" class="loading-overlay"><div class="spinner"></div></div>
                    <table v-else-if="expandedResults.length" class="avail-detail-table">
                      <thead>
                        <tr>
                          <th>Coin</th>
                          <th>URL</th>
                          <th>Status</th>
                          <th>Reason</th>
                          <th>HTTP</th>
                          <th>Agent</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-for="r in expandedResults" :key="r.id">
                          <td>{{ r.coinName }}</td>
                          <td><a v-if="r.url" :href="r.url" target="_blank" rel="noopener" class="avail-link" @click.stop>{{ truncateUrl(r.url) }}</a><span v-else class="text-muted">--</span></td>
                          <td>
                            <span class="listing-status-badge" :class="'listing-' + r.status">{{ r.status }}</span>
                          </td>
                          <td class="avail-reason">{{ r.reason || '--' }}</td>
                          <td>{{ r.httpStatus ?? '--' }}</td>
                          <td>{{ r.agentUsed ? 'Yes' : 'No' }}</td>
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
            <button class="btn btn-secondary btn-sm" :disabled="availPage <= 1" @click="availPage--; loadAvailRuns()">Prev</button>
            <span class="avail-page-info">Page {{ availPage }}</span>
            <button class="btn btn-secondary btn-sm" :disabled="availRuns.length < 5" @click="availPage++; loadAvailRuns()">Next</button>
          </div>
        </template>

        <hr class="section-divider" />

        <!-- Collection Valuation -->
        <h3 class="subsection-title">Collection Valuation</h3>
        <div class="avail-settings">
          <div class="form-group avail-toggle-row">
            <label class="form-label">Enable Scheduled Valuation</label>
            <label class="toggle-switch">
              <input
                type="checkbox"
                :checked="settings.ValuationCheckEnabled === 'true'"
                @change="settings.ValuationCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
              />
              <span class="toggle-slider"></span>
            </label>
          </div>
          <div class="form-group">
            <label class="form-label">Start Time (daily anchor)</label>
            <input
              v-model="settings.ValuationCheckStartTime"
              class="form-input avail-interval-input"
              type="time"
            />
            <span class="form-hint">The valuation cycle starts at this time on scheduled days.</span>
          </div>
          <div class="form-group">
            <label class="form-label">Repeat Interval (days)</label>
            <input
              v-model="settings.ValuationCheckIntervalDays"
              class="form-input avail-interval-input"
              type="number"
              min="1"
              step="1"
            />
            <span class="form-hint">How often to run (e.g. 7 = weekly). AI valuations are costly so daily runs are not recommended.</span>
          </div>
          <div class="form-group">
            <label class="form-label">Max Coins Per Run</label>
            <input
              v-model="settings.ValuationMaxCoins"
              class="form-input avail-interval-input"
              type="number"
              min="1"
              step="10"
            />
            <span class="form-hint">Limit how many coins are valuated per run to control AI costs.</span>
          </div>
          <div class="avail-save-row">
            <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="saveSettings()">
              {{ settingsSaving ? 'Saving...' : 'Save Valuation Settings' }}
            </button>
            <button class="btn btn-secondary btn-sm" :disabled="valTriggerLoading" @click="triggerManualValuation()">
              {{ valTriggerLoading ? 'Starting...' : 'Run Now' }}
            </button>
            <span v-if="valSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': valSettingsError }">{{ valSettingsMsg }}</span>
          </div>
        </div>

        <hr class="section-divider" />
        <h3 class="subsection-title">Valuation Run History</h3>

        <div v-if="valLoading" class="loading-overlay"><div class="spinner"></div></div>
        <div v-else-if="valRuns.length === 0" class="logs-empty">No valuation runs recorded yet.</div>
        <template v-else>
          <table class="users-table avail-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Trigger</th>
                <th>Status</th>
                <th>Checked</th>
                <th>Updated</th>
                <th>Skipped</th>
                <th>Errors</th>
                <th>Duration</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="run in valRuns" :key="run.id">
                <tr class="avail-row" :class="{ 'avail-row-expanded': valExpandedRunId === run.id }" @click="toggleValRunDetail(run.id)">
                  <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
                  <td>{{ run.triggerType }}</td>
                  <td>
                    <span class="val-status-badge" :class="'val-status-' + run.status">{{ run.status }}</span>
                    <span v-if="run.status === 'running' && run.totalCoins > 0" class="val-progress">
                      {{ run.coinsChecked + run.coinsSkipped + run.errors }} / {{ run.totalCoins }}
                    </span>
                    <button v-if="run.status === 'running'" class="btn-cancel-run" @click.stop="cancelRun(run.id)">Cancel</button>
                  </td>
                  <td>{{ run.coinsChecked }}</td>
                  <td class="avail-count-available">{{ run.coinsUpdated }}</td>
                  <td class="avail-count-unknown">{{ run.coinsSkipped }}</td>
                  <td class="avail-count-unavailable">{{ run.errors }}</td>
                  <td>{{ formatDuration(run.durationMs) }}</td>
                </tr>
                <tr v-if="valExpandedRunId === run.id && valExpandedResults" class="avail-detail-row">
                  <td colspan="8">
                    <div v-if="valExpandedLoading" class="loading-overlay"><div class="spinner"></div></div>
                    <table v-else-if="valExpandedResults.length" class="avail-detail-table val-detail-table">
                      <thead>
                        <tr>
                          <th>Coin</th>
                          <th>Previous</th>
                          <th>Estimated</th>
                          <th>Confidence</th>
                          <th>Status</th>
                          <th>Reasoning</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-for="r in valExpandedResults" :key="r.id">
                          <td>{{ r.coinName }}</td>
                          <td>{{ r.previousValue != null ? `$${r.previousValue.toFixed(2)}` : '--' }}</td>
                          <td class="val-value">{{ r.estimatedValue > 0 ? `$${r.estimatedValue.toFixed(2)}` : '--' }}</td>
                          <td>
                            <span v-if="r.confidence" class="val-confidence" :class="'val-conf-' + r.confidence">{{ r.confidence }}</span>
                            <span v-else>--</span>
                          </td>
                          <td>
                            <span class="listing-status-badge" :class="'val-result-' + r.status">{{ r.status }}</span>
                          </td>
                          <td class="avail-reason">{{ r.reasoning || r.errorMessage || '--' }}</td>
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
            <button class="btn btn-secondary btn-sm" :disabled="valPage <= 1" @click="valPage--; loadValRuns()">Prev</button>
            <span class="avail-page-info">Page {{ valPage }}</span>
            <button class="btn btn-secondary btn-sm" :disabled="valRuns.length < 5" @click="valPage++; loadValRuns()">Next</button>
          </div>
        </template>
      </section>

      <!-- Reset Password Modal -->
      <div v-if="resetTarget" class="modal-overlay" @click.self="resetTarget = null">
        <div class="modal card">
          <h3>Reset Password for {{ resetTarget.username }}</h3>
          <form @submit.prevent="handleResetPassword">
            <div class="form-group">
              <label class="form-label">New Password</label>
              <input v-model="resetNewPassword" type="password" class="form-input" required minlength="6" />
            </div>
            <p v-if="resetMsg" class="msg" :class="{ error: resetError }">{{ resetMsg }}</p>
            <div class="modal-actions">
              <button type="button" class="btn btn-secondary btn-sm" @click="resetTarget = null">Cancel</button>
              <button type="submit" class="btn btn-primary btn-sm" :disabled="resetLoading">
                {{ resetLoading ? 'Resetting...' : 'Reset Password' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, type Component } from 'vue'
import { useAuthStore } from '@/stores/auth'
import {
  getUsers, deleteUser, resetUserPassword,
  getAppSettings, getAppSettingDefaults, updateAppSettings, getAdminLogs, getOllamaStatus,
  getAnthropicModels, getCoinSearchPrompt, getCoinShowsPrompt, getValuationPrompt,
  testAnthropicConnection, testSearXNGConnection,
  getAvailabilityRuns, getAvailabilityRunDetail,
  getValuationRuns, getValuationRunDetail, triggerValuation, cancelValuationRun,
} from '@/api/client'
import type { AnthropicModel } from '@/api/client'
import { LOG_LEVELS } from '@/types'
import type { UserInfo, AppSettings, LogEntry, AvailabilityRun, ValuationRun } from '@/types'
import { useDialog } from '@/composables/useDialog'
import { Users, Cpu, Wrench, ScrollText, Download, ShieldCheck, CalendarClock } from 'lucide-vue-next'

const tabIcons: Record<string, Component> = { users: Users, ai: Cpu, system: Wrench, logs: ScrollText, schedules: CalendarClock }

const { showConfirm, showAlert } = useDialog()
const auth = useAuthStore()

const rawVersion = import.meta.env.VITE_APP_VERSION || 'dev'
const appVersion = computed(() => {
  if (rawVersion === 'dev') return 'dev'
  return rawVersion.length > 8 ? rawVersion.substring(0, 7) : rawVersion
})
const buildDate = computed(() => {
  const raw = import.meta.env.VITE_BUILD_DATE
  if (!raw) return ''
  try {
    return new Date(raw).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
  } catch {
    return raw
  }
})

const tabs = [
  { id: 'users', icon: 'users', label: 'Users' },
  { id: 'ai', icon: 'cpu', label: 'AI' },
  { id: 'system', icon: 'wrench', label: 'System' },
  { id: 'logs', icon: 'scroll-text', label: 'Logs' },
  { id: 'schedules', icon: 'calendar-clock', label: 'Schedules' },
]
const activeTab = ref('users')

// Users
const users = ref<UserInfo[]>([])
const usersLoading = ref(true)

async function loadUsers() {
  usersLoading.value = true
  try {
    const res = await getUsers()
    users.value = res.data
  } finally {
    usersLoading.value = false
  }
}

async function handleDeleteUser(user: UserInfo) {
  if (!await showConfirm(`Delete user "${user.username}" and all their data? This cannot be undone.`, { title: 'Delete User', variant: 'danger' })) return
  try {
    await deleteUser(user.id)
    users.value = users.value.filter((u) => u.id !== user.id)
  } catch {
    await showAlert('Failed to delete user', { title: 'Error' })
  }
}

// Reset password modal
const resetTarget = ref<UserInfo | null>(null)
const resetNewPassword = ref('')
const resetMsg = ref('')
const resetError = ref(false)
const resetLoading = ref(false)

function openResetModal(user: UserInfo) {
  resetTarget.value = user
  resetNewPassword.value = ''
  resetMsg.value = ''
  resetError.value = false
}

async function handleResetPassword() {
  if (!resetTarget.value) return
  resetLoading.value = true
  resetMsg.value = ''
  try {
    await resetUserPassword(resetTarget.value.id, resetNewPassword.value)
    resetMsg.value = 'Password reset successfully'
    setTimeout(() => { resetTarget.value = null }, 1200)
  } catch {
    resetMsg.value = 'Failed to reset password'
    resetError.value = true
  } finally {
    resetLoading.value = false
  }
}

// Settings
const settings = ref<AppSettings>({
  AIProvider: '',
  OllamaURL: 'http://localhost:11434',
  OllamaModel: 'llava',
  ObversePrompt: '',
  ReversePrompt: '',
  TextExtractionPrompt: '',
  OllamaTimeout: '300',
  SearXNGURL: '',
  LogLevel: 'info',
})
const settingDefaults = ref<AppSettings>({
  AIProvider: '',
  OllamaURL: '',
  OllamaModel: '',
  ObversePrompt: '',
  ReversePrompt: '',
  TextExtractionPrompt: '',
  OllamaTimeout: '',
  SearXNGURL: '',
  LogLevel: '',
})
const settingsMsg = ref('')
const settingsError = ref(false)
const settingsSaving = ref(false)
const ollamaTesting = ref(false)
const ollamaTestResult = ref('')
const ollamaTestOk = ref(false)
const anthropicTesting = ref(false)
const anthropicTestResult = ref('')
const anthropicTestOk = ref(false)
const searxngTesting = ref(false)
const searxngTestResult = ref('')
const searxngTestOk = ref(false)
const anthropicModels = ref<AnthropicModel[]>([
  { id: 'claude-sonnet-4-20250514', name: 'Claude Sonnet 4' },
  { id: 'claude-haiku-4-20250414', name: 'Claude Haiku 4' },
  { id: 'claude-opus-4-20250514', name: 'Claude Opus 4' },
])
const coinSearchPromptDefault = ref('')
const coinShowsPromptDefault = ref('')
const valuationPromptDefault = ref('')

async function loadSettings() {
  try {
    const [settingsRes, defaultsRes] = await Promise.all([
      getAppSettings(),
      getAppSettingDefaults(),
    ])
    settingDefaults.value = { ...settingDefaults.value, ...defaultsRes.data }
    settings.value = { ...settings.value, ...settingsRes.data }

    // Load Anthropic models and prompts in parallel
    const [modelsRes, coinSearchRes, coinShowsRes, valPromptRes] = await Promise.all([
      getAnthropicModels().catch(() => null),
      getCoinSearchPrompt().catch(() => null),
      getCoinShowsPrompt().catch(() => null),
      getValuationPrompt().catch(() => null),
    ])

    if (modelsRes?.data?.length) {
      anthropicModels.value = modelsRes.data
    }

    if (coinSearchRes?.data) {
      coinSearchPromptDefault.value = coinSearchRes.data.default
      if (!settings.value.CoinSearchPrompt) {
        settings.value.CoinSearchPrompt = coinSearchRes.data.prompt
      }
    }

    if (coinShowsRes?.data) {
      coinShowsPromptDefault.value = coinShowsRes.data.default
      if (!settings.value.CoinShowsPrompt) {
        settings.value.CoinShowsPrompt = coinShowsRes.data.prompt
      }
    }

    if (valPromptRes?.data) {
      valuationPromptDefault.value = valPromptRes.data.default
      if (!settings.value.ValuationPrompt) {
        settings.value.ValuationPrompt = valPromptRes.data.prompt
      }
    }
  } catch { /* use defaults */ }
}

async function saveSettings() {
  settingsSaving.value = true
  settingsMsg.value = ''
  settingsError.value = false
  availSettingsMsg.value = ''
  availSettingsError.value = false
  valSettingsMsg.value = ''
  valSettingsError.value = false
  try {
    const entries = Object.entries(settings.value).map(([key, value]) => ({ key, value: String(value) }))
    await updateAppSettings(entries)
    settingsMsg.value = 'Settings saved'
    availSettingsMsg.value = 'Settings saved'
    valSettingsMsg.value = 'Settings saved'
    setTimeout(() => { availSettingsMsg.value = ''; valSettingsMsg.value = '' }, 3000)
  } catch {
    settingsMsg.value = 'Failed to save settings'
    settingsError.value = true
    availSettingsMsg.value = 'Failed to save settings'
    availSettingsError.value = true
    valSettingsMsg.value = 'Failed to save settings'
    valSettingsError.value = true
  } finally {
    settingsSaving.value = false
  }
}

async function testOllamaConnection() {
  ollamaTesting.value = true
  ollamaTestResult.value = ''
  try {
    const res = await getOllamaStatus()
    ollamaTestOk.value = res.data.available
    ollamaTestResult.value = res.data.message
  } catch {
    ollamaTestOk.value = false
    ollamaTestResult.value = 'Failed to check Ollama status'
  } finally {
    ollamaTesting.value = false
  }
}

async function testAnthropicConn() {
  anthropicTesting.value = true
  anthropicTestResult.value = ''
  try {
    const res = await testAnthropicConnection()
    anthropicTestOk.value = res.data.available
    anthropicTestResult.value = res.data.message
  } catch {
    anthropicTestOk.value = false
    anthropicTestResult.value = 'Failed to test Anthropic connection'
  } finally {
    anthropicTesting.value = false
  }
}

async function testSearxngConn() {
  searxngTesting.value = true
  searxngTestResult.value = ''
  try {
    const res = await testSearXNGConnection()
    searxngTestOk.value = res.data.available
    searxngTestResult.value = res.data.message
  } catch {
    searxngTestOk.value = false
    searxngTestResult.value = 'Failed to test SearXNG connection'
  } finally {
    searxngTesting.value = false
  }
}

// Logs
const logs = ref<LogEntry[]>([])
const logsLoading = ref(false)
const logsFilter = ref('')
const logsAutoRefresh = ref(false)
let logsInterval: ReturnType<typeof setInterval> | null = null

async function loadLogs() {
  logsLoading.value = true
  try {
    const res = await getAdminLogs(500, logsFilter.value || undefined)
    logs.value = res.data.logs || []
  } catch { /* ignore */ } finally {
    logsLoading.value = false
  }
}

function toggleAutoRefresh() {
  logsAutoRefresh.value = !logsAutoRefresh.value
  if (logsAutoRefresh.value) {
    logsInterval = setInterval(loadLogs, 3000)
  } else if (logsInterval) {
    clearInterval(logsInterval)
    logsInterval = null
  }
}

function exportLogs() {
  if (logs.value.length === 0) return
  const lines = logs.value.map(
    (e) => `${e.timestamp} [${e.level.padEnd(5)}] ${e.message}`
  )
  const blob = new Blob([lines.join('\n')], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  const date = new Date().toISOString().slice(0, 10)
  a.download = `ancientcoins-logs-${date}.log`
  a.click()
  URL.revokeObjectURL(url)
}

function logLevelClass(level: string) {
  switch (level) {
    case 'ERROR': return 'log-error'
    case 'WARN': return 'log-warn'
    case 'DEBUG': return 'log-debug'
    case 'TRACE': return 'log-trace'
    default: return 'log-info'
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}

// Availability
const availSettingsMsg = ref('')
const availSettingsError = ref(false)
const availRuns = ref<AvailabilityRun[]>([])
const availTotal = ref(0)
const availPage = ref(1)
const availLoading = ref(false)
const expandedRunId = ref<number | null>(null)
const expandedResults = ref<AvailabilityRun['results']>(undefined)
const expandedLoading = ref(false)

async function loadAvailRuns() {
  availLoading.value = true
  try {
    const res = await getAvailabilityRuns(availPage.value, 5)
    availRuns.value = res.data.runs ?? []
    availTotal.value = res.data.total ?? 0
  } catch { /* ignore */ } finally {
    availLoading.value = false
  }
}

async function toggleRunDetail(runId: number) {
  if (expandedRunId.value === runId) {
    expandedRunId.value = null
    expandedResults.value = undefined
    return
  }
  expandedRunId.value = runId
  expandedResults.value = []
  expandedLoading.value = true
  try {
    const res = await getAvailabilityRunDetail(runId)
    expandedResults.value = res.data.results ?? []
  } catch {
    expandedResults.value = []
  } finally {
    expandedLoading.value = false
  }
}

// Valuation
const valSettingsMsg = ref('')
const valSettingsError = ref(false)
const valRuns = ref<ValuationRun[]>([])
const valTotal = ref(0)
const valPage = ref(1)
const valLoading = ref(false)
const valTriggerLoading = ref(false)
const valExpandedRunId = ref<number | null>(null)
const valExpandedResults = ref<ValuationRun['results']>(undefined)
const valExpandedLoading = ref(false)
let valPollTimer: ReturnType<typeof setInterval> | null = null

async function loadValRuns() {
  valLoading.value = true
  try {
    const res = await getValuationRuns(valPage.value, 5)
    valRuns.value = res.data.runs ?? []
    valTotal.value = res.data.total ?? 0

    // Auto-poll while any run is still "running"
    const hasRunning = valRuns.value.some(r => r.status === 'running')
    if (hasRunning && !valPollTimer) {
      valPollTimer = setInterval(() => { loadValRuns() }, 5000)
    } else if (!hasRunning && valPollTimer) {
      clearInterval(valPollTimer)
      valPollTimer = null
    }
  } catch { /* ignore */ } finally {
    valLoading.value = false
  }
}

async function toggleValRunDetail(runId: number) {
  if (valExpandedRunId.value === runId) {
    valExpandedRunId.value = null
    valExpandedResults.value = undefined
    return
  }
  valExpandedRunId.value = runId
  valExpandedResults.value = []
  valExpandedLoading.value = true
  try {
    const res = await getValuationRunDetail(runId)
    valExpandedResults.value = res.data.results ?? []
  } catch {
    valExpandedResults.value = []
  } finally {
    valExpandedLoading.value = false
  }
}

async function triggerManualValuation() {
  valTriggerLoading.value = true
  valSettingsMsg.value = ''
  valSettingsError.value = false
  try {
    await triggerValuation()
    valSettingsMsg.value = 'Valuation started — progress updates below'
    setTimeout(() => { valSettingsMsg.value = '' }, 10000)
    // Kick off initial reload; auto-poll will continue while run is active
    setTimeout(() => { loadValRuns() }, 2000)
  } catch {
    valSettingsMsg.value = 'Failed to trigger valuation'
    valSettingsError.value = true
  } finally {
    valTriggerLoading.value = false
  }
}

async function cancelRun(runId: number) {
  try {
    await cancelValuationRun(runId)
    valSettingsMsg.value = 'Cancellation requested'
    setTimeout(() => { valSettingsMsg.value = '' }, 5000)
    setTimeout(() => { loadValRuns() }, 1000)
  } catch {
    valSettingsMsg.value = 'Failed to cancel run'
    valSettingsError.value = true
  }
}

function formatDuration(ms: number) {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function truncateUrl(url: string) {
  try {
    const u = new URL(url)
    const path = u.pathname.length > 20 ? u.pathname.substring(0, 17) + '...' : u.pathname
    return u.hostname + path
  } catch {
    if (url.length <= 35) return url
    return url.substring(0, 32) + '...'
  }
}

onMounted(() => {
  loadUsers()
  loadSettings()
  loadAvailRuns()
  loadValRuns()
})

onUnmounted(() => {
  if (logsInterval) clearInterval(logsInterval)
  if (valPollTimer) clearInterval(valPollTimer)
})
</script>

<style scoped>
.admin-layout {
  max-width: 800px;
  margin-left: auto;
  margin-right: auto;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.tab-nav {
  display: flex;
  gap: 0.25rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 0.3rem;
}

.tab-btn {
  flex: 1;
  padding: 0.6rem 1rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tab-btn:hover:not(.active) {
  color: var(--text-primary);
}

.admin-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.users-table {
  width: 100%;
  border-collapse: collapse;
}

.users-table th,
.users-table td {
  text-align: left;
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.users-table th {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
}

.username {
  font-weight: 500;
}

.you-badge {
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-left: 0.3rem;
}

.date-cell {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.action-btns {
  display: flex;
  gap: 0.4rem;
}

.text-muted {
  color: var(--text-muted);
}

.form-hint {
  display: block;
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: #e74c3c;
}

.ai-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.prompt-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.25rem;
}

.prompt-header .form-label {
  margin-bottom: 0;
}

.btn-ghost {
  background: transparent;
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-ghost:hover:not(:disabled) {
  color: var(--accent-gold);
  border-color: var(--accent-gold);
}

.btn-ghost:disabled {
  opacity: 0.35;
  cursor: default;
}

.btn-xs {
  padding: 0.2rem 0.5rem;
  font-size: 0.7rem;
  border-radius: var(--radius-sm);
}

.connectivity-result {
  margin-top: 0.75rem;
  padding: 0.6rem 0.8rem;
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.connectivity-result.success {
  background: rgba(46, 204, 113, 0.1);
  border: 1px solid rgba(46, 204, 113, 0.3);
  color: #2ecc71;
}

.connectivity-result.error {
  background: rgba(231, 76, 60, 0.1);
  border: 1px solid rgba(231, 76, 60, 0.3);
  color: #e74c3c;
}

.connectivity-icon {
  font-size: 0.7rem;
}

/* Provider Toggle */
.provider-toggle {
  display: flex;
  gap: 1rem;
  margin-top: 0.5rem;
}

.provider-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 1rem;
  border: 2px solid var(--border-subtle, #333);
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.provider-option:hover {
  border-color: var(--accent-gold, #d4a843);
}

.provider-option.active {
  border-color: var(--accent-gold, #d4a843);
  background: rgba(212, 168, 67, 0.08);
}

.provider-option input[type="radio"] {
  display: none;
}

.provider-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary, #e0e0e0);
}

.provider-desc {
  font-size: 0.8rem;
  color: var(--text-secondary, #999);
}

.provider-warning {
  margin-top: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: rgba(231, 176, 60, 0.1);
  border: 1px solid rgba(231, 176, 60, 0.3);
  border-radius: 6px;
  color: #e7b03c;
  font-size: 0.85rem;
}

.section-divider {
  border: none;
  border-top: 1px solid var(--border-subtle, #333);
  margin: 1.5rem 0;
}

.subsection-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--text-primary, #e0e0e0);
}

.connectivity-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  margin-bottom: 0.5rem;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: 1rem;
}

.modal {
  width: 100%;
  max-width: 400px;
}

.modal h3 {
  margin-bottom: 1rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
}

@media (max-width: 640px) {
  .tab-nav {
    flex-wrap: wrap;
  }
  .users-table {
    font-size: 0.85rem;
  }
  .action-btns {
    flex-direction: column;
  }
}

/* Logs */
.logs-toolbar {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 1rem;
}

.logs-filter {
  width: auto;
  min-width: 120px;
}

.logs-container {
  max-height: 500px;
  overflow-y: auto;
  background: var(--bg-body);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.5rem;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.78rem;
  line-height: 1.5;
}

.logs-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-family: 'Inter', sans-serif;
}

.log-entry {
  display: flex;
  gap: 0.5rem;
  padding: 0.15rem 0.25rem;
  border-radius: 2px;
}

.log-entry:hover {
  background: var(--bg-card);
}

.log-time {
  color: var(--text-muted);
  flex-shrink: 0;
}

.log-level-badge {
  flex-shrink: 0;
  min-width: 48px;
  text-align: center;
  font-weight: 600;
  border-radius: 2px;
  padding: 0 4px;
}

.log-msg {
  word-break: break-word;
}

.log-error .log-level-badge { color: #e74c3c; }
.log-error .log-msg { color: #e74c3c; }
.log-warn .log-level-badge { color: #f39c12; }
.log-debug .log-level-badge { color: #3498db; }
.log-trace .log-level-badge { color: #7f8c8d; }
.log-info .log-level-badge { color: #2ecc71; }

.version-info {
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.78rem;
  color: var(--text-muted);
}

.version-label {
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.version-value {
  font-family: 'Courier New', Courier, monospace;
  color: var(--text-secondary);
}

.version-date {
  margin-left: 0.25rem;
}

/* Availability */
.avail-settings {
  margin-bottom: 1rem;
}

.avail-toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.avail-save-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 1rem;
}

.avail-save-msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
}

.avail-save-error {
  color: #e74c3c;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 42px;
  height: 22px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
  border-radius: 22px;
  transition: background 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  width: 16px;
  height: 16px;
  left: 2px;
  bottom: 2px;
  background: var(--text-secondary);
  border-radius: 50%;
  transition: transform 0.2s;
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
}

.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(20px);
  background: var(--accent-gold);
}

.avail-interval-input {
  max-width: 120px;
}

.avail-table {
  font-size: 0.82rem;
  table-layout: fixed;
  width: 100%;
}

.avail-row {
  cursor: pointer;
  transition: background var(--transition-fast);
}

.avail-row:hover {
  background: var(--bg-primary);
}

.avail-row-expanded {
  background: var(--bg-primary);
}

.avail-count-available { color: #2ecc71; font-weight: 600; }
.avail-count-unavailable { color: #e74c3c; font-weight: 600; }
.avail-count-unknown { color: #f1c40f; font-weight: 600; }

.avail-detail-row td {
  padding: 0.5rem;
  background: var(--bg-body);
  overflow: hidden;
}

.avail-detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.78rem;
  table-layout: fixed;
}

.avail-detail-table th,
.avail-detail-table td {
  padding: 0.4rem 0.5rem;
  text-align: left;
  border-bottom: 1px solid var(--border-subtle);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Column widths for detail table */
.avail-detail-table th:nth-child(1),
.avail-detail-table td:nth-child(1) { width: 22%; }
.avail-detail-table th:nth-child(2),
.avail-detail-table td:nth-child(2) { width: 22%; }
.avail-detail-table th:nth-child(3),
.avail-detail-table td:nth-child(3) { width: 10%; }
.avail-detail-table th:nth-child(4),
.avail-detail-table td:nth-child(4) { width: 28%; }
.avail-detail-table th:nth-child(5),
.avail-detail-table td:nth-child(5) { width: 8%; }
.avail-detail-table th:nth-child(6),
.avail-detail-table td:nth-child(6) { width: 10%; }

.avail-detail-table th {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
}

.avail-link {
  color: var(--accent-gold);
  text-decoration: none;
  font-size: 0.75rem;
}

.avail-link:hover {
  text-decoration: underline;
}

.avail-reason {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.listing-status-badge {
  display: inline-block;
  padding: 0.15rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}

.listing-available {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.listing-unavailable {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.listing-unknown {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
}

.avail-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-top: 1rem;
}

.avail-page-info {
  font-size: 0.82rem;
  color: var(--text-secondary);
}

/* Valuation */
.val-status-badge {
  display: inline-block;
  padding: 0.15rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}

.val-status-running {
  background: rgba(52, 152, 219, 0.15);
  color: #3498db;
}

.val-progress {
  margin-left: 0.35rem;
  font-size: 0.7rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.val-status-completed {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.val-status-failed {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.val-status-cancelled {
  background: rgba(243, 156, 18, 0.15);
  color: #f39c12;
}

.btn-cancel-run {
  margin-left: 0.4rem;
  padding: 0.1rem 0.4rem;
  font-size: 0.65rem;
  border: 1px solid rgba(231, 76, 60, 0.4);
  border-radius: var(--radius-full);
  background: transparent;
  color: #e74c3c;
  cursor: pointer;
  vertical-align: middle;
}
.btn-cancel-run:hover {
  background: rgba(231, 76, 60, 0.15);
}

.val-value {
  font-weight: 600;
  color: var(--accent-gold);
}

.val-confidence {
  display: inline-block;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 600;
}

.val-conf-high {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.val-conf-medium {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
}

.val-conf-low {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.val-result-success {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.val-result-skipped {
  background: rgba(149, 165, 166, 0.15);
  color: #95a5a6;
}

.val-result-error {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.val-detail-table th:nth-child(1),
.val-detail-table td:nth-child(1) { width: 22%; }
.val-detail-table th:nth-child(2),
.val-detail-table td:nth-child(2) { width: 12%; }
.val-detail-table th:nth-child(3),
.val-detail-table td:nth-child(3) { width: 12%; }
.val-detail-table th:nth-child(4),
.val-detail-table td:nth-child(4) { width: 10%; }
.val-detail-table th:nth-child(5),
.val-detail-table td:nth-child(5) { width: 10%; }
.val-detail-table th:nth-child(6),
.val-detail-table td:nth-child(6) { width: 34%; }
</style>
