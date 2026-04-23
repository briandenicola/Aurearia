<template>
  <PullToRefresh :on-refresh="handleRefresh">
  <div class="container">
    <div class="page-header">
      <h1>Settings</h1>
      <div v-if="isPwa" class="settings-menu-wrapper">
        <button class="btn btn-secondary btn-sm settings-menu-btn" @click="settingsMenuOpen = !settingsMenuOpen">
          <component :is="tabIcons[activeTab]" :size="16" />
          {{ tabs.find(t => t.id === activeTab)?.label }}
          <Menu :size="16" />
        </button>
        <Transition name="fade">
          <div v-if="settingsMenuOpen" class="settings-dropdown">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              class="settings-dropdown-item"
              :class="{ active: activeTab === tab.id }"
              @click="selectTab(tab.id); settingsMenuOpen = false"
            >
              <component :is="tabIcons[tab.id]" :size="16" />
              {{ tab.label }}
            </button>
          </div>
        </Transition>
      </div>
    </div>

    <div class="settings-layout">
      <!-- Tab Nav (desktop only) -->
      <div v-if="!isPwa" class="tab-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{ active: activeTab === tab.id }"
          @click="selectTab(tab.id)"
        ><component :is="tabIcons[tab.id]" :size="16" /> {{ tab.label }}</button>
      </div>

      <!-- Account Tab -->
      <section v-if="activeTab === 'account'" class="settings-section card">
        <h2>Account</h2>

        <!-- Avatar -->
        <div class="setting-item avatar-section">
          <div class="avatar-preview">
            <img :src="avatarUrl" alt="Avatar" class="avatar-img" />
          </div>
          <div class="avatar-actions">
            <label class="btn btn-secondary btn-sm">
              Upload Avatar
              <input type="file" accept="image/*" hidden @change="handleAvatarUpload" />
            </label>
            <button v-if="auth.user?.avatarPath" class="btn btn-danger btn-sm" @click="handleAvatarDelete">Remove</button>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Username</span>
            <span class="setting-value">{{ auth.user?.username }}</span>
          </div>
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Role</span>
            <span class="setting-value badge" :class="`badge-${auth.user?.role === 'admin' ? 'roman' : 'modern'}`">
              {{ auth.user?.role }}
            </span>
          </div>
        </div>

        <!-- Profile / Social Settings -->
        <h3>Profile</h3>
        <div class="form-group">
          <label class="form-label">Email</label>
          <input v-model="profileEmail" type="email" class="form-input" placeholder="you@example.com" />
        </div>
        <div class="form-group">
          <label class="form-label">Bio</label>
          <input v-model="profileBio" class="form-input" placeholder="Tell collectors about yourself..." maxlength="200" />
        </div>
        <div class="form-group">
          <label class="form-label">ZIP Code</label>
          <input v-model="profileZipCode" class="form-input" placeholder="e.g. 90210" maxlength="10" />
          <span class="setting-desc" style="font-size: 0.8rem; margin-top: 0.25rem; display: block">Used by the Agent to find nearby coin shows and dealers</span>
        </div>

        <h3>NumisBids Integration</h3>
        <p class="setting-desc" style="margin-bottom: 0.75rem">
          Connect your NumisBids account to sync your watchlist and track auction lots.
        </p>
        <div class="form-group">
          <label class="form-label">NumisBids Username</label>
          <input v-model="nbUsername" class="form-input" placeholder="Your NumisBids username" autocomplete="off" />
        </div>
        <div class="form-group">
          <label class="form-label">NumisBids Password</label>
          <input v-model="nbPassword" type="password" class="form-input" placeholder="Your NumisBids password" autocomplete="new-password" />
          <span class="setting-desc" style="font-size: 0.8rem; margin-top: 0.25rem; display: block">Stored securely on the server. Used only for watchlist sync.</span>
        </div>
        <div v-if="nbValidating" class="nb-status validating">
          Validating NumisBids credentials...
        </div>
        <div v-else-if="nbValidationError" class="nb-status error">
          {{ nbValidationError }}
        </div>
        <div v-else-if="auth.user?.numisBidsConfigured" class="nb-status connected">
          NumisBids account connected
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Public Collection</span>
            <span class="setting-desc">Allow other users to follow you and view your coins</span>
          </div>
          <label class="toggle">
            <input type="checkbox" :checked="profilePublic" @change="onPublicToggle" />
            <span class="toggle-slider"></span>
          </label>
        </div>
        <button class="btn btn-primary btn-sm" @click="handleSaveProfile" :disabled="profileSaving || nbValidating" style="margin-top: 0.5rem">
          {{ nbValidating ? 'Validating...' : profileSaving ? 'Saving...' : 'Save Profile' }}
        </button>
        <p v-if="profileMsg" class="msg" :class="{ error: profileError }" style="margin-top: 0.5rem">{{ profileMsg }}</p>

    <!-- Privacy Warning Modal -->
    <Teleport to="body">
      <div v-if="showPrivacyWarning" class="modal-overlay" @click.self="cancelGoPrivate">
        <div class="modal-content card" style="max-width: 440px;">
          <div class="modal-header">
            <h2 style="display: flex; align-items: center; gap: 0.5rem; margin: 0; font-size: 1rem;">
              ⚠️ Make Collection Private?
            </h2>
          </div>
          <div class="modal-body" style="padding: 1.25rem;">
            <p style="color: var(--text-secondary); line-height: 1.5; margin: 0 0 0.75rem;">
              Setting your profile to private will <strong style="color: var(--text-primary);">permanently remove all your followers</strong>.
              They will need to send new follow requests if you make your profile public again.
            </p>
            <p style="color: var(--text-secondary); line-height: 1.5; margin: 0 0 1rem;">
              You will also be hidden from user search results.
            </p>
            <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
              <button class="btn btn-secondary btn-sm" @click="cancelGoPrivate">Cancel</button>
              <button class="btn btn-danger btn-sm" @click="confirmGoPrivate">Make Private</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

        <h3>Change Password</h3>
        <form class="password-form" @submit.prevent="handleChangePassword">
          <div class="form-group">
            <label class="form-label">Current Password</label>
            <input v-model="currentPassword" type="password" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">New Password</label>
            <input v-model="newPassword" type="password" class="form-input" required minlength="6" />
          </div>
          <div class="form-group">
            <label class="form-label">Confirm New Password</label>
            <input v-model="confirmPassword" type="password" class="form-input" required />
          </div>
          <p v-if="passwordMsg" class="msg" :class="{ error: passwordError }">{{ passwordMsg }}</p>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="passwordLoading">
            {{ passwordLoading ? 'Changing...' : 'Change Password' }}
          </button>
        </form>

        <template v-if="supportsWebAuthn">
          <h3>Biometric Login</h3>
          <p class="setting-desc" style="margin-bottom: 0.75rem">
            Register Face ID, Touch ID, or fingerprint for quick sign-in on this device.
          </p>

          <button
            class="btn btn-primary btn-sm"
            :disabled="registeringCredential"
            @click="handleRegisterCredential"
          >
            {{ registeringCredential ? 'Registering...' : '🔐 Register Biometric' }}
          </button>
          <p v-if="credentialMsg" class="msg" :class="{ error: credentialError }" style="margin-top: 0.5rem">{{ credentialMsg }}</p>

          <div v-if="webauthnCredentials.length" class="apikey-list">
            <div v-for="cred in webauthnCredentials" :key="cred.id" class="apikey-item">
              <div class="apikey-item-info">
                <span class="apikey-item-name">{{ cred.name }}</span>
                <span class="apikey-item-meta">Registered {{ formatDate(cred.createdAt) }}</span>
              </div>
              <button class="btn btn-danger btn-sm" @click="handleDeleteCredential(cred.id)">Remove</button>
            </div>
          </div>
          <p v-else-if="!registeringCredential" class="setting-desc" style="margin-top: 0.5rem">No biometric credentials registered.</p>
        </template>
      </section>

      <!-- Blocked Users Tab -->
      <!-- Appearance Tab -->
      <section v-if="activeTab === 'appearance'" class="settings-section card">
        <h2>Appearance</h2>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Theme</span>
            <span class="setting-desc">Choose your preferred color scheme</span>
          </div>
          <div class="theme-toggle">
            <button
              class="theme-btn"
              :class="{ active: theme === 'dark' }"
              @click="setTheme('dark')"
            >Dark</button>
            <button
              class="theme-btn"
              :class="{ active: theme === 'light' }"
              @click="setTheme('light')"
            >Light</button>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Timezone</span>
            <span class="setting-desc">Used for date display</span>
          </div>
          <select v-model="timezone" class="form-select tz-select" @change="saveTimezone">
            <option v-for="tz in timezones" :key="tz" :value="tz">{{ tz }}</option>
          </select>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Default View</span>
            <span class="setting-desc">Preferred collection view on mobile / PWA</span>
          </div>
          <div class="theme-toggle">
            <button
              class="theme-btn"
              :class="{ active: defaultView === 'swipe' }"
              @click="setDefaultView('swipe')"
            >Swipe</button>
            <button
              class="theme-btn"
              :class="{ active: defaultView === 'grid' }"
              @click="setDefaultView('grid')"
            >Grid</button>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Default Sort</span>
            <span class="setting-desc">How coins are sorted by default</span>
          </div>
          <select v-model="defaultSort" class="form-select sort-select" @change="saveDefaultSort">
            <option value="updated_at_desc">Last Updated</option>
            <option value="created_at_desc">Newest First</option>
            <option value="created_at_asc">Oldest First</option>
            <option value="current_value_desc">Price: High → Low</option>
            <option value="current_value_asc">Price: Low → High</option>
          </select>
        </div>
      </section>

      <!-- Data Tab -->
      <section v-if="activeTab === 'data'" class="settings-section card">
        <h2>Data Management</h2>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Export Collection</span>
            <span class="setting-desc">Download your collection data and photos as a zip archive</span>
          </div>
          <button class="btn btn-secondary btn-sm" :disabled="exporting" @click="handleExport">
            {{ exporting ? 'Exporting...' : 'Export ZIP' }}
          </button>
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">PDF Catalog</span>
            <span class="setting-desc">Generate a styled PDF catalog with photos, grades, and valuations</span>
          </div>
          <button class="btn btn-secondary btn-sm" :disabled="exportingPdf" @click="handleExportPDF">
            {{ exportingPdf ? 'Generating...' : 'Export PDF' }}
          </button>
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Import Collection</span>
            <span class="setting-desc">Import coins from a JSON file</span>
          </div>
          <label class="btn btn-secondary btn-sm import-btn">
            📤 Import
            <input type="file" accept=".json" hidden @change="handleImport" />
          </label>
        </div>
        <p v-if="dataMsg" class="msg" :class="{ error: dataError }">{{ dataMsg }}</p>

        <h3>API Keys</h3>
        <p class="setting-desc" style="margin-bottom: 1rem">
          Generate API keys to access your collection from external tools and scripts. Use the <code>X-API-Key</code> header to authenticate.
        </p>

        <div class="apikey-generate">
          <input
            v-model="apiKeyName"
            type="text"
            class="form-input"
            placeholder="Key name (e.g. My Script)"
            :disabled="generatingKey"
          />
          <button
            class="btn btn-primary btn-sm"
            :disabled="!apiKeyName.trim() || generatingKey"
            @click="handleGenerateKey"
          >
            {{ generatingKey ? 'Generating...' : '🔑 Generate Key' }}
          </button>
        </div>

        <div v-if="newlyGeneratedKey" class="apikey-reveal">
          <p class="apikey-reveal-warning">
            ⚠️ Copy this key now — it will not be shown again.
          </p>
          <div class="apikey-reveal-box">
            <code class="apikey-reveal-value">{{ newlyGeneratedKey }}</code>
            <button class="btn btn-secondary btn-sm" @click="copyKey">
              {{ keyCopied ? '✓ Copied' : '📋 Copy' }}
            </button>
          </div>
        </div>

        <p v-if="apiKeyMsg" class="msg" :class="{ error: apiKeyError }">{{ apiKeyMsg }}</p>

        <div v-if="apiKeys.length" class="apikey-list">
          <div
            v-for="key in apiKeys"
            :key="key.id"
            class="apikey-item"
            :class="{ revoked: key.revokedAt }"
          >
            <div class="apikey-item-info">
              <span class="apikey-item-name">{{ key.name }}</span>
              <span class="apikey-item-meta">
                ...{{ key.keyPrefix }}
                · Created {{ formatDate(key.createdAt) }}
                <template v-if="key.lastUsedAt"> · Last used {{ formatDate(key.lastUsedAt) }}</template>
              </span>
            </div>
            <span v-if="key.revokedAt" class="apikey-item-badge revoked-badge">Revoked</span>
            <button
              v-else
              class="btn btn-danger btn-sm"
              @click="handleRevokeKey(key.id)"
            >
              Revoke
            </button>
          </div>
        </div>
        <p v-else-if="!generatingKey" class="setting-desc" style="margin-top: 0.5rem">No API keys yet.</p>

        <h3 style="margin-top: 2rem">Tags</h3>
        <p class="setting-desc">Create custom tags to organize and filter your coins.</p>

        <div class="tag-create-form">
          <input
            v-model="newTagName"
            type="text"
            class="form-input"
            placeholder="New tag name..."
            maxlength="50"
            @keydown.enter="handleCreateTag"
          />
          <div class="tag-color-picker">
            <button
              v-for="c in TAG_COLORS"
              :key="c"
              class="color-swatch"
              :class="{ active: newTagColor === c }"
              :style="{ backgroundColor: c }"
              @click="newTagColor = c"
            ></button>
          </div>
          <button class="btn btn-primary btn-sm" @click="handleCreateTag" :disabled="!newTagName.trim()">Create Tag</button>
        </div>
        <p v-if="tagError" class="tag-error">{{ tagError }}</p>

        <div v-if="tagList.length" class="tag-list">
          <div v-for="tag in tagList" :key="tag.id" class="tag-list-item">
            <template v-if="editingTag?.id === tag.id">
              <input v-model="editTagName" class="form-input tag-edit-input" maxlength="50" @keydown.enter="handleSaveTag" />
              <div class="tag-color-picker">
                <button
                  v-for="c in TAG_COLORS"
                  :key="c"
                  class="color-swatch sm"
                  :class="{ active: editTagColor === c }"
                  :style="{ backgroundColor: c }"
                  @click="editTagColor = c"
                ></button>
              </div>
              <button class="btn btn-primary btn-sm" @click="handleSaveTag">Save</button>
              <button class="btn btn-secondary btn-sm" @click="editingTag = null">Cancel</button>
            </template>
            <template v-else>
              <span class="tag-preview" :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }">{{ tag.name }}</span>
              <div class="tag-actions">
                <button class="btn btn-secondary btn-sm" @click="startEditTag(tag)">Edit</button>
                <button class="btn btn-danger btn-sm" @click="handleDeleteTag(tag)">Delete</button>
              </div>
            </template>
          </div>
        </div>
        <p v-else class="empty-tags">No tags created yet. Create your first tag above.</p>
      </section>

      <!-- Tools Tab -->
      <section v-if="activeTab === 'tools'" class="settings-section card">
        <h2>Image Processor</h2>
        <p class="setting-desc" style="margin-bottom: 1rem">
          Remove backgrounds and crop coin images for your collection.
        </p>
        <ImageProcessor @saved="handleProcessSaved" />

        <h3 style="margin-top: 2rem">Blocked Users</h3>
        <p class="setting-desc" style="margin-bottom: 0.75rem">
          Blocked users cannot send you follow requests or view your collection.
        </p>
        <div v-if="blockedUsers.length" class="apikey-list">
          <div v-for="user in blockedUsers" :key="user.id" class="apikey-item">
            <div class="apikey-item-info" style="display: flex; align-items: center; gap: 0.5rem;">
              <img
                :src="user.avatarPath ? `/uploads/${user.avatarPath}` : '/coin-logo.jpg'"
                :alt="user.username"
                style="width: 28px; height: 28px; border-radius: 50%; object-fit: cover; border: 1px solid var(--border-subtle);"
              />
              <span class="apikey-item-name">{{ user.username }}</span>
            </div>
            <button class="btn btn-secondary btn-sm" :disabled="blockedLoading" @click="handleUnblock(user)">Unblock</button>
          </div>
        </div>
        <p v-else class="setting-desc" style="margin-top: 0.5rem">No blocked users.</p>
      </section>

      <!-- Conversations Tab -->
      <section v-if="activeTab === 'conversations'" class="settings-section card">
        <h2>Saved Conversations</h2>
        <p class="setting-desc" style="margin-bottom: 1rem">
          Your saved AI coin search conversations. Open one to continue the search or review results.
        </p>

        <div v-if="conversationsLoading" class="loading-inline">Loading...</div>

        <div v-else-if="conversations.length" class="apikey-list">
          <div v-for="conv in conversations" :key="conv.id" class="apikey-item">
            <div class="apikey-item-info" style="cursor: pointer" @click="openConversation(conv.id)">
              <span class="apikey-item-name">{{ conv.title }}</span>
              <span class="apikey-item-meta">{{ formatDate(conv.updatedAt) }}</span>
            </div>
            <div class="conv-actions">
              <button class="btn btn-secondary btn-sm" @click="openConversation(conv.id)">Open</button>
              <button class="btn btn-danger btn-sm" @click="handleDeleteConversation(conv.id)">Delete</button>
            </div>
          </div>
        </div>
        <p v-else class="setting-desc" style="margin-top: 0.5rem">No saved conversations yet. Use the Save button in the coin search chat to save a conversation.</p>
      </section>

      <!-- Help Tab -->
      <section v-if="activeTab === 'help'" class="settings-section card help-section">
        <h2>Beginner's Guide to Ancient Coins</h2>
        <p class="setting-desc" style="margin-bottom: 1.5rem">
          Everything you need to know to start collecting ancient coins with confidence.
        </p>

        <details class="help-accordion" open>
          <summary class="help-summary">Types of Ancient Coins</summary>
          <div class="help-content">
            <p>Ancient coins span thousands of years across many civilizations. Here are the major categories:</p>

            <h4>Greek (c. 600 BC – 31 BC)</h4>
            <p>Among the earliest coins ever minted. Greek coins are known for their artistic beauty and variety. Key types include:</p>
            <ul>
              <li><strong>Stater</strong> — A standard gold or silver denomination used across Greek city-states</li>
              <li><strong>Drachm / Tetradrachm</strong> — Silver coins; tetradrachms (4 drachms) are especially prized by collectors for their large size and detailed artwork</li>
              <li><strong>Obol</strong> — Small silver coin, originally 1/6 of a drachm</li>
              <li><strong>Hemidrachm</strong> — Half a drachm, commonly found in smaller denominations</li>
            </ul>
            <p class="help-tip">💡 <strong>Popular starting points:</strong> Athenian owl tetradrachms, Alexander the Great drachms, and Ptolemaic bronzes are widely available and recognizable.</p>

            <h4>Roman Republic (c. 280 – 27 BC)</h4>
            <p>Coins of the Roman Republic feature anonymous designs early on, evolving to depict political figures and events.</p>
            <ul>
              <li><strong>Denarius</strong> — The workhorse silver coin of Rome, roughly the size of a dime</li>
              <li><strong>As / Semis</strong> — Bronze denominations for everyday transactions</li>
              <li><strong>Victoriatus</strong> — Early silver coin with a Victory motif</li>
            </ul>

            <h4>Roman Imperial (27 BC – 476 AD)</h4>
            <p>The most popular category for collectors. Imperial coins always feature the emperor's portrait on the obverse.</p>
            <ul>
              <li><strong>Aureus</strong> — Gold coin (~7.7g), the most valuable denomination</li>
              <li><strong>Denarius</strong> — Silver coin, the standard unit of account</li>
              <li><strong>Sestertius</strong> — Large bronze coin, prized for detailed reverse designs</li>
              <li><strong>Antoninianus</strong> — Silver/billon coin introduced by Caracalla (215 AD), gradually debased</li>
              <li><strong>Follis</strong> — Late Roman bronze, common in the 4th century</li>
              <li><strong>Solidus</strong> — Gold coin introduced by Constantine I, replaced the aureus</li>
            </ul>
            <p class="help-tip">💡 <strong>Best for beginners:</strong> Late Roman bronzes (4th century) are affordable, plentiful, and come in recognizable emperor portraits. Start with Constantine I, Constantius II, or Valentinian I.</p>

            <h4>Byzantine (330 – 1453 AD)</h4>
            <p>Continuation of the Eastern Roman Empire. Notable for their distinctive cup-shaped coins (scyphate) and religious imagery.</p>
            <ul>
              <li><strong>Solidus / Histamenon</strong> — Gold standard coin</li>
              <li><strong>Follis</strong> — Bronze coin, often featuring Christ or the emperor with a cross</li>
              <li><strong>Hyperpyron</strong> — Later gold denomination, often cup-shaped</li>
            </ul>

            <h4>Celtic, Parthian & Other</h4>
            <p>Don't overlook less-collected areas — Celtic coins from Gaul and Britain, Parthian drachms, Sasanian silver, Judean bronzes, and Nabataean coins all offer fascinating collecting opportunities, often at lower price points.</p>
          </div>
        </details>

        <details class="help-accordion">
          <summary class="help-summary">Understanding Coin Grading</summary>
          <div class="help-content">
            <p>Grading describes a coin's physical condition and is the single biggest factor in determining value. Grades are standardized across the hobby:</p>

            <table class="help-table">
              <thead>
                <tr><th>Grade</th><th>Abbreviation</th><th>Description</th></tr>
              </thead>
              <tbody>
                <tr><td>Poor</td><td>P / PO</td><td>Barely identifiable, heavy wear. Type and ruler may be unrecognizable.</td></tr>
                <tr><td>Fair</td><td>FR</td><td>Heavily worn but type is identifiable. Legend mostly gone.</td></tr>
                <tr><td>About Good</td><td>AG</td><td>Very heavily worn; outline visible, some legend readable.</td></tr>
                <tr><td>Good</td><td>G</td><td>Major design elements visible but flat. Legends partially readable.</td></tr>
                <tr><td>Very Good</td><td>VG</td><td>Main features clear with moderate wear. About half the legend readable.</td></tr>
                <tr><td>Fine</td><td>F</td><td>Moderate wear on high points. Most legend and design details visible.</td></tr>
                <tr><td>Very Fine</td><td>VF</td><td>Light wear on high points only. Full legend readable. Most detail sharp.</td></tr>
                <tr><td>Extremely Fine</td><td>EF / XF</td><td>Slight wear on highest points. Nearly full detail. Attractive coin.</td></tr>
                <tr><td>Almost Uncirculated</td><td>AU</td><td>Trace wear only. Full luster in protected areas.</td></tr>
                <tr><td>Mint State</td><td>MS</td><td>No wear at all. Varies from MS-60 (bag marks) to MS-70 (perfect).</td></tr>
                <tr><td>Superb</td><td>FDC</td><td>"Fleur de Coin" — perfect or near-perfect mint state. Extremely rare for ancients.</td></tr>
              </tbody>
            </table>

            <p class="help-tip">💡 <strong>For ancient coins</strong>, VF is considered a very respectable grade. Many ancient coins in EF or above command significant premiums. Don't expect modern-coin perfection — 2,000-year-old coins have character!</p>

            <h4>Grading Services</h4>
            <p>Professional grading services (NGC Ancients, PCGS) authenticate and grade coins, sealing them in tamper-proof holders ("slabs"). This adds cost but provides confidence in authenticity and grade. NGC Ancients is the most widely used service for ancient coins.</p>
            <p>Not all coins need to be slabbed. Many experienced collectors prefer "raw" (unslabbed) coins for their tactile appeal and lower total cost.</p>
          </div>
        </details>

        <details class="help-accordion">
          <summary class="help-summary">Reference Catalogs (RIC, RPC, Sear & More)</summary>
          <div class="help-content">
            <p>Reference catalogs assign standard numbers to coin types, making identification and communication precise. When you see a number like "RIC 207," it refers to a specific coin type in a specific volume.</p>

            <table class="help-table">
              <thead>
                <tr><th>Catalog</th><th>Full Name</th><th>Coverage</th></tr>
              </thead>
              <tbody>
                <tr><td><strong>RIC</strong></td><td>Roman Imperial Coinage</td><td>The definitive reference for Roman Imperial coins (27 BC – 491 AD). 10 volumes covering every emperor. Most commonly cited catalog for Roman coins.</td></tr>
                <tr><td><strong>RRC</strong></td><td>Roman Republican Coinage</td><td>Crawford's catalog of Roman Republic coins. Referenced as "Cr." or "Crawford."</td></tr>
                <tr><td><strong>RPC</strong></td><td>Roman Provincial Coinage</td><td>Covers coins minted in Roman provinces (not Rome itself). Many volumes, also available online at rpc.ashmus.ox.ac.uk.</td></tr>
                <tr><td><strong>Sear</strong></td><td>Roman Coins and Their Values</td><td>David Sear's accessible guide, good for identification and price estimates. 5 volumes for Imperial, separate books for Greek and Byzantine.</td></tr>
                <tr><td><strong>BMC</strong></td><td>British Museum Catalogue</td><td>Scholarly reference published by the British Museum. Covers Greek, Roman, and other series.</td></tr>
                <tr><td><strong>SNG</strong></td><td>Sylloge Nummorum Graecorum</td><td>Multi-volume catalog of Greek coins from various museum collections worldwide.</td></tr>
                <tr><td><strong>DOC</strong></td><td>Dumbarton Oaks Collection</td><td>The primary reference for Byzantine coins.</td></tr>
                <tr><td><strong>RSC</strong></td><td>Roman Silver Coins</td><td>Seaby's reference focused on silver denominations.</td></tr>
              </tbody>
            </table>

            <p class="help-tip">💡 <strong>Getting started:</strong> You don't need to buy these expensive books right away. Websites like <a href="https://www.wildwinds.com" target="_blank" rel="noopener">WildWinds</a>, <a href="https://www.acsearch.info" target="_blank" rel="noopener">ACSearch</a>, and <a href="https://en.numista.com" target="_blank" rel="noopener">Numista</a> provide free searchable databases with images and catalog references.</p>

            <h4>How to Read a Catalog Reference</h4>
            <p><strong>Example: "RIC VII 162"</strong></p>
            <ul>
              <li><strong>RIC</strong> — The catalog (Roman Imperial Coinage)</li>
              <li><strong>VII</strong> — Volume number (Constantine I era)</li>
              <li><strong>162</strong> — The specific type number in that volume</li>
            </ul>
            <p>Always include the volume number when citing RIC, since type numbers restart in each volume.</p>
          </div>
        </details>

        <details class="help-accordion">
          <summary class="help-summary">Reading Inscriptions (Legends)</summary>
          <div class="help-content">
            <p>Roman coin inscriptions follow conventions that, once understood, make identification much easier. Legends are typically in Latin and read clockwise starting from the lower left.</p>

            <h4>Common Obverse Elements</h4>
            <table class="help-table">
              <thead>
                <tr><th>Abbreviation</th><th>Meaning</th><th>Example</th></tr>
              </thead>
              <tbody>
                <tr><td>IMP</td><td>Imperator (Commander/Emperor)</td><td>IMP CAESAR = Emperor Caesar</td></tr>
                <tr><td>CAES / CAESAR</td><td>Caesar (title)</td><td>—</td></tr>
                <tr><td>AVG</td><td>Augustus (title of honor)</td><td>IMP CAES TRAIANVS AVG</td></tr>
                <tr><td>P M / PONT MAX</td><td>Pontifex Maximus (chief priest)</td><td>—</td></tr>
                <tr><td>TR P / TRIB POT</td><td>Tribunicia Potestas (tribunician power) — renewed annually, useful for dating</td><td>TR P XV = 15th year of tribunician power</td></tr>
                <tr><td>COS</td><td>Consul (with number = which consulship)</td><td>COS III = third consulship</td></tr>
                <tr><td>P P</td><td>Pater Patriae (Father of the Fatherland)</td><td>—</td></tr>
                <tr><td>D N</td><td>Dominus Noster (Our Lord) — late Roman</td><td>D N CONSTANTIVS P F AVG</td></tr>
                <tr><td>P F</td><td>Pius Felix (Dutiful and Happy)</td><td>—</td></tr>
              </tbody>
            </table>

            <h4>Common Reverse Inscriptions</h4>
            <table class="help-table">
              <thead>
                <tr><th>Inscription</th><th>Meaning</th></tr>
              </thead>
              <tbody>
                <tr><td>S C (on bronzes)</td><td>Senatus Consulto — "by decree of the Senate"</td></tr>
                <tr><td>VICTORIA AVG</td><td>Victory of the Emperor</td></tr>
                <tr><td>PAX AVGVSTI</td><td>Peace of the Emperor</td></tr>
                <tr><td>CONCORDIA</td><td>Harmony/Agreement</td></tr>
                <tr><td>PROVIDENTIA AVG</td><td>Foresight of the Emperor</td></tr>
                <tr><td>GLORIA EXERCITVS</td><td>Glory of the Army — common on 4th century bronzes</td></tr>
                <tr><td>VOT V / X / XX</td><td>Votive offerings for 5/10/20 years of rule</td></tr>
                <tr><td>FEL TEMP REPARATIO</td><td>Restoration of Happy Times — common Constantine-era type</td></tr>
              </tbody>
            </table>

            <h4>Latin Reading Tips</h4>
            <ul>
              <li><strong>V = U</strong> — Latin had no letter U; AVGVSTVS = Augustus</li>
              <li><strong>I = J</strong> — No letter J; IVLIA = Julia</li>
              <li><strong>Ligatures</strong> — Letters sometimes share strokes (AE merged into Æ)</li>
              <li><strong>Retrograde</strong> — Very early coins sometimes have reversed letters</li>
              <li><strong>Worn legends</strong> — Use known text patterns and reference photos to fill gaps</li>
            </ul>

            <p class="help-tip">💡 <strong>Pro tip:</strong> The obverse legend usually tells you WHO (the emperor) and WHEN (via titles like TR P and COS numbers). The reverse tells you the MESSAGE (propaganda, virtues, military victories).</p>
          </div>
        </details>

        <details class="help-accordion">
          <summary class="help-summary">Buying Your First Coins</summary>
          <div class="help-content">
            <h4>Where to Buy</h4>
            <ul>
              <li><strong>Established auction houses</strong> — CNG (Classical Numismatic Group), Heritage Auctions, Roma Numismatics, Nomos, Leu Numismatik. These guarantee authenticity.</li>
              <li><strong>Reputable dealers</strong> — Look for dealers who are members of professional organizations (ANA, IAPN, PNG).</li>
              <li><strong>VCoins / MA-Shops</strong> — Online marketplaces with vetted dealer storefronts. Return policies provide safety.</li>
              <li><strong>Coin shows</strong> — Great for handling coins and meeting dealers. Many shows have ancient coin sections.</li>
            </ul>

            <h4>What to Look for in a Coin</h4>
            <ul>
              <li><strong>Centering</strong> — A well-centered strike shows the full design. Off-center coins are less desirable (unless extremely rare).</li>
              <li><strong>Portrait quality</strong> — Sharp, detailed portraits are more appealing and valuable.</li>
              <li><strong>Full legends</strong> — Readable legends are important for identification and value.</li>
              <li><strong>Patina</strong> — Natural patina (surface coloring from age) is desirable. Don't clean ancient coins.</li>
              <li><strong>Eye appeal</strong> — Subjective but important. A coin should look attractive for its grade.</li>
              <li><strong>Provenance</strong> — Documented ownership history ("ex-collection") adds value and confirms legitimacy.</li>
            </ul>

            <h4>Budget Guidelines</h4>
            <table class="help-table">
              <thead>
                <tr><th>Budget</th><th>What You Can Expect</th></tr>
              </thead>
              <tbody>
                <tr><td>$10–$50</td><td>Late Roman bronzes (VG–F), Greek bronzes, uncleaned lots</td></tr>
                <tr><td>$50–$200</td><td>Better Roman bronzes (VF), common denarii (F–VF), interesting Greek bronzes</td></tr>
                <tr><td>$200–$500</td><td>Nice denarii (VF–EF), provincial silver, sestertii (F–VF)</td></tr>
                <tr><td>$500–$2,000</td><td>Choice denarii (EF+), Greek silver drachms, nice sestertii, gold fraction</td></tr>
                <tr><td>$2,000+</td><td>Gold aurei/solidi, Greek tetradrachms, rare types, top-grade coins</td></tr>
              </tbody>
            </table>
          </div>
        </details>

        <details class="help-accordion">
          <summary class="help-summary">⚠️ What to Watch Out For</summary>
          <div class="help-content">
            <h4>Forgeries & Fakes</h4>
            <p>Forgery has been a problem in numismatics for centuries. Modern fakes, especially from certain overseas workshops, can be very convincing. Here's how to protect yourself:</p>
            <ul>
              <li><strong>Buy from reputable sources</strong> — This is the #1 rule. Established auction houses and dealers guarantee authenticity. If a coin is fake, you get your money back.</li>
              <li><strong>Beware of "too good to be true"</strong> — A rare coin at a bargain price is almost certainly fake. If an eBay listing offers an EF Julius Caesar denarius for $50, walk away.</li>
              <li><strong>Check weight and diameter</strong> — Fakes often have incorrect weight or dimensions. Invest in a precision scale (0.01g) and calipers.</li>
              <li><strong>Study die characteristics</strong> — Genuine coins were struck from hand-engraved dies, creating subtle variations. Cast fakes often look "mushy" or have casting bubbles on the edge.</li>
              <li><strong>Look for casting seams</strong> — Genuine ancient coins were struck (hammered between two dies). Cast fakes may show a seam around the edge where the mold halves met.</li>
              <li><strong>Use resources</strong> — <a href="https://www.forumancientcoins.com/fakes/" target="_blank" rel="noopener">Forum Ancient Coins Fake Reports</a> and the Fake Ancient Coins group on Facebook are excellent resources.</li>
              <li><strong>Get it slabbed</strong> — For expensive purchases, consider NGC or PCGS authentication.</li>
            </ul>

            <h4>Cleaning</h4>
            <p class="help-warning">🚫 <strong>Never clean an ancient coin.</strong> Cleaning almost always reduces a coin's value, often dramatically. The natural patina that develops over centuries is considered part of the coin's character and beauty.</p>
            <ul>
              <li><strong>No brushing, polishing, or chemical dips</strong> — These remove patina permanently</li>
              <li><strong>Avoid "coin cleaning" products</strong> — They are designed for modern coins and will damage ancients</li>
              <li><strong>"Tooled" coins</strong> — Some fakes or cleaned coins have been re-engraved with a tool to restore detail. Look for unnatural scratch patterns in the fields.</li>
              <li><strong>Exception</strong> — Professional conservators can stabilize bronze disease or remove harmful deposits, but this is specialized work, not cleaning.</li>
            </ul>

            <h4>Common Scams & Pitfalls</h4>
            <ul>
              <li><strong>Uncleaned coin lots</strong> — Bulk lots of "uncleaned Roman coins" are a fun gamble for learning, but rarely contain anything valuable. Most will be low-grade Late Roman bronzes. Don't overpay.</li>
              <li><strong>eBay from certain regions</strong> — Be very cautious buying from regions known for producing fakes. Stick to sellers with excellent feedback and return policies.</li>
              <li><strong>"Guaranteed authentic"</strong> — Anyone can write this. A guarantee is only as good as the seller's reputation and return policy.</li>
              <li><strong>Over-grading</strong> — Sellers sometimes describe coins more generously than warranted. Learn to grade yourself by studying reference photos.</li>
              <li><strong>Misattributed coins</strong> — A coin labeled as one emperor may actually be another. Always verify the identification yourself using references.</li>
              <li><strong>Legal concerns</strong> — Be aware of cultural heritage laws. Buy coins with documented provenance or from dealers who comply with import/export regulations. Some countries restrict the export of antiquities.</li>
            </ul>

            <h4>Red Flags Summary</h4>
            <table class="help-table">
              <thead>
                <tr><th>🚩 Red Flag</th><th>What It Means</th></tr>
              </thead>
              <tbody>
                <tr><td>Price far below market value</td><td>Likely fake or misrepresented</td></tr>
                <tr><td>No return policy</td><td>Seller isn't confident in authenticity</td></tr>
                <tr><td>Casting bubbles on edge</td><td>Coin was cast (poured), not struck</td></tr>
                <tr><td>Wrong weight or dimensions</td><td>Common sign of modern reproduction</td></tr>
                <tr><td>Unnaturally shiny surface</td><td>Coin has been cleaned or is a modern fake</td></tr>
                <tr><td>Seam around the edge</td><td>Cast fake from a two-part mold</td></tr>
                <tr><td>"Museum quality" on eBay</td><td>Marketing language from unreliable sellers</td></tr>
              </tbody>
            </table>
          </div>
        </details>

        <details class="help-accordion">
          <summary class="help-summary">Helpful Resources</summary>
          <div class="help-content">
            <h4>Online Databases & References</h4>
            <ul>
              <li><a href="https://www.wildwinds.com" target="_blank" rel="noopener"><strong>WildWinds</strong></a> — Free database of ancient coin images organized by ruler and catalog reference</li>
              <li><a href="https://en.numista.com" target="_blank" rel="noopener"><strong>Numista</strong></a> — Collaborative coin catalog with detailed type information (integrated into this app)</li>
              <li><a href="https://www.acsearch.info" target="_blank" rel="noopener"><strong>ACSearch</strong></a> — Auction archives with realized prices — essential for valuation</li>
              <li><a href="https://rpc.ashmus.ox.ac.uk" target="_blank" rel="noopener"><strong>RPC Online</strong></a> — Free searchable database of Roman Provincial coins</li>
              <li><a href="https://www.coinarchives.com" target="_blank" rel="noopener"><strong>CoinArchives</strong></a> — Auction results archive for price research</li>
              <li><a href="https://www.forumancientcoins.com" target="_blank" rel="noopener"><strong>Forum Ancient Coins</strong></a> — Dealer, educational articles, and community discussion forum</li>
            </ul>

            <h4>Communities</h4>
            <ul>
              <li><a href="https://www.reddit.com/r/AncientCoins/" target="_blank" rel="noopener"><strong>r/AncientCoins</strong></a> — Active Reddit community for identification help, purchases, and discussion</li>
              <li><a href="https://www.cointalk.com/forums/ancient-coins.pair7/" target="_blank" rel="noopener"><strong>CoinTalk Ancient Coins</strong></a> — Long-running numismatic forum</li>
              <li><strong>Facebook groups</strong> — "Ancient Coins," "Fake Ancient Coins" (educational), and regional groups</li>
            </ul>

            <h4>Books for Beginners</h4>
            <ul>
              <li><em>Ancient Coin Collecting</em> by Wayne Sayles — The classic introduction to the hobby</li>
              <li><em>Roman Coins and Their Values</em> by David Sear — Identification and pricing guide (5 volumes)</li>
              <li><em>A Dictionary of Ancient Roman Coins</em> by John Melville Jones — Explains all the terminology</li>
              <li><em>Handbook of Greek Coinage</em> by Oliver Hoover — If Greek coins interest you</li>
            </ul>
          </div>
        </details>
      </section>

      <CoinSearchChat
        v-if="showChat"
        :loadConversation="chatConversation"
        @close="showChat = false; chatConversation = null"
        @added="() => {}"
      />
    </div>
  </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import PullToRefresh from '@/components/PullToRefresh.vue'
