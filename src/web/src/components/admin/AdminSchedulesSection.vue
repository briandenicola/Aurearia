<template>
  <section class="admin-section card flex flex-col">
    <h2 class="mb-5 border-b border-border-subtle pb-3 text-xl font-medium">Schedules</h2>

    <!-- Wishlist Availability Check -->
    <h3 class="mb-4 text-base font-semibold text-text-primary">Wishlist Availability Check</h3>
    <p class="mb-4 text-base text-text-secondary">Monitors dealer sites for coins on your wishlist and sends alerts when availability changes.</p>
    <div class="mb-4">
      <div class="form-group flex items-center justify-between gap-3">
        <label class="form-label">Enable Automatic Checks</label>
        <label class="relative inline-block h-[22px] w-[42px]">
          <input
            class="peer sr-only" type="checkbox"
            :checked="settings.WishlistCheckEnabled === 'true'"
            @change="settings.WishlistCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.WishlistCheckStartTime"
          class="form-input w-full max-w-[120px]"
          type="time"
        />
        <span class="form-hint">The first check runs at this time each day. Subsequent checks repeat at the interval below.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.WishlistCheckInterval"
          class="form-input w-full max-w-[120px]"
          type="number"
          min="5"
          step="5"
        />
        <span class="form-hint">How often to repeat after the start time (e.g. 120 = every 2 hours).</span>
      </div>
      <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Schedule Settings' }}
        </button>
        <span v-if="availSettingsMsg" class="text-body text-gold md:mr-auto" :class="availSettingsError ? 'text-[var(--color-negative)]' : ''">{{ availSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="availTriggerLoading" @click="triggerManualAvailabilityCheck()">
          {{ availTriggerLoading ? 'Queuing...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="my-6 border-0 border-t border-border-subtle" />
    <h3 class="mb-4 text-base font-semibold text-text-primary">Availability Run History</h3>

    <div v-if="availLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
    <div v-else-if="availRuns.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No availability runs recorded yet.</div>
    <template v-else>
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th class="hidden md:table-cell">User</th>
            <th class="hidden md:table-cell">Status</th>
            <th>Checked</th>
            <th class="hidden md:table-cell">Avail</th>
            <th>Unavail</th>
            <th class="hidden md:table-cell">Unknown</th>
            <th class="hidden md:table-cell">Errors</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in availRuns" :key="run.id">
            <tr class="cursor-pointer transition-colors hover:bg-surface" :class="{ 'bg-surface': expandedRunId === run.id }" @click="toggleRunDetail(run.id)">
              <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
              <td class="hidden md:table-cell">{{ run.triggerType }}</td>
              <td class="hidden md:table-cell">{{ run.userName || '—' }}</td>
              <td class="hidden md:table-cell">
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-[0.72rem] font-medium uppercase tracking-[0.04em]" :class="run.status === 'queued' ? 'bg-[rgba(201,168,76,0.15)] text-gold' : run.status === 'running' ? 'bg-[rgba(52,152,219,0.15)] text-[#5dade2]' : run.status === 'failed' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(46,204,113,0.12)] text-[#58d68d]'">{{ run.status === 'completed' ? 'done' : run.status }}</span>
              </td>
              <td>{{ run.coinsChecked }}</td>
              <td class="hidden font-semibold text-[var(--color-positive)] md:table-cell">{{ run.available }}</td>
              <td class="font-semibold text-[var(--color-negative)]">{{ run.unavailable }}</td>
              <td class="hidden font-semibold text-warning md:table-cell">{{ run.unknown }}</td>
              <td class="hidden md:table-cell">{{ run.errors }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="expandedRunId === run.id && expandedResults" class="bg-surface-secondary">
              <td :colspan="availColspan">
                <div v-if="expandedLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
                <table v-else-if="expandedResults.length" class="w-full border-collapse text-[0.78rem] md:table-fixed [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-[0.4rem] [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-[0.4rem] [&_td]:overflow-hidden [&_td]:text-ellipsis [&_td]:whitespace-nowrap">
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
                          class="text-gold no-underline hover:underline focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                          @click.stop
                        >
                          {{ truncateUrl(r.url) }}
                        </SafeExternalLink>
                        <span v-else class="text-text-muted">--</span>
                      </td>
                      <td>
                        <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="r.status === 'available' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : r.status === 'unavailable' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(241,196,15,0.15)] text-warning'">{{ r.status }}</span>
                      </td>
                      <td class="max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">{{ r.reason || '--' }}</td>
                      <td>{{ r.httpStatus ?? '--' }}</td>
                      <td>{{ r.agentUsed ? 'Yes' : 'No' }}</td>
                    </tr>
                  </tbody>
                </table>
                <p v-else class="px-8 py-8 text-center font-sans text-text-muted">No results for this run.</p>
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="mt-4 flex items-center justify-center gap-3">
        <button class="btn btn-secondary btn-sm" :disabled="availPage <= 1" @click="prevAvailPage()">Prev</button>
        <span class="text-[0.82rem] text-text-secondary">Page {{ availPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="availRuns.length < 5" @click="nextAvailPage()">Next</button>
      </div>
    </template>

    <hr class="my-6 border-0 border-t border-border-subtle" />

    <!-- Auction Ending Alerts -->
    <h3 class="mb-4 text-base font-semibold text-text-primary">Auction Ending Alerts</h3>
    <p class="mb-4 text-base text-text-secondary">Checks watched auction lots that are ending soon and sends Pushover reminders before bidding closes.</p>
    <div class="mb-4">
      <div class="form-group flex items-center justify-between gap-3">
        <label class="form-label">Enable Automatic Alerts</label>
        <label class="relative inline-block h-[22px] w-[42px]">
          <input
            class="peer sr-only" type="checkbox"
            :checked="settings.AuctionEndingCheckEnabled === 'true'"
            @change="settings.AuctionEndingCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.AuctionEndingCheckStartTime"
          class="form-input w-full max-w-[120px]"
          type="time"
        />
        <span class="form-hint">The first ending-alert check runs at this time each day.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.AuctionEndingCheckInterval"
          class="form-input w-full max-w-[120px]"
          type="number"
          min="60"
          step="60"
        />
        <span class="form-hint">How often to check for lots ending soon after the start time. Default 1440 (daily).</span>
      </div>
      <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Alert Settings' }}
        </button>
        <span v-if="auctionSettingsMsg" class="text-body text-gold md:mr-auto" :class="auctionSettingsError ? 'text-[var(--color-negative)]' : ''">{{ auctionSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="auctionTriggerLoading" @click="triggerManualAuctionCheck()">
          {{ auctionTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="my-6 border-0 border-t border-border-subtle" />
    <h3 class="mb-4 text-base font-semibold text-text-primary">Auction Ending Alert Run History</h3>

    <div v-if="auctionLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
    <div v-else-if="auctionRuns.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No auction ending alert runs recorded yet.</div>
    <template v-else>
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Lots</th>
            <th>Alerts</th>
            <th class="hidden md:table-cell">Status</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in auctionRuns" :key="run.id">
            <tr>
              <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
              <td class="hidden md:table-cell">
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.triggerType === 'manual' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(241,196,15,0.15)] text-warning'">
                  {{ run.triggerType }}
                </span>
              </td>
              <td>{{ run.lotsChecked }}</td>
              <td class="font-semibold text-[var(--color-positive)]">{{ run.alertsSent }}</td>
              <td class="hidden md:table-cell">
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.status === 'error' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : (run.status === 'success' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : 'bg-[rgba(241,196,15,0.15)] text-warning')">
                  {{ run.status }}
                </span>
              </td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="mt-4 flex items-center justify-center gap-3">
        <button class="btn btn-secondary btn-sm" :disabled="auctionPage <= 1" @click="prevAuctionPage()">Prev</button>
        <span class="text-[0.82rem] text-text-secondary">Page {{ auctionPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="auctionRuns.length < 5" @click="nextAuctionPage()">Next</button>
      </div>
    </template>

    <hr class="my-6 border-0 border-t border-border-subtle" />

    <!-- Auction Price Alerts and Bid Reminders -->
    <h3 class="mb-4 text-base font-semibold text-text-primary">Auction Price Alerts and Bid Reminders</h3>
    <p class="mb-4 text-base text-text-secondary">Checks watched auction lots for price thresholds and close-time bid reminders.</p>
    <div class="mb-4">
      <div class="form-group flex items-center justify-between gap-3">
        <label class="form-label">Enable Automatic Checks</label>
        <label class="relative inline-block h-[22px] w-[42px]">
          <input
            class="peer sr-only" type="checkbox"
            :checked="settings.AuctionAlertsCheckEnabled === 'true'"
            @change="settings.AuctionAlertsCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.AuctionAlertsCheckStartTime"
          class="form-input w-full max-w-[120px]"
          type="time"
        />
        <span class="form-hint">The first price-alert and reminder check runs at this time each day.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.AuctionAlertsCheckInterval"
          class="form-input w-full max-w-[120px]"
          type="number"
          min="15"
          step="15"
        />
        <span class="form-hint">How often to re-check price thresholds and bid reminder windows. Default 60 (hourly).</span>
      </div>
      <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Alert and Reminder Settings' }}
        </button>
        <span v-if="alertReminderSettingsMsg" class="text-body text-gold md:mr-auto" :class="alertReminderSettingsError ? 'text-[var(--color-negative)]' : ''">{{ alertReminderSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="alertReminderTriggerLoading" @click="triggerManualAlertReminderCheck()">
          {{ alertReminderTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="my-6 border-0 border-t border-border-subtle" />
    <h3 class="mb-4 text-base font-semibold text-text-primary">Auction Price Alert and Reminder Run History</h3>

    <div v-if="alertReminderLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
    <div v-else-if="alertReminderRuns.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No auction price alert or reminder runs recorded yet.</div>
    <template v-else>
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Lots</th>
            <th>Alerts</th>
            <th>Reminders</th>
            <th class="hidden md:table-cell">Status</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in alertReminderRuns" :key="run.id">
            <tr>
              <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
              <td class="hidden md:table-cell">
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.triggerType === 'manual' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(241,196,15,0.15)] text-warning'">
                  {{ run.triggerType }}
                </span>
              </td>
              <td>{{ run.lotsChecked ?? run.alertsChecked ?? 0 }}</td>
              <td class="font-semibold text-[var(--color-positive)]">{{ run.priceAlertsTriggered ?? run.alertsSent ?? run.alertsTriggered ?? 0 }}</td>
              <td class="font-semibold text-[var(--color-positive)]">{{ run.bidRemindersSent ?? run.remindersSent ?? run.remindersNotified ?? 0 }}</td>
              <td class="hidden md:table-cell">
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.status === 'error' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : (run.status === 'success' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : 'bg-[rgba(241,196,15,0.15)] text-warning')">
                  {{ run.status }}
                </span>
              </td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="mt-4 flex items-center justify-center gap-3">
        <button class="btn btn-secondary btn-sm" :disabled="alertReminderPage <= 1" @click="prevAlertReminderPage()">Prev</button>
        <span class="text-[0.82rem] text-text-secondary">Page {{ alertReminderPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="alertReminderRuns.length < 5" @click="nextAlertReminderPage()">Next</button>
      </div>
    </template>

    <hr class="my-6 border-0 border-t border-border-subtle" />

    <!-- Auction Watch Bid Digest -->
    <h3 class="mb-4 text-base font-semibold text-text-primary">Auction Watch Bid Digest</h3>
    <p class="mb-4 text-base text-text-secondary">Refreshes NumisBids and CNG watched lots, updates current high bids in Auctions, and sends one Pushover digest while lots are active.</p>
    <div class="mb-4">
      <div class="form-group flex items-center justify-between gap-3">
        <label class="form-label">Enable Automatic Digests</label>
        <label class="relative inline-block h-[22px] w-[42px]">
          <input
            class="peer sr-only" type="checkbox"
            :checked="settings.AuctionWatchBidDigestEnabled === 'true'"
            @change="settings.AuctionWatchBidDigestEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.AuctionWatchBidDigestStartTime"
          class="form-input w-full max-w-[120px]"
          type="time"
        />
        <span class="form-hint">The first digest run starts at this time each day.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.AuctionWatchBidDigestInterval"
          class="form-input w-full max-w-[120px]"
          type="number"
          min="60"
          step="60"
        />
        <span class="form-hint">How often to refresh watched lots and send the digest after the start time. Default 1440 (daily).</span>
      </div>
      <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Digest Settings' }}
        </button>
        <span v-if="watchBidDigestSettingsMsg" class="text-body text-gold md:mr-auto" :class="watchBidDigestSettingsError ? 'text-[var(--color-negative)]' : ''">{{ watchBidDigestSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="watchBidDigestTriggerLoading" @click="triggerManualWatchBidDigest()">
          {{ watchBidDigestTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="my-6 border-0 border-t border-border-subtle" />
    <h3 class="mb-4 text-base font-semibold text-text-primary">Auction Watch Bid Digest Run History</h3>

    <div v-if="watchBidDigestLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
    <div v-else-if="watchBidDigestRuns.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No auction watch bid digest runs recorded yet.</div>
    <template v-else>
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Lots</th>
            <th>Digests</th>
            <th class="hidden md:table-cell">Status</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in watchBidDigestRuns" :key="run.id">
            <tr>
              <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
              <td class="hidden md:table-cell">
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.triggerType === 'manual' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(241,196,15,0.15)] text-warning'">
                  {{ run.triggerType }}
                </span>
              </td>
              <td>{{ run.lotsChecked }}</td>
              <td class="font-semibold text-[var(--color-positive)]">{{ run.digestsSent }}</td>
              <td class="hidden md:table-cell">
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.status === 'error' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : (run.status === 'success' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : 'bg-[rgba(241,196,15,0.15)] text-warning')">
                  {{ run.status }}
                </span>
              </td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="mt-4 flex items-center justify-center gap-3">
        <button class="btn btn-secondary btn-sm" :disabled="watchBidDigestPage <= 1" @click="prevWatchBidDigestPage()">Prev</button>
        <span class="text-[0.82rem] text-text-secondary">Page {{ watchBidDigestPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="watchBidDigestRuns.length < 5" @click="nextWatchBidDigestPage()">Next</button>
      </div>
    </template>

    <hr class="my-6 border-0 border-t border-border-subtle" />

    <!-- Collection Valuation -->
    <h3 class="mb-4 text-base font-semibold text-text-primary">Collection Valuation</h3>
    <div class="mb-4">
      <div class="form-group flex items-center justify-between gap-3">
        <label class="form-label">Enable Scheduled Valuation</label>
        <label class="relative inline-block h-[22px] w-[42px]">
          <input
            class="peer sr-only" type="checkbox"
            :checked="settings.ValuationCheckEnabled === 'true'"
            @change="settings.ValuationCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.ValuationCheckStartTime"
          class="form-input w-full max-w-[120px]"
          type="time"
        />
        <span class="form-hint">The valuation cycle starts at this time on scheduled days.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (days)</label>
        <input
          v-model="settings.ValuationCheckIntervalDays"
          class="form-input w-full max-w-[120px]"
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
          class="form-input w-full max-w-[120px]"
          type="number"
          min="1"
          step="10"
        />
        <span class="form-hint">Limit how many coins are valuated per run to control AI costs.</span>
      </div>
      <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Valuation Settings' }}
        </button>
        <span v-if="valSettingsMsg" class="text-body text-gold md:mr-auto" :class="valSettingsError ? 'text-[var(--color-negative)]' : ''">{{ valSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="valTriggerLoading" @click="triggerManualValuation()">
          {{ valTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="my-6 border-0 border-t border-border-subtle" />
    <h3 class="mb-4 text-base font-semibold text-text-primary">Valuation Run History</h3>

    <div v-if="valLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
    <div v-else-if="valRuns.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No valuation runs recorded yet.</div>
    <template v-else>
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Status</th>
            <th>Checked</th>
            <th class="hidden md:table-cell">Updated</th>
            <th class="hidden md:table-cell">Skipped</th>
            <th class="hidden md:table-cell">Errors</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in valRuns" :key="run.id">
            <tr class="cursor-pointer transition-colors hover:bg-surface" :class="{ 'bg-surface': valExpandedRunId === run.id }" @click="toggleValRunDetail(run.id)">
              <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
              <td class="hidden md:table-cell">{{ run.triggerType }}</td>
              <td>
                <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.status === 'running' ? 'bg-[rgba(52,152,219,0.15)] text-[#3498db]' : run.status === 'completed' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : run.status === 'failed' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(243,156,18,0.15)] text-[#f39c12]'">{{ run.status }}</span>
                <span v-if="run.status === 'running' && run.totalCoins > 0" class="ml-[0.35rem] text-label font-medium text-text-secondary">
                  {{ run.coinsChecked + run.coinsSkipped + run.errors }} / {{ run.totalCoins }}
                </span>
                <button v-if="run.status === 'running'" class="ml-[0.4rem] rounded-full border border-[rgba(231,76,60,0.4)] bg-transparent px-[0.4rem] py-[0.1rem] text-[0.65rem] text-[var(--color-negative)] transition-colors hover:bg-[rgba(231,76,60,0.15)]" @click.stop="cancelRun(run.id)">Cancel</button>
              </td>
              <td>{{ run.coinsChecked }}</td>
              <td class="hidden font-semibold text-[var(--color-positive)] md:table-cell">{{ run.coinsUpdated }}</td>
              <td class="hidden font-semibold text-warning md:table-cell">{{ run.coinsSkipped }}</td>
              <td class="hidden font-semibold text-[var(--color-negative)] md:table-cell">{{ run.errors }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="valExpandedRunId === run.id && valExpandedResults" class="bg-surface-secondary">
              <td :colspan="valColspan">
                <div v-if="valExpandedLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
                <table v-else-if="valExpandedResults.length" class="w-full border-collapse text-[0.78rem] md:table-fixed [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-[0.4rem] [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-[0.4rem] [&_td]:overflow-hidden [&_td]:text-ellipsis [&_td]:whitespace-nowrap">
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
                      <td class="font-semibold text-gold">{{ r.estimatedValue > 0 ? `$${r.estimatedValue.toFixed(2)}` : '--' }}</td>
                      <td>
                        <span v-if="r.confidence" class="inline-block rounded-sm px-[0.3rem] py-[0.1rem] text-label font-semibold" :class="r.confidence === 'high' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--confidence-high)]' : r.confidence === 'medium' ? 'bg-[rgba(241,196,15,0.15)] text-[var(--confidence-medium)]' : 'bg-[rgba(231,76,60,0.15)] text-[var(--confidence-low)]'">{{ r.confidence }}</span>
                        <span v-else>--</span>
                      </td>
                      <td>
                        <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="r.status === 'success' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : r.status === 'skipped' ? 'bg-[rgba(149,165,166,0.15)] text-[#95a5a6]' : 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]'">{{ r.status }}</span>
                      </td>
                      <td class="max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">
                        <div v-if="r.changeExplanation" class="mb-[0.35rem] font-medium text-gold">{{ r.changeExplanation }}</div>
                        <div>{{ r.reasoning || r.errorMessage || '--' }}</div>
                      </td>
                    </tr>
                  </tbody>
                </table>
                <p v-else class="px-8 py-8 text-center font-sans text-text-muted">No results for this run.</p>
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="mt-4 flex items-center justify-center gap-3">
        <button class="btn btn-secondary btn-sm" :disabled="valPage <= 1" @click="prevValPage()">Prev</button>
        <span class="text-[0.82rem] text-text-secondary">Page {{ valPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="valRuns.length < 5" @click="nextValPage()">Next</button>
      </div>
    </template>

    <hr class="my-6 border-0 border-t border-border-subtle" />

    <!-- Collection Health Snapshots -->
    <h3 class="mb-4 text-base font-semibold text-text-primary">Collection Health Snapshots</h3>
    <p class="mb-4 text-base text-text-secondary">Captures daily health baselines used by the 30-day collection health trend.</p>
    <div class="mb-4">
      <div class="form-group flex items-center justify-between gap-3">
        <label class="form-label">Enable Daily Snapshots</label>
        <label class="relative inline-block h-[22px] w-[42px]">
          <input
            class="peer sr-only" type="checkbox"
            :checked="settings.CollectionHealthSnapshotsEnabled === 'true'"
            @change="settings.CollectionHealthSnapshotsEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily)</label>
        <input
          v-model="settings.CollectionHealthSnapshotsStartTime"
          class="form-input w-full max-w-[120px]"
          type="time"
        />
        <span class="form-hint">Time of day when collection health baselines are captured for trend calculations.</span>
      </div>
      <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Snapshot Settings' }}
        </button>
        <span v-if="healthSettingsMsg" class="text-body text-gold md:mr-auto" :class="healthSettingsError ? 'text-[var(--color-negative)]' : ''">{{ healthSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="healthTriggerLoading" @click="triggerManualHealthSnapshots()">
          {{ healthTriggerLoading ? 'Running...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="my-6 border-0 border-t border-border-subtle" />

    <!-- Coin of the Day -->
    <h3 class="mb-4 text-base font-semibold text-text-primary">Coin of the Day</h3>
    <p class="mb-4 text-base text-text-secondary">Picks one coin per day from each user's collection and sends an in-app and Pushover notification. Each coin in a user's collection appears once before any coin repeats.</p>
    <div class="mb-4">
      <div class="form-group flex items-center justify-between gap-3">
        <label class="form-label">Enable Daily Feature</label>
        <label class="relative inline-block h-[22px] w-[42px]">
          <input
            class="peer sr-only" type="checkbox"
            :checked="settings.CoinOfDayEnabled === 'true'"
            @change="settings.CoinOfDayEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily)</label>
        <input
          v-model="settings.CoinOfDayStartTime"
          class="form-input w-full max-w-[120px]"
          type="time"
        />
        <span class="form-hint">Time of day when the daily featured coin is picked for each enrolled user.</span>
      </div>
      <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save')">
          {{ settingsSaving ? 'Saving...' : 'Save Coin of the Day Settings' }}
        </button>
        <span v-if="cotdSettingsMsg" class="text-body text-gold md:mr-auto" :class="cotdSettingsError ? 'text-[var(--color-negative)]' : ''">{{ cotdSettingsMsg }}</span>
        <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="cotdTriggerLoading" @click="triggerManualCoinOfDay()">
          {{ cotdTriggerLoading ? 'Running...' : 'Run Now' }}
        </button>
      </div>
    </div>

    <hr class="my-6 border-0 border-t border-border-subtle" />
    <h3 class="mb-4 text-base font-semibold text-text-primary">Coin of the Day Run History</h3>
    <div v-if="cotdLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
    <div v-else-if="cotdRuns.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No Coin of the Day runs recorded yet.</div>
    <template v-else>
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th>Status</th>
            <th>Picked</th>
            <th>Skipped</th>
            <th>Errors</th>
            <th class="hidden md:table-cell">Trigger</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="run in cotdRuns" :key="run.id">
            <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
            <td>{{ run.status }}</td>
            <td>{{ run.picked }}</td>
            <td>{{ run.skipped }}</td>
            <td>{{ run.errors }}</td>
            <td class="hidden md:table-cell">{{ run.triggerType }}</td>
          </tr>
        </tbody>
      </table>

      <div class="mt-4 flex items-center justify-center gap-3">
        <button class="btn btn-secondary btn-sm" :disabled="cotdPage <= 1" @click="prevCoinOfDayPage()">Prev</button>
        <span class="text-[0.82rem] text-text-secondary">Page {{ cotdPage }}</span>
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
