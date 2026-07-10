<template>
  <section class="admin-section card">
    <h2>Schedules</h2>

    <!-- Wishlist Availability Check -->
    <h3 class="subsection-title">Wishlist Availability Check</h3>
    <p class="subsection-desc">Monitors dealer sites for coins on your wishlist and sends alerts when availability changes.</p>
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
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Schedule Settings' }}
        </button>
        <span v-if="availSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': availSettingsError }">{{ availSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm schedule-run-now" :disabled="availTriggerLoading" @click="triggerManualAvailabilityCheck()">
          {{ availTriggerLoading ? 'Queuing...' : 'Run Now' }}
        </button>
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
            <th class="hide-mobile">Trigger</th>
            <th class="hide-mobile">User</th>
            <th class="hide-mobile">Status</th>
            <th>Checked</th>
            <th class="hide-mobile">Avail</th>
            <th>Unavail</th>
            <th class="hide-mobile">Unknown</th>
            <th class="hide-mobile">Errors</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in availRuns" :key="run.id">
            <tr class="avail-row" :class="{ 'avail-row-expanded': expandedRunId === run.id }" @click="toggleRunDetail(run.id)">
              <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
              <td class="hide-mobile">{{ run.triggerType }}</td>
              <td class="hide-mobile">{{ run.userName || '—' }}</td>
              <td class="hide-mobile">
                <span v-if="run.status && run.status !== 'completed'" class="avail-status-badge" :class="'avail-status-' + run.status">{{ run.status }}</span>
                <span v-else class="avail-status-badge avail-status-completed">done</span>
              </td>
              <td>{{ run.coinsChecked }}</td>
              <td class="hide-mobile avail-count-available">{{ run.available }}</td>
              <td class="avail-count-unavailable">{{ run.unavailable }}</td>
              <td class="hide-mobile avail-count-unknown">{{ run.unknown }}</td>
              <td class="hide-mobile">{{ run.errors }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="expandedRunId === run.id && expandedResults" class="avail-detail-row">
              <td :colspan="availColspan">
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
                      <td>
                        <SafeExternalLink
                          v-if="safeRunUrl(r.url)"
                          :href="r.url"
                          target="_blank"
                          rel="noopener"
                          class="avail-link"
                          @click.stop
                        >
                          {{ truncateUrl(r.url) }}
                        </SafeExternalLink>
                        <span v-else class="text-muted">--</span>
                      </td>
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
        <button class="btn btn-secondary btn-sm" :disabled="availPage <= 1" @click="prevAvailPage()">Prev</button>
        <span class="avail-page-info">Page {{ availPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="availRuns.length < 5" @click="nextAvailPage()">Next</button>
      </div>
    </template>

    <hr class="section-divider" />

    <!-- Auction Ending Alerts -->
    <h3 class="subsection-title">Auction Ending Alerts</h3>
    <p class="subsection-desc">Checks watched auction lots that are ending soon and sends Pushover reminders before bidding closes.</p>
    <div class="avail-settings">
      <div class="form-group avail-toggle-row">
        <label class="form-label">Enable Automatic Alerts</label>
        <label class="toggle-switch">
          <input
            type="checkbox"
            :checked="settings.AuctionEndingCheckEnabled === 'true'"
            @change="settings.AuctionEndingCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="toggle-slider"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.AuctionEndingCheckStartTime"
          class="form-input avail-interval-input"
          type="time"
        />
        <span class="form-hint">The first ending-alert check runs at this time each day.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.AuctionEndingCheckInterval"
          class="form-input avail-interval-input"
          type="number"
          min="60"
          step="60"
        />
        <span class="form-hint">How often to check for lots ending soon after the start time. Default 1440 (daily).</span>
      </div>
      <div class="avail-save-row">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Alert Settings' }}
        </button>
        <span v-if="auctionSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': auctionSettingsError }">{{ auctionSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm schedule-run-now" :disabled="auctionTriggerLoading" @click="triggerManualAuctionCheck()">
          {{ auctionTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="section-divider" />
    <h3 class="subsection-title">Auction Ending Alert Run History</h3>

    <div v-if="auctionLoading" class="loading-overlay"><div class="spinner"></div></div>
    <div v-else-if="auctionRuns.length === 0" class="logs-empty">No auction ending alert runs recorded yet.</div>
    <template v-else>
      <table class="users-table avail-table">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hide-mobile">Trigger</th>
            <th>Lots</th>
            <th>Alerts</th>
            <th class="hide-mobile">Status</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in auctionRuns" :key="run.id">
            <tr>
              <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
              <td class="hide-mobile">
                <span class="listing-status-badge" :class="run.triggerType === 'manual' ? 'listing-unavailable' : 'listing-unknown'">
                  {{ run.triggerType }}
                </span>
              </td>
              <td>{{ run.lotsChecked }}</td>
              <td class="avail-count-available">{{ run.alertsSent }}</td>
              <td class="hide-mobile">
                <span class="listing-status-badge" :class="run.status === 'error' ? 'listing-unavailable' : (run.status === 'success' ? 'listing-available' : 'listing-unknown')">
                  {{ run.status }}
                </span>
              </td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="avail-pagination">
        <button class="btn btn-secondary btn-sm" :disabled="auctionPage <= 1" @click="prevAuctionPage()">Prev</button>
        <span class="avail-page-info">Page {{ auctionPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="auctionRuns.length < 5" @click="nextAuctionPage()">Next</button>
      </div>
    </template>

    <hr class="section-divider" />

    <!-- Auction Price Alerts and Bid Reminders -->
    <h3 class="subsection-title">Auction Price Alerts and Bid Reminders</h3>
    <p class="subsection-desc">Checks watched auction lots for price thresholds and close-time bid reminders.</p>
    <div class="avail-settings">
      <div class="form-group avail-toggle-row">
        <label class="form-label">Enable Automatic Checks</label>
        <label class="toggle-switch">
          <input
            type="checkbox"
            :checked="settings.AuctionAlertsCheckEnabled === 'true'"
            @change="settings.AuctionAlertsCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="toggle-slider"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.AuctionAlertsCheckStartTime"
          class="form-input avail-interval-input"
          type="time"
        />
        <span class="form-hint">The first price-alert and reminder check runs at this time each day.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.AuctionAlertsCheckInterval"
          class="form-input avail-interval-input"
          type="number"
          min="15"
          step="15"
        />
        <span class="form-hint">How often to re-check price thresholds and bid reminder windows. Default 60 (hourly).</span>
      </div>
      <div class="avail-save-row">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Alert and Reminder Settings' }}
        </button>
        <span v-if="alertReminderSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': alertReminderSettingsError }">{{ alertReminderSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm schedule-run-now" :disabled="alertReminderTriggerLoading" @click="triggerManualAlertReminderCheck()">
          {{ alertReminderTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="section-divider" />
    <h3 class="subsection-title">Auction Price Alert and Reminder Run History</h3>

    <div v-if="alertReminderLoading" class="loading-overlay"><div class="spinner"></div></div>
    <div v-else-if="alertReminderRuns.length === 0" class="logs-empty">No auction price alert or reminder runs recorded yet.</div>
    <template v-else>
      <table class="users-table avail-table">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hide-mobile">Trigger</th>
            <th>Lots</th>
            <th>Alerts</th>
            <th>Reminders</th>
            <th class="hide-mobile">Status</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in alertReminderRuns" :key="run.id">
            <tr>
              <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
              <td class="hide-mobile">
                <span class="listing-status-badge" :class="run.triggerType === 'manual' ? 'listing-unavailable' : 'listing-unknown'">
                  {{ run.triggerType }}
                </span>
              </td>
              <td>{{ run.lotsChecked ?? run.alertsChecked ?? 0 }}</td>
              <td class="avail-count-available">{{ run.priceAlertsTriggered ?? run.alertsSent ?? run.alertsTriggered ?? 0 }}</td>
              <td class="avail-count-available">{{ run.bidRemindersSent ?? run.remindersSent ?? run.remindersNotified ?? 0 }}</td>
              <td class="hide-mobile">
                <span class="listing-status-badge" :class="run.status === 'error' ? 'listing-unavailable' : (run.status === 'success' ? 'listing-available' : 'listing-unknown')">
                  {{ run.status }}
                </span>
              </td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="avail-pagination">
        <button class="btn btn-secondary btn-sm" :disabled="alertReminderPage <= 1" @click="prevAlertReminderPage()">Prev</button>
        <span class="avail-page-info">Page {{ alertReminderPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="alertReminderRuns.length < 5" @click="nextAlertReminderPage()">Next</button>
      </div>
    </template>

    <hr class="section-divider" />

    <!-- Auction Watch Bid Digest -->
    <h3 class="subsection-title">Auction Watch Bid Digest</h3>
    <p class="subsection-desc">Refreshes NumisBids and CNG watched lots, updates current high bids in Auctions, and sends one Pushover digest while lots are active.</p>
    <div class="avail-settings">
      <div class="form-group avail-toggle-row">
        <label class="form-label">Enable Automatic Digests</label>
        <label class="toggle-switch">
          <input
            type="checkbox"
            :checked="settings.AuctionWatchBidDigestEnabled === 'true'"
            @change="settings.AuctionWatchBidDigestEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="toggle-slider"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.AuctionWatchBidDigestStartTime"
          class="form-input avail-interval-input"
          type="time"
        />
        <span class="form-hint">The first digest run starts at this time each day.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.AuctionWatchBidDigestInterval"
          class="form-input avail-interval-input"
          type="number"
          min="60"
          step="60"
        />
        <span class="form-hint">How often to refresh watched lots and send the digest after the start time. Default 1440 (daily).</span>
      </div>
      <div class="avail-save-row">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Digest Settings' }}
        </button>
        <span v-if="watchBidDigestSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': watchBidDigestSettingsError }">{{ watchBidDigestSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm schedule-run-now" :disabled="watchBidDigestTriggerLoading" @click="triggerManualWatchBidDigest()">
          {{ watchBidDigestTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="section-divider" />
    <h3 class="subsection-title">Auction Watch Bid Digest Run History</h3>

    <div v-if="watchBidDigestLoading" class="loading-overlay"><div class="spinner"></div></div>
    <div v-else-if="watchBidDigestRuns.length === 0" class="logs-empty">No auction watch bid digest runs recorded yet.</div>
    <template v-else>
      <table class="users-table avail-table">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hide-mobile">Trigger</th>
            <th>Lots</th>
            <th>Digests</th>
            <th class="hide-mobile">Status</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in watchBidDigestRuns" :key="run.id">
            <tr>
              <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
              <td class="hide-mobile">
                <span class="listing-status-badge" :class="run.triggerType === 'manual' ? 'listing-unavailable' : 'listing-unknown'">
                  {{ run.triggerType }}
                </span>
              </td>
              <td>{{ run.lotsChecked }}</td>
              <td class="avail-count-available">{{ run.digestsSent }}</td>
              <td class="hide-mobile">
                <span class="listing-status-badge" :class="run.status === 'error' ? 'listing-unavailable' : (run.status === 'success' ? 'listing-available' : 'listing-unknown')">
                  {{ run.status }}
                </span>
              </td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="avail-pagination">
        <button class="btn btn-secondary btn-sm" :disabled="watchBidDigestPage <= 1" @click="prevWatchBidDigestPage()">Prev</button>
        <span class="avail-page-info">Page {{ watchBidDigestPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="watchBidDigestRuns.length < 5" @click="nextWatchBidDigestPage()">Next</button>
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
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Valuation Settings' }}
        </button>
        <span v-if="valSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': valSettingsError }">{{ valSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm schedule-run-now" :disabled="valTriggerLoading" @click="triggerManualValuation()">
          {{ valTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
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
            <th class="hide-mobile">Trigger</th>
            <th>Status</th>
            <th>Checked</th>
            <th class="hide-mobile">Updated</th>
            <th class="hide-mobile">Skipped</th>
            <th class="hide-mobile">Errors</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in valRuns" :key="run.id">
            <tr class="avail-row" :class="{ 'avail-row-expanded': valExpandedRunId === run.id }" @click="toggleValRunDetail(run.id)">
              <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
              <td class="hide-mobile">{{ run.triggerType }}</td>
              <td>
                <span class="val-status-badge" :class="'val-status-' + run.status">{{ run.status }}</span>
                <span v-if="run.status === 'running' && run.totalCoins > 0" class="val-progress">
                  {{ run.coinsChecked + run.coinsSkipped + run.errors }} / {{ run.totalCoins }}
                </span>
                <button v-if="run.status === 'running'" class="btn-cancel-run" @click.stop="cancelRun(run.id)">Cancel</button>
              </td>
              <td>{{ run.coinsChecked }}</td>
              <td class="hide-mobile avail-count-available">{{ run.coinsUpdated }}</td>
              <td class="hide-mobile avail-count-unknown">{{ run.coinsSkipped }}</td>
              <td class="hide-mobile avail-count-unavailable">{{ run.errors }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="valExpandedRunId === run.id && valExpandedResults" class="avail-detail-row">
              <td :colspan="valColspan">
                <div v-if="valExpandedLoading" class="loading-overlay"><div class="spinner"></div></div>
                <table v-else-if="valExpandedResults.length" class="avail-detail-table val-detail-table">
                  <thead>
                    <tr>
                      <th>Coin</th>
                      <th>Previous</th>
                      <th>Estimated</th>
                      <th>Confidence</th>
                      <th>Status</th>
                      <th>Explanation</th>
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
                      <td class="avail-reason">
                        <div v-if="r.changeExplanation" class="val-change-explanation">{{ r.changeExplanation }}</div>
                        <div>{{ r.reasoning || r.errorMessage || '--' }}</div>
                      </td>
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
        <button class="btn btn-secondary btn-sm" :disabled="valPage <= 1" @click="prevValPage()">Prev</button>
        <span class="avail-page-info">Page {{ valPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="valRuns.length < 5" @click="nextValPage()">Next</button>
      </div>
    </template>

    <hr class="section-divider" />

    <!-- Collection Health Snapshots -->
    <h3 class="subsection-title">Collection Health Snapshots</h3>
    <p class="subsection-desc">Captures daily health baselines used by the 30-day collection health trend.</p>
    <div class="avail-settings">
      <div class="form-group avail-toggle-row">
        <label class="form-label">Enable Daily Snapshots</label>
        <label class="toggle-switch">
          <input
            type="checkbox"
            :checked="settings.CollectionHealthSnapshotsEnabled === 'true'"
            @change="settings.CollectionHealthSnapshotsEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="toggle-slider"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily)</label>
        <input
          v-model="settings.CollectionHealthSnapshotsStartTime"
          class="form-input avail-interval-input"
          type="time"
        />
        <span class="form-hint">Time of day when collection health baselines are captured for trend calculations.</span>
      </div>
      <div class="avail-save-row">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Snapshot Settings' }}
        </button>
        <span v-if="healthSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': healthSettingsError }">{{ healthSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm schedule-run-now" :disabled="healthTriggerLoading" @click="triggerManualHealthSnapshots()">
          {{ healthTriggerLoading ? 'Running...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="section-divider" />

    <!-- Coin of the Day -->
    <h3 class="subsection-title">Coin of the Day</h3>
    <p class="subsection-desc">Picks one coin per day from each user's collection and sends an in-app and Pushover notification. Each coin in a user's collection appears once before any coin repeats.</p>
    <div class="avail-settings">
      <div class="form-group avail-toggle-row">
        <label class="form-label">Enable Daily Feature</label>
        <label class="toggle-switch">
          <input
            type="checkbox"
            :checked="settings.CoinOfDayEnabled === 'true'"
            @change="settings.CoinOfDayEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="toggle-slider"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily)</label>
        <input
          v-model="settings.CoinOfDayStartTime"
          class="form-input avail-interval-input"
          type="time"
        />
        <span class="form-hint">Time of day when the daily featured coin is picked for each enrolled user.</span>
      </div>
      <div class="avail-save-row">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Coin of the Day Settings' }}
        </button>
        <span v-if="cotdSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': cotdSettingsError }">{{ cotdSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm schedule-run-now" :disabled="cotdTriggerLoading" @click="triggerManualCoinOfDay()">
          {{ cotdTriggerLoading ? 'Running...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="section-divider" />
    <h3 class="subsection-title">Coin of the Day Run History</h3>
    <div v-if="cotdLoading" class="loading-overlay"><div class="spinner"></div></div>
    <div v-else-if="cotdRuns.length === 0" class="logs-empty">No Coin of the Day runs recorded yet.</div>
    <template v-else>
      <table class="users-table avail-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Status</th>
            <th>Picked</th>
            <th>Skipped</th>
            <th>Errors</th>
            <th class="hide-mobile">Trigger</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="run in cotdRuns" :key="run.id">
            <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
            <td>{{ run.status }}</td>
            <td>{{ run.picked }}</td>
            <td>{{ run.skipped }}</td>
            <td>{{ run.errors }}</td>
            <td class="hide-mobile">{{ run.triggerType }}</td>
          </tr>
        </tbody>
      </table>

      <div class="avail-pagination">
        <button class="btn btn-secondary btn-sm" :disabled="cotdPage <= 1" @click="prevCoinOfDayPage()">Prev</button>
        <span class="avail-page-info">Page {{ cotdPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="cotdRuns.length < 5" @click="nextCoinOfDayPage()">Next</button>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import {
  getAvailabilityRuns, getAvailabilityRunDetail,
  triggerAvailabilityCheck,
  getValuationRuns, getValuationRunDetail, triggerValuation, cancelValuationRun,
  getAuctionEndingRuns, getAuctionEndingRun, triggerAuctionEndingCheck,
  getAuctionAlertReminderRuns, triggerAuctionAlertReminderCheck,
  getAuctionWatchBidDigestRuns, triggerAuctionWatchBidDigest,
  triggerCollectionHealthSnapshots,
  triggerCoinOfDayRun, getCoinOfDayRuns, getCoinOfDayRunDetail,
} from '@/api/client'
import { useRunHistoryPagination } from '@/composables/useRunHistoryPagination'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import type { AppSettings, AvailabilityRun, ValuationRun, AuctionEndingRun, AuctionAlertReminderRun, AuctionWatchBidDigestRun, CoinOfDayRun } from '@/types'

// Props are type-checked but not referenced directly in script
const _props = defineProps<{
  settings: AppSettings
  settingsSaving: boolean
  availSettingsMsg: string
  availSettingsError: boolean
  auctionSettingsMsg: string
  auctionSettingsError: boolean
  alertReminderSettingsMsg: string
  alertReminderSettingsError: boolean
  watchBidDigestSettingsMsg: string
  watchBidDigestSettingsError: boolean
  healthSettingsMsg: string
  healthSettingsError: boolean
  valSettingsMsg: string
  valSettingsError: boolean
}>()

const emit = defineEmits<{
  save: []
  'update:valSettingsMsg': [val: string]
  'update:valSettingsError': [val: boolean]
  'update:auctionSettingsMsg': [val: string]
  'update:auctionSettingsError': [val: boolean]
  'update:alertReminderSettingsMsg': [val: string]
  'update:alertReminderSettingsError': [val: boolean]
  'update:watchBidDigestSettingsMsg': [val: string]
  'update:watchBidDigestSettingsError': [val: boolean]
  'update:availSettingsMsg': [val: string]
  'update:availSettingsError': [val: boolean]
  'update:healthSettingsMsg': [val: string]
  'update:healthSettingsError': [val: boolean]
}>()

// Availability
const isMobile = ref(window.innerWidth <= 600)
const availColspan = computed(() => isMobile.value ? 4 : 10)
const valColspan = computed(() => isMobile.value ? 4 : 8)

function safeRunUrl(url: string | null | undefined): string | null {
  return sanitizeExternalUrl(url)
}

function onResize() { isMobile.value = window.innerWidth <= 600 }

const {
  runs: availRuns,
  total: _availTotal,
  page: availPage,
  loading: availLoading,
  loadRuns: loadAvailRuns,
  prevPage: prevAvailPage,
  nextPage: nextAvailPage,
} = useRunHistoryPagination<AvailabilityRun>(async (page, limit) => {
  const res = await getAvailabilityRuns(page, limit)
  return res.data ?? {}
})
const expandedRunId = ref<number | null>(null)
const expandedResults = ref<AvailabilityRun['results']>(undefined)
const expandedLoading = ref(false)
const availTriggerLoading = ref(false)

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

async function loadAvailRunsWithPoll() {
  try {
    await loadAvailRuns()
    const hasActive = availRuns.value.some(r => r.status === 'queued' || r.status === 'running')
    if (hasActive && !availPollTimer) {
      availPollTimer = setInterval(() => { loadAvailRunsWithPoll() }, 4000)
    } else if (!hasActive && availPollTimer) {
      clearInterval(availPollTimer)
      availPollTimer = null
    }
  } catch { /* ignore */ }
}

async function triggerManualAvailabilityCheck() {
  availTriggerLoading.value = true
  emit('update:availSettingsMsg', '')
  emit('update:availSettingsError', false)
  try {
    const res = await triggerAvailabilityCheck()
    emit('update:availSettingsMsg', `Run #${res.data.runId} queued — history updates below`)
    timers.push(setTimeout(() => { emit('update:availSettingsMsg', '') }, 12000))
    timers.push(setTimeout(() => { loadAvailRunsWithPoll() }, 1000))
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } })?.response?.status
    if (status === 409) {
      emit('update:availSettingsMsg', 'A manual availability run is already in progress')
    } else {
      emit('update:availSettingsMsg', 'Failed to queue availability check')
    }
    emit('update:availSettingsError', true)
  } finally {
    availTriggerLoading.value = false
  }
}

// Auction Ending
const {
  runs: auctionRuns,
  total: _auctionTotal,
  page: auctionPage,
  loading: auctionLoading,
  loadRuns: loadAuctionRunsBase,
  prevPage: prevAuctionPage,
  nextPage: nextAuctionPage,
} = useRunHistoryPagination<AuctionEndingRun>(async (page, limit) => {
  const res = await getAuctionEndingRuns(page, limit)
  return res.data ?? {}
})
const auctionTriggerLoading = ref(false)
let auctionPollTimer: ReturnType<typeof setInterval> | null = null

async function loadAuctionRuns() {
  try {
    await loadAuctionRunsBase()
    const hasActive = auctionRuns.value.some(r => r.status === 'queued' || r.status === 'running')
    if (hasActive && !auctionPollTimer) {
      auctionPollTimer = setInterval(() => { loadAuctionRuns() }, 3000)
    } else if (!hasActive && auctionPollTimer) {
      clearInterval(auctionPollTimer)
      auctionPollTimer = null
    }
  } catch { /* ignore */ }
}

async function triggerManualAuctionCheck() {
  auctionTriggerLoading.value = true
  emit('update:auctionSettingsMsg', '')
  emit('update:auctionSettingsError', false)
  try {
    const res = await triggerAuctionEndingCheck()
    const { runId, status } = res.data
    if (status === 'queued' || status === 'running') {
      emit('update:auctionSettingsMsg', `Run #${runId} queued — checking for results…`)
      // Poll until run reaches a terminal state
      const pollCompleted = async () => {
        try {
          const runRes = await getAuctionEndingRun(runId)
          const run = runRes.data
          if (run.status === 'success') {
            const durationSec = run.durationMs != null ? ` in ${(run.durationMs / 1000).toFixed(1)}s` : ''
            emit('update:auctionSettingsMsg', `Run #${runId} completed — ${run.lotsChecked} lots checked, ${run.alertsSent} alerts sent${durationSec}`)
            timers.push(setTimeout(() => { emit('update:auctionSettingsMsg', '') }, 10000))
            loadAuctionRuns()
          } else if (run.status === 'error') {
            emit('update:auctionSettingsMsg', `Run #${runId} failed`)
            emit('update:auctionSettingsError', true)
            loadAuctionRuns()
          } else {
            timers.push(setTimeout(pollCompleted, 2000))
          }
        } catch {
          loadAuctionRuns()
        }
      }
      timers.push(setTimeout(pollCompleted, 1500))
    } else if (status === 'error') {
      emit('update:auctionSettingsMsg', `Run #${runId} failed`)
      emit('update:auctionSettingsError', true)
      timers.push(setTimeout(() => { loadAuctionRuns() }, 1000))
    } else {
      timers.push(setTimeout(() => { loadAuctionRuns() }, 2000))
    }
  } catch {
    emit('update:auctionSettingsMsg', 'Failed to trigger auction ending alerts')
    emit('update:auctionSettingsError', true)
  } finally {
    auctionTriggerLoading.value = false
  }
}

// Auction Price Alerts and Bid Reminders
const {
  runs: alertReminderRuns,
  total: _alertReminderTotal,
  page: alertReminderPage,
  loading: alertReminderLoading,
  loadRuns: loadAlertReminderRuns,
  prevPage: prevAlertReminderPage,
  nextPage: nextAlertReminderPage,
} = useRunHistoryPagination<AuctionAlertReminderRun>(async (page, limit) => {
  const res = await getAuctionAlertReminderRuns(page, limit)
  return res.data ?? {}
})
const alertReminderTriggerLoading = ref(false)

async function triggerManualAlertReminderCheck() {
  alertReminderTriggerLoading.value = true
  emit('update:alertReminderSettingsMsg', '')
  emit('update:alertReminderSettingsError', false)
  try {
    const res = await triggerAuctionAlertReminderCheck()
    const { message, runId, alertsTriggered, alertsSent, priceAlertsTriggered, remindersNotified, remindersSent, bidRemindersSent, status, durationMs } = res.data
    if (status === 'error') {
      emit('update:alertReminderSettingsMsg', runId ? `Run #${runId} failed` : 'Alert and reminder check failed')
      emit('update:alertReminderSettingsError', true)
    } else if (message) {
      emit('update:alertReminderSettingsMsg', message)
    } else {
      const alertCount = priceAlertsTriggered ?? alertsSent ?? alertsTriggered ?? 0
      const reminderCount = bidRemindersSent ?? remindersSent ?? remindersNotified ?? 0
      const durationPart = durationMs != null ? ` in ${(durationMs / 1000).toFixed(1)}s` : ''
      emit('update:alertReminderSettingsMsg', `Run${runId ? ` #${runId}` : ''} completed — ${alertCount} alerts, ${reminderCount} reminders${durationPart}`)
    }
    timers.push(setTimeout(() => { emit('update:alertReminderSettingsMsg', '') }, 10000))
    timers.push(setTimeout(() => { loadAlertReminderRuns() }, 2000))
  } catch {
    emit('update:alertReminderSettingsMsg', 'Failed to trigger auction price alerts and bid reminders')
    emit('update:alertReminderSettingsError', true)
  } finally {
    alertReminderTriggerLoading.value = false
  }
}

// Auction Watch Bid Digest
const {
  runs: watchBidDigestRuns,
  total: _watchBidDigestTotal,
  page: watchBidDigestPage,
  loading: watchBidDigestLoading,
  loadRuns: loadWatchBidDigestRuns,
  prevPage: prevWatchBidDigestPage,
  nextPage: nextWatchBidDigestPage,
} = useRunHistoryPagination<AuctionWatchBidDigestRun>(async (page, limit) => {
  const res = await getAuctionWatchBidDigestRuns(page, limit)
  return res.data ?? {}
})
const watchBidDigestTriggerLoading = ref(false)

async function triggerManualWatchBidDigest() {
  watchBidDigestTriggerLoading.value = true
  emit('update:watchBidDigestSettingsMsg', '')
  emit('update:watchBidDigestSettingsError', false)
  try {
    const res = await triggerAuctionWatchBidDigest()
    emit('update:watchBidDigestSettingsMsg', res.data.message ?? 'Auction watch bid digest started')
    timers.push(setTimeout(() => { emit('update:watchBidDigestSettingsMsg', '') }, 10000))
    timers.push(setTimeout(() => { loadWatchBidDigestRuns() }, 2000))
  } catch {
    emit('update:watchBidDigestSettingsMsg', 'Failed to trigger auction watch bid digest')
    emit('update:watchBidDigestSettingsError', true)
  } finally {
    watchBidDigestTriggerLoading.value = false
  }
}

// Valuation
const {
  runs: valRuns,
  total: _valTotal,
  page: valPage,
  loading: valLoading,
  loadRuns: loadValRunsBase,
  prevPage: prevValPage,
  nextPage: nextValPage,
} = useRunHistoryPagination<ValuationRun>(async (page, limit) => {
  const res = await getValuationRuns(page, limit)
  return res.data ?? {}
})
const valTriggerLoading = ref(false)
const valExpandedRunId = ref<number | null>(null)
const valExpandedResults = ref<ValuationRun['results']>(undefined)
const valExpandedLoading = ref(false)
let valPollTimer: ReturnType<typeof setInterval> | null = null
let availPollTimer: ReturnType<typeof setInterval> | null = null
const timers: ReturnType<typeof setTimeout>[] = []

async function loadValRuns() {
  try {
    await loadValRunsBase()

    const hasRunning = valRuns.value.some(r => r.status === 'running')
    if (hasRunning && !valPollTimer) {
      valPollTimer = setInterval(() => { loadValRuns() }, 5000)
    } else if (!hasRunning && valPollTimer) {
      clearInterval(valPollTimer)
      valPollTimer = null
    }
  } catch { /* ignore */ }
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
  emit('update:valSettingsMsg', '')
  emit('update:valSettingsError', false)
  try {
    await triggerValuation()
    emit('update:valSettingsMsg', 'Valuation started — progress updates below')
    timers.push(setTimeout(() => { emit('update:valSettingsMsg', '') }, 10000))
    timers.push(setTimeout(() => { loadValRuns() }, 2000))
  } catch {
    emit('update:valSettingsMsg', 'Failed to trigger valuation')
    emit('update:valSettingsError', true)
  } finally {
    valTriggerLoading.value = false
  }
}

async function cancelRun(runId: number) {
  try {
    await cancelValuationRun(runId)
    emit('update:valSettingsMsg', 'Cancellation requested')
    timers.push(setTimeout(() => { emit('update:valSettingsMsg', '') }, 5000))
    timers.push(setTimeout(() => { loadValRuns() }, 1000))
  } catch {
    emit('update:valSettingsMsg', 'Failed to cancel run')
    emit('update:valSettingsError', true)
  }
}

// Collection Health Snapshots
const healthTriggerLoading = ref(false)

async function triggerManualHealthSnapshots() {
  healthTriggerLoading.value = true
  emit('update:healthSettingsMsg', '')
  emit('update:healthSettingsError', false)
  try {
    const res = await triggerCollectionHealthSnapshots()
    const { message, users, snapshotsCreated, skipped, errors, durationMs } = res.data
    const parts = [
      snapshotsCreated != null ? `${snapshotsCreated} snapshots` : null,
      users != null ? `${users} users` : null,
      skipped != null ? `${skipped} skipped` : null,
      errors ? `${errors} errors` : null,
      durationMs != null ? `${(durationMs / 1000).toFixed(1)}s` : null,
    ].filter((part): part is string => part != null)
    emit('update:healthSettingsMsg', message ?? (parts.length ? `Snapshot run complete — ${parts.join(', ')}` : 'Snapshot run complete'))
    if (errors) {
      emit('update:healthSettingsError', true)
    }
    timers.push(setTimeout(() => { emit('update:healthSettingsMsg', '') }, 10000))
  } catch {
    emit('update:healthSettingsMsg', 'Failed to run collection health snapshots')
    emit('update:healthSettingsError', true)
  } finally {
    healthTriggerLoading.value = false
  }
}

// Coin of the Day
const cotdTriggerLoading = ref(false)
const cotdSettingsMsg = ref('')
const cotdSettingsError = ref(false)
const {
  runs: cotdRuns,
  total: _cotdTotal,
  page: cotdPage,
  loading: cotdLoading,
  loadRuns: loadCoinOfDayRuns,
  prevPage: prevCoinOfDayPage,
  nextPage: nextCoinOfDayPage,
} = useRunHistoryPagination<CoinOfDayRun>(async (page, limit) => {
  const res = await getCoinOfDayRuns(page, limit)
  return res.data ?? {}
})
let cotdPollTimer: ReturnType<typeof setInterval> | null = null

function coinOfDayRunIsTerminal(status: string) {
  return status === 'completed' || status === 'failed'
}

function refreshCoinOfDayPolling() {
  const hasActive = cotdRuns.value.some((run) => !coinOfDayRunIsTerminal(run.status))
  if (hasActive && !cotdPollTimer) {
    cotdPollTimer = setInterval(() => { loadCoinOfDayRuns() }, 5000)
  } else if (!hasActive && cotdPollTimer) {
    clearInterval(cotdPollTimer)
    cotdPollTimer = null
  }

  watch(cotdRuns, () => {
    refreshCoinOfDayPolling()
  })
}

async function triggerManualCoinOfDay() {
  cotdTriggerLoading.value = true
  cotdSettingsMsg.value = ''
  cotdSettingsError.value = false
  try {
    const res = await triggerCoinOfDayRun()
    const runId = Number(res.data.runId ?? 0)
    cotdSettingsMsg.value = runId ? `Coin of the Day run #${runId} queued` : 'Coin of the Day run queued'
    if (runId) {
      const detail = await getCoinOfDayRunDetail(runId)
      const run = detail.data
      if (coinOfDayRunIsTerminal(run.status)) {
        cotdSettingsMsg.value = `Picked ${run.picked}, skipped ${run.skipped}${run.errors ? `, errors ${run.errors}` : ''}`
        cotdSettingsError.value = run.status === 'failed'
      }
    }
    await loadCoinOfDayRuns()
    refreshCoinOfDayPolling()
    timers.push(setTimeout(() => { cotdSettingsMsg.value = '' }, 10000))
  } catch {
    cotdSettingsMsg.value = 'Failed to run Coin of the Day'
    cotdSettingsError.value = true
  } finally {
    cotdTriggerLoading.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
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
  window.addEventListener('resize', onResize)
  loadAvailRunsWithPoll()
  loadAuctionRuns()
  loadAlertReminderRuns()
  loadWatchBidDigestRuns()
  loadValRuns()
  loadCoinOfDayRuns()
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  if (valPollTimer) clearInterval(valPollTimer)
  if (auctionPollTimer) clearInterval(auctionPollTimer)
  if (availPollTimer) clearInterval(availPollTimer)
  if (cotdPollTimer) clearInterval(cotdPollTimer)
  timers.forEach(clearTimeout)
})
</script>

<style scoped>
.admin-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.subsection-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--text-primary, #e0e0e0);
}

.section-divider {
  border: none;
  border-top: 1px solid var(--border-subtle, #333);
  margin: 1.5rem 0;
}

.form-hint {
  display: block;
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}

.logs-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-family: 'Inter', sans-serif;
}

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
  width: 100%;
}

.avail-save-msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin-right: auto;
}

.schedule-run-now {
  margin-left: auto;
}

.avail-save-error {
  color: var(--color-negative);
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

.date-cell {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.text-muted {
  color: var(--text-muted);
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

.avail-count-available { color: var(--color-positive); font-weight: 600; }
.avail-count-unavailable { color: var(--color-negative); font-weight: 600; }
.avail-count-unknown { color: var(--text-warning); font-weight: 600; }

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

.val-change-explanation {
  margin-bottom: 0.35rem;
  color: var(--accent-gold);
  font-weight: 500;
}

.listing-status-badge {
  display: inline-block;
  padding: 0.15rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}

.avail-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  margin-top: 1rem;
}

.avail-page-info {
  font-size: 0.82rem;
  color: var(--text-secondary);
}

.listing-available {
  background: rgba(46, 204, 113, 0.15);
  color: var(--color-positive);
}

.listing-unavailable {
  background: rgba(231, 76, 60, 0.15);
  color: var(--color-negative);
}

.listing-unknown {
  background: rgba(241, 196, 15, 0.15);
  color: var(--text-warning);
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
  color: var(--color-positive);
}

.val-status-failed {
  background: rgba(231, 76, 60, 0.15);
  color: var(--color-negative);
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
  color: var(--color-negative);
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
  border-radius: var(--radius-sm);
  font-size: 0.7rem;
  font-weight: 600;
}

.val-conf-high {
  background: rgba(46, 204, 113, 0.15);
  color: var(--confidence-high);
}

.val-conf-medium {
  background: rgba(241, 196, 15, 0.15);
  color: var(--confidence-medium);
}

.val-conf-low {
  background: rgba(231, 76, 60, 0.15);
  color: var(--confidence-low);
}

.val-result-success {
  background: rgba(46, 204, 113, 0.15);
  color: var(--color-positive);
}

.val-result-skipped {
  background: rgba(149, 165, 166, 0.15);
  color: #95a5a6;
}

.val-result-error {
  background: rgba(231, 76, 60, 0.15);
  color: var(--color-negative);
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

.auction-detail-table th:nth-child(1),
.auction-detail-table td:nth-child(1) { width: 12%; }
.auction-detail-table th:nth-child(2),
.auction-detail-table td:nth-child(2) { width: 26%; }
.auction-detail-table th:nth-child(3),
.auction-detail-table td:nth-child(3) { width: 18%; }
.auction-detail-table th:nth-child(4),
.auction-detail-table td:nth-child(4) { width: 12%; }
.auction-detail-table th:nth-child(5),
.auction-detail-table td:nth-child(5) { width: 10%; }
.auction-detail-table th:nth-child(6),
.auction-detail-table td:nth-child(6) { width: 22%; }

/* Mobile responsive: hide non-essential columns */
@media (max-width: 600px) {
  .hide-mobile {
    display: none !important;
  }

  .avail-table {
    table-layout: auto;
    font-size: 0.8rem;
  }

  .users-table th,
  .users-table td {
    padding: 0.5rem 0.35rem;
  }

  .date-cell {
    font-size: 0.8rem;
  }
}

.avail-status-badge {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  font-size: 0.72rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.avail-status-queued {
  background: rgba(201, 168, 76, 0.15);
  color: var(--accent-gold);
}

.avail-status-running {
  background: rgba(52, 152, 219, 0.15);
  color: #5dade2;
}

.avail-status-completed {
  background: rgba(46, 204, 113, 0.12);
  color: #58d68d;
}

.avail-status-failed {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}
</style>