import {
  changePassword, exportCollection, exportCatalogPDF, importCollection,
  generateApiKey, listApiKeys, revokeApiKey,
  webauthnRegisterBegin, webauthnRegisterFinish,
  webauthnListCredentials, webauthnDeleteCredential,
  listConversations, getConversation, deleteConversation,
  uploadAvatar, deleteAvatar, updateProfile, getMe,
  getBlockedUsers, unblockFollower, validateNumisBidsCredentials,
  getTags, createTag, updateTag as updateTagApi, deleteTag,
} from '@/api/client'
import type { ConversationSummary } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { usePwa } from '@/composables/usePwa'
import type { Coin, Theme, ApiKey, WebAuthnCredentialInfo, Tag } from '@/types'
import CoinSearchChat from '@/components/CoinSearchChat.vue'
import ImageProcessor from '@/components/ImageProcessor.vue'
import { User, Palette, Database, MessageSquare, HelpCircle, Wrench, Menu, ShieldCheck, Tags } from 'lucide-vue-next'

const tabIcons: Record<string, Component> = {
  account: User,
  appearance: Palette,
  data: Database,
  tools: Wrench,
  conversations: MessageSquare,
  help: HelpCircle,
  admin: ShieldCheck,
}

const { showConfirm, showAlert } = useDialog()
const activeTab = ref('account')
const settingsMenuOpen = ref(false)
const { isPwa } = usePwa()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const baseTabs = [
  { id: 'account', label: 'Account' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'data', label: 'Data' },
  { id: 'tools', label: 'Tools' },
  { id: 'conversations', label: 'Conversations' },
  { id: 'help', label: 'Help' },
]
const tabs = computed(() => {
  if (isPwa && auth.isAdmin) {
    return [
      { id: 'account', label: 'Account' },
      { id: 'admin', label: 'Admin' },
      { id: 'appearance', label: 'Appearance' },
      { id: 'data', label: 'Data' },
      { id: 'tools', label: 'Tools' },
      { id: 'conversations', label: 'Conversations' },
      { id: 'help', label: 'Help' },
    ]
  }
  return baseTabs
})

// Set active tab from query param (e.g. /settings?tab=process)
if (route.query.tab && baseTabs.map(t => t.id).concat('admin').includes(route.query.tab as string)) {
  activeTab.value = route.query.tab as string
}

function selectTab(tabId: string) {
  if (tabId === 'admin') {
    router.push('/admin')
    return
  }
  activeTab.value = tabId
}

function handleProcessSaved(savedCoinId: number) {
  router.push(`/edit/${savedCoinId}`)
}

// Blocked users
const blockedUsers = ref<{ id: number; username: string; avatarPath: string }[]>([])
const blockedLoading = ref(false)

async function loadBlockedUsers() {
  try {
    const res = await getBlockedUsers()
    blockedUsers.value = res.data.blocked
  } catch {
    blockedUsers.value = []
  }
}

async function handleUnblock(user: { id: number; username: string; avatarPath: string }) {
  blockedLoading.value = true
  try {
    await unblockFollower(user.id)
    blockedUsers.value = blockedUsers.value.filter(u => u.id !== user.id)
  } catch {
    // ignore
  } finally {
    blockedLoading.value = false
  }
}

// Avatar
const avatarUrl = ref('/coin-logo.jpg')

// Tag management
const tagList = ref<Tag[]>([])
const newTagName = ref('')
const newTagColor = ref('#6b7280')
const editingTag = ref<Tag | null>(null)
const editTagName = ref('')
const editTagColor = ref('')
const tagError = ref('')

const TAG_COLORS = ['#6b7280', '#ef4444', '#f59e0b', '#10b981', '#3b82f6', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#6366f1']

async function loadTags() {
  try {
    const res = await getTags()
    tagList.value = res.data?.tags ?? []
  } catch { tagList.value = [] }
}

async function handleCreateTag() {
  tagError.value = ''
  const name = newTagName.value.trim()
  if (!name) return
  try {
    await createTag({ name, color: newTagColor.value })
    newTagName.value = ''
    newTagColor.value = '#6b7280'
    await loadTags()
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'response' in e) {
      const axiosErr = e as { response?: { data?: { error?: string } } }
      tagError.value = axiosErr.response?.data?.error ?? 'Failed to create tag'
    } else {
      tagError.value = 'Failed to create tag'
    }
  }
}

function startEditTag(tag: Tag) {
  editingTag.value = tag
  editTagName.value = tag.name
  editTagColor.value = tag.color
}

async function handleSaveTag() {
  tagError.value = ''
  if (!editingTag.value) return
  try {
    await updateTagApi(editingTag.value.id, { name: editTagName.value.trim(), color: editTagColor.value })
    editingTag.value = null
    await loadTags()
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'response' in e) {
      const axiosErr = e as { response?: { data?: { error?: string } } }
      tagError.value = axiosErr.response?.data?.error ?? 'Failed to update tag'
    } else {
      tagError.value = 'Failed to update tag'
    }
  }
}

async function handleDeleteTag(tag: Tag) {
  const confirmed = await showConfirm(`Delete tag "${tag.name}"? It will be removed from all coins.`, { title: 'Delete Tag', variant: 'danger' })
  if (!confirmed) return
  try {
    await deleteTag(tag.id)
    await loadTags()
  } catch { /* ignore */ }
}

function updateAvatarUrl() {
  avatarUrl.value = auth.user?.avatarPath ? `/uploads/${auth.user.avatarPath}` : '/coin-logo.jpg'
}
updateAvatarUrl()

async function handleAvatarUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  try {
    const res = await uploadAvatar(file)
    if (auth.user) {
      auth.user.avatarPath = res.data.avatarPath
      localStorage.setItem('user', JSON.stringify(auth.user))
    }
    updateAvatarUrl()
  } catch { /* ignore */ }
}

async function handleAvatarDelete() {
  try {
    await deleteAvatar()
    if (auth.user) {
      auth.user.avatarPath = ''
      localStorage.setItem('user', JSON.stringify(auth.user))
    }
    updateAvatarUrl()
  } catch { /* ignore */ }
}

// Profile
const profileEmail = ref(auth.user?.email || '')
const profileBio = ref(auth.user?.bio || '')
const profileZipCode = ref(auth.user?.zipCode || '')
const nbUsername = ref(auth.user?.numisBidsUsername || '')
const nbPassword = ref('')
const profilePublic = ref(auth.user?.isPublic || false)
const profileMsg = ref('')
const profileError = ref(false)
const profileSaving = ref(false)
const showPrivacyWarning = ref(false)

function onPublicToggle(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  if (!checked && profilePublic.value) {
    // Going from public → private: show warning
    ;(e.target as HTMLInputElement).checked = true // revert checkbox visually
    showPrivacyWarning.value = true
  } else {
    profilePublic.value = checked
  }
}

function confirmGoPrivate() {
  profilePublic.value = false
  showPrivacyWarning.value = false
}

function cancelGoPrivate() {
  showPrivacyWarning.value = false
}

const nbValidating = ref(false)
const nbValidationError = ref('')

async function handleSaveProfile() {
  profileMsg.value = ''
  profileError.value = false
  profileSaving.value = true
  nbValidationError.value = ''
  try {
    // If user entered new NB credentials, validate them first
    if (nbPassword.value && nbUsername.value) {
      nbValidating.value = true
      try {
        const valRes = await validateNumisBidsCredentials(nbUsername.value, nbPassword.value)
        if (!valRes.data.valid) {
          nbValidationError.value = valRes.data.error || 'Invalid NumisBids credentials'
          profileSaving.value = false
          nbValidating.value = false
          return
        }
      } catch {
        nbValidationError.value = 'Could not validate NumisBids credentials. Check your username and password.'
        profileSaving.value = false
        nbValidating.value = false
        return
      } finally {
        nbValidating.value = false
      }
    }

    const data: Record<string, unknown> = {
      email: profileEmail.value,
      bio: profileBio.value,
      zipCode: profileZipCode.value,
      isPublic: profilePublic.value,
      numisBidsUsername: nbUsername.value,
    }
    if (nbPassword.value) {
      data.numisBidsPassword = nbPassword.value
    }
    const res = await updateProfile(data as Parameters<typeof updateProfile>[0])
    if (auth.user) {
      auth.user.email = res.data.email
      auth.user.bio = res.data.bio
      auth.user.zipCode = res.data.zipCode
      auth.user.isPublic = res.data.isPublic
      auth.user.numisBidsUsername = res.data.numisBidsUsername
      auth.user.numisBidsConfigured = res.data.numisBidsConfigured
      localStorage.setItem('user', JSON.stringify(auth.user))
    }
    nbPassword.value = ''
    profileMsg.value = 'Profile saved'
  } catch {
    profileMsg.value = 'Failed to save profile'
    profileError.value = true
  } finally {
    profileSaving.value = false
  }
}

// Password
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordMsg = ref('')
const passwordError = ref(false)
const passwordLoading = ref(false)

async function handleChangePassword() {
  passwordMsg.value = ''
  passwordError.value = false

  if (newPassword.value !== confirmPassword.value) {
    passwordMsg.value = 'New passwords do not match'
    passwordError.value = true
    return
  }

  passwordLoading.value = true
  try {
    await changePassword(currentPassword.value, newPassword.value)
    passwordMsg.value = 'Password changed successfully'
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch {
    passwordMsg.value = 'Failed — check your current password'
    passwordError.value = true
  } finally {
    passwordLoading.value = false
  }
}

// Theme
const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')

function setTheme(t: Theme) {
  theme.value = t
  localStorage.setItem('theme', t)
  document.documentElement.setAttribute('data-theme', t)
}

// Timezone
const timezones = 'supportedValuesOf' in Intl
  ? (Intl as unknown as { supportedValuesOf: (key: string) => string[] }).supportedValuesOf('timeZone')
  : [] as string[]
const timezone = ref(localStorage.getItem('timezone') || Intl.DateTimeFormat().resolvedOptions().timeZone)

function saveTimezone() {
  localStorage.setItem('timezone', timezone.value)
}

// Default view
const defaultView = ref<'swipe' | 'grid'>((localStorage.getItem('defaultView') as 'swipe' | 'grid') || 'swipe')

function setDefaultView(v: 'swipe' | 'grid') {
  defaultView.value = v
  localStorage.setItem('defaultView', v)
}

// Default sort
const defaultSort = ref(localStorage.getItem('defaultSort') || 'updated_at_desc')

function saveDefaultSort() {
  localStorage.setItem('defaultSort', defaultSort.value)
}

// Data
const exporting = ref(false)
const exportingPdf = ref(false)
const dataMsg = ref('')
const dataError = ref(false)

async function handleExport() {
  exporting.value = true
  dataMsg.value = ''
  try {
    const res = await exportCollection()
    const blob = new Blob([res.data], { type: 'application/zip' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `ancient-coins-export-${new Date().toISOString().slice(0, 10)}.zip`
    a.click()
    URL.revokeObjectURL(url)
    dataMsg.value = 'Export downloaded'
  } catch {
    dataMsg.value = 'Export failed'
    dataError.value = true
  } finally {
    exporting.value = false
  }
}

async function handleExportPDF() {
  exportingPdf.value = true
  dataMsg.value = ''
  try {
    const res = await exportCatalogPDF()
    const blob = new Blob([res.data], { type: 'application/pdf' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `coin-catalog-${new Date().toISOString().slice(0, 10)}.pdf`
    a.click()
    URL.revokeObjectURL(url)
    dataMsg.value = 'PDF catalog downloaded'
  } catch {
    dataMsg.value = 'PDF generation failed'
    dataError.value = true
  } finally {
    exportingPdf.value = false
  }
}

async function handleImport(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  dataMsg.value = ''
  dataError.value = false

  try {
    const text = await file.text()
    const coins: Coin[] = JSON.parse(text)
    const res = await importCollection(coins)
    dataMsg.value = `Imported ${res.data.imported} coins`
  } catch {
    dataMsg.value = 'Import failed — ensure valid JSON format'
    dataError.value = true
  }
}

// API Keys
const apiKeys = ref<ApiKey[]>([])
const apiKeyName = ref('')
const newlyGeneratedKey = ref('')
const keyCopied = ref(false)
const generatingKey = ref(false)
const apiKeyMsg = ref('')
const apiKeyError = ref(false)

async function loadApiKeys() {
  try {
    const res = await listApiKeys()
    apiKeys.value = res.data
  } catch {
    // silently fail on load
  }
}

async function handleGenerateKey() {
  if (!apiKeyName.value.trim()) return

  generatingKey.value = true
  apiKeyMsg.value = ''
  apiKeyError.value = false
  newlyGeneratedKey.value = ''
  keyCopied.value = false

  try {
    const res = await generateApiKey(apiKeyName.value.trim())
    newlyGeneratedKey.value = res.data.key
    apiKeyName.value = ''
    await loadApiKeys()
  } catch {
    apiKeyMsg.value = 'Failed to generate API key'
    apiKeyError.value = true
  } finally {
    generatingKey.value = false
  }
}

async function copyKey() {
  try {
    await navigator.clipboard.writeText(newlyGeneratedKey.value)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 3000)
  } catch {
    // Fallback for non-HTTPS contexts
    const textarea = document.createElement('textarea')
    textarea.value = newlyGeneratedKey.value
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 3000)
  }
}

async function handleRevokeKey(id: number) {
  apiKeyMsg.value = ''
  apiKeyError.value = false
  try {
    await revokeApiKey(id)
    await loadApiKeys()
    newlyGeneratedKey.value = ''
  } catch {
    apiKeyMsg.value = 'Failed to revoke key'
    apiKeyError.value = true
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}

// WebAuthn Biometric
const supportsWebAuthn = !!window.PublicKeyCredential
const webauthnCredentials = ref<WebAuthnCredentialInfo[]>([])
const registeringCredential = ref(false)
const credentialMsg = ref('')
const credentialError = ref(false)

async function loadCredentials() {
  try {
    const res = await webauthnListCredentials()
    webauthnCredentials.value = res.data
  } catch {
    // silently fail
  }
}

function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
  const pad = base64.length % 4 === 0 ? '' : '='.repeat(4 - (base64.length % 4))
  const binary = atob(base64 + pad)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes.buffer
}

async function handleRegisterCredential() {
  registeringCredential.value = true
  credentialMsg.value = ''
  credentialError.value = false

  try {
    // Begin registration — get options from server
    const beginRes = await webauthnRegisterBegin()
    const options = beginRes.data

    // Convert base64url fields to ArrayBuffers for the browser API
    const publicKeyOptions: PublicKeyCredentialCreationOptions = {
      challenge: base64urlToBuffer(options.publicKey.challenge),
      rp: options.publicKey.rp,
      user: {
        id: base64urlToBuffer(options.publicKey.user.id),
        name: options.publicKey.user.name,
        displayName: options.publicKey.user.displayName,
      },
      pubKeyCredParams: options.publicKey.pubKeyCredParams,
      timeout: options.publicKey.timeout || 60000,
      authenticatorSelection: options.publicKey.authenticatorSelection,
      attestation: options.publicKey.attestation || 'none',
      excludeCredentials: (options.publicKey.excludeCredentials || []).map((c: { id: string; type: string; transports?: string[] }) => ({
        id: base64urlToBuffer(c.id),
        type: c.type,
        transports: c.transports,
      })),
    }

    // Call browser WebAuthn API (triggers Face ID / fingerprint prompt)
    const credential = await navigator.credentials.create({
      publicKey: publicKeyOptions,
    }) as PublicKeyCredential

    // Finish registration — send attestation to server
    await webauthnRegisterFinish(credential)

    credentialMsg.value = 'Biometric credential registered!'
    await loadCredentials()
  } catch (e: unknown) {
    credentialMsg.value = e instanceof Error ? e.message : 'Registration failed'
    credentialError.value = true
  } finally {
    registeringCredential.value = false
  }
}

async function handleDeleteCredential(id: number) {
  if (!await showConfirm('Remove this biometric credential?', { title: 'Remove Credential' })) return
  try {
    await webauthnDeleteCredential(id)
    await loadCredentials()
  } catch {
    credentialMsg.value = 'Failed to remove credential'
    credentialError.value = true
  }
}

// Saved Conversations
const conversations = ref<ConversationSummary[]>([])
const conversationsLoading = ref(false)
const showChat = ref(false)
const chatConversation = ref<{ id: number; title: string; messages: string } | null>(null)

async function loadConversations() {
  conversationsLoading.value = true
  try {
    const res = await listConversations()
    conversations.value = res.data
  } catch {
    // silently fail
  } finally {
    conversationsLoading.value = false
  }
}

async function openConversation(id: number) {
  try {
    const res = await getConversation(id)
    chatConversation.value = {
      id: res.data.id,
      title: res.data.title,
      messages: res.data.messages,
    }
    showChat.value = true
  } catch {
    await showAlert('Failed to load conversation', { title: 'Error' })
  }
}

async function handleDeleteConversation(id: number) {
  if (!await showConfirm('Delete this saved conversation?', { title: 'Delete Conversation', variant: 'danger' })) return
  try {
    await deleteConversation(id)
    conversations.value = conversations.value.filter(c => c.id !== id)
  } catch {
    await showAlert('Failed to delete conversation', { title: 'Error' })
  }
}

async function handleRefresh() {
  await Promise.all([
    loadApiKeys(),
    loadConversations(),
    loadBlockedUsers(),
    supportsWebAuthn ? loadCredentials() : Promise.resolve(),
  ])
}

onMounted(() => {
  loadApiKeys()
  loadConversations()
  loadBlockedUsers()
  loadTags()
  if (supportsWebAuthn) loadCredentials()
})
</script>

<style scoped>
.avatar-section {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.avatar-preview {
  flex-shrink: 0;
}

.avatar-img {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--accent-gold-dim, #c9a84c);
}

.avatar-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.settings-layout {
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
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
}

.tab-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tab-btn:hover:not(.active) {
  color: var(--text-primary);
}

.settings-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.settings-section h3 {
  font-size: 0.95rem;
  margin-top: 1.25rem;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 1rem;
}

.setting-item:last-child {
  border-bottom: none;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.setting-label {
  font-size: 0.9rem;
  font-weight: 500;
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.nb-status {
  font-size: 0.82rem;
  padding: 0.4rem 0.75rem;
  border-radius: var(--radius-sm);
  margin-top: 0.25rem;
}

.nb-status.connected {
  background: rgba(74, 222, 128, 0.1);
  color: #4ade80;
  border: 1px solid rgba(74, 222, 128, 0.2);
}

.nb-status.validating {
  background: rgba(250, 204, 21, 0.1);
  color: #facc15;
  border: 1px solid rgba(250, 204, 21, 0.2);
}

.nb-status.error {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.setting-value {
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.password-form {
  max-width: 350px;
}

.theme-toggle {
  display: flex;
  gap: 0.25rem;
  background: var(--bg-primary);
  border-radius: var(--radius-full);
  padding: 0.2rem;
}

.theme-btn {
  padding: 0.35rem 0.75rem;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.theme-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tz-select {
  max-width: 250px;
}

.sort-select {
  max-width: 250px;
}

.import-btn {
  cursor: pointer;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: #e74c3c;
}

.apikey-generate {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 0.75rem;
}

.apikey-generate .form-input {
  flex: 1;
  max-width: 280px;
}

.apikey-reveal {
  background: var(--bg-primary);
  border: 1px solid var(--accent-gold-dim);
  border-radius: var(--radius-sm);
  padding: 0.75rem 1rem;
  margin-bottom: 0.75rem;
}

.apikey-reveal-warning {
  font-size: 0.8rem;
  color: var(--accent-gold);
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.apikey-reveal-box {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.apikey-reveal-value {
  flex: 1;
  font-size: 0.78rem;
  background: var(--bg-card);
  padding: 0.4rem 0.6rem;
  border-radius: var(--radius-sm);
  word-break: break-all;
  user-select: all;
}

.apikey-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.apikey-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.6rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 0.75rem;
}

.apikey-item:last-child {
  border-bottom: none;
}

.apikey-item.revoked {
  opacity: 0.5;
}

.apikey-item-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.apikey-item-name {
  font-size: 0.9rem;
  font-weight: 500;
}

.apikey-item-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.revoked-badge {
  font-size: 0.7rem;
  padding: 0.15rem 0.5rem;
  background: var(--bg-primary);
  border-radius: var(--radius-full);
  color: var(--text-muted);
}

.btn-danger {
  background: #e74c3c;
  color: #fff;
  border: none;
  cursor: pointer;
}

.btn-danger:hover {
  background: #c0392b;
}

/* Settings hamburger menu (PWA) */
.page-header {
  display: flex;
  flex-direction: row !important;
  justify-content: space-between;
  align-items: center !important;
  margin-bottom: 1.5rem;
  flex-wrap: nowrap;
}

.settings-menu-wrapper {
  position: relative;
}

.settings-menu-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
}

.settings-dropdown {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  z-index: 50;
  min-width: 180px;
  padding: 0.3rem;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.settings-dropdown-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 0.75rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
  width: 100%;
}

.settings-dropdown-item.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.settings-dropdown-item:hover:not(.active) {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 640px) {
  .setting-item {
    flex-direction: column;
    align-items: stretch;
  }

  .tab-nav {
    flex-wrap: wrap;
  }

  .tab-btn {
    font-size: 0.78rem;
    padding: 0.5rem 0.6rem;
  }
}

.conv-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.loading-inline {
  color: var(--text-muted);
  font-style: italic;
  padding: 0.5rem 0;
}

/* Help Section */
.help-section {
  overflow: hidden;
  min-width: 0;
}

.help-section h2 {
  margin-bottom: 0.5rem;
}

.help-accordion {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  margin-bottom: 0.5rem;
  overflow: hidden;
}

.help-accordion[open] {
  border-color: var(--accent-gold-dim);
}

.help-summary {
  padding: 0.75rem 1rem;
  font-weight: 600;
  font-size: 0.95rem;
  cursor: pointer;
  background: var(--bg-primary);
  color: var(--text-primary);
  list-style: none;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  transition: background var(--transition-fast);
}

.help-summary::-webkit-details-marker {
  display: none;
}

.help-summary::before {
  content: '▸';
  font-size: 0.8rem;
  color: var(--text-muted);
  transition: transform 0.2s;
}

.help-accordion[open] > .help-summary::before {
  transform: rotate(90deg);
}

.help-summary:hover {
  background: var(--bg-card-hover, var(--bg-card));
}

.help-content {
  padding: 1rem;
  font-size: 0.9rem;
  line-height: 1.65;
  color: var(--text-secondary);
  overflow-x: auto;
}

.help-content h4 {
  color: var(--text-primary);
  margin: 1.25rem 0 0.5rem;
  font-size: 0.9rem;
}

.help-content h4:first-child {
  margin-top: 0;
}

.help-content p {
  margin-bottom: 0.75rem;
}

.help-content ul {
  margin: 0 0 0.75rem 1.25rem;
  padding: 0;
}

.help-content li {
  margin-bottom: 0.35rem;
}

.help-content a {
  color: var(--accent-gold);
  text-decoration: none;
}

.help-content a:hover {
  text-decoration: underline;
}

.help-tip {
  background: var(--accent-gold-glow, rgba(212, 175, 55, 0.08));
  border-left: 3px solid var(--accent-gold);
  padding: 0.6rem 0.85rem;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  font-size: 0.85rem;
  margin: 0.75rem 0;
}

.help-warning {
  background: rgba(231, 76, 60, 0.08);
  border-left: 3px solid #e74c3c;
  padding: 0.6rem 0.85rem;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  font-size: 0.85rem;
  margin: 0.75rem 0;
  color: var(--text-primary);
}

.help-table {
  width: 100%;
  border-collapse: collapse;
  margin: 0.75rem 0;
  font-size: 0.85rem;
  table-layout: fixed;
}

.help-table th,
.help-table td {
  padding: 0.5rem 0.65rem;
  text-align: left;
  border-bottom: 1px solid var(--border-subtle);
  word-wrap: break-word;
  overflow-wrap: break-word;
}

.help-table th {
  color: var(--text-muted);
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.help-table tr:last-child td {
  border-bottom: none;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 15vh 1rem;
  z-index: 1000;
}

.modal-content {
  width: 100%;
}

.modal-header {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-body {
  padding: 1.25rem;
}

/* Tag Manager */
.tag-create-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  margin: 1rem 0;
}

.tag-create-form .form-input {
  flex: 1;
  min-width: 150px;
}

.tag-color-picker {
  display: flex;
  gap: 0.3rem;
  align-items: center;
}

.color-swatch {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
}

.color-swatch.active {
  border-color: var(--text-primary);
  box-shadow: 0 0 0 2px var(--bg-card);
}

.color-swatch.sm {
  width: 18px;
  height: 18px;
}

.tag-error {
  color: #ef4444;
  font-size: 0.85rem;
  margin-top: 0.25rem;
}

.tag-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tag-list-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  flex-wrap: wrap;
}

.tag-preview {
  font-size: 0.8rem;
  padding: 0.2rem 0.6rem;
  border-radius: 9999px;
  border: 1px solid;
  flex-shrink: 0;
}

.tag-edit-input {
  flex: 1;
  min-width: 120px;
}

.tag-actions {
  margin-left: auto;
  display: flex;
  gap: 0.25rem;
}

.empty-tags {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-top: 1rem;
}
</style>
