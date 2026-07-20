# Admin Settings

> Configure AI providers, manage users, enable scheduled tasks, and manage system-wide settings.

## Overview

Admin Settings are accessed by admins and provide configuration for AI providers, user management, logging, OIDC, catalogs, security, and scheduled tasks. The first registered user is assigned admin automatically.

## Users Tab

**User Management:**
- View all registered users
- Delete user accounts (with confirmation)
- Reset user passwords
- See user creation date
- View last login time

## AI Configuration Tab

**Provider Selection:**
- Choose **Anthropic Claude** (recommended) or **Ollama**
- Both require valid configuration before AI features work

**Anthropic Setup:**
1. Get API key: [console.anthropic.com](https://console.anthropic.com/)
2. Paste into **Anthropic API Key**
3. Model auto-populated from API; manually select if desired
4. (Optional) Customize agent prompts

**Ollama Setup:**
1. Run Ollama: `ollama serve`
2. Set **Ollama URL** (default: http://localhost:11434)
3. Set **Vision Model** (default: llava)
4. Set timeout (default: 300s)
5. Set **SearXNG URL** for web search (default: http://localhost:8888)

## System Tab

**Application Settings:**
- **Log Level** — trace, debug, info, warn, error
- **Numista API Key** — For coin catalog lookups
- **Pushover App Token** — Enables push notifications for configured users
- **Public App URL** — Used in outbound links and notifications
- **Version Info** — Displays app version/build date

## Coin Properties Tab

**Configurable Coin Metadata:**
- **Categories** — Newline-delimited list of category values shown in coin forms
- **Eras** — Newline-delimited list of era values shown in coin forms
- **Lookup Compatibility** — Coin Lookup normalizes extracted era values to backend-supported save values while user-facing forms use the configured lists
- **Defaults** — Roman, Greek, Byzantine, Modern, Other categories and ancient, medieval, modern eras are available by default

## Catalogs Tab

Manage normalized numismatic catalog definitions used by structured references:

- Configure catalog labels and metadata
- Mark catalogs that require volume values, such as RIC/RPC/SNG
- Keep catalog rules consistent with create/edit validation

## OIDC Login Tab

Configure OpenID Connect providers for login and account linking:

- Microsoft Entra ID and generic OIDC/Pocket ID-style providers
- Write-only client secrets
- Separate login and account-link callback URLs
- Provider tests before enabling

## Security Tab

Monitor account security controls:

- Registration mode
- User lockout/unlock actions
- Local recovery-admin visibility

## Logs Tab

**Real-Time Logging:**
- View application logs as they occur
- Filter by log level
- Auto-refresh toggle
- Export logs to file

## Schedules Tab

Centralized schedule configuration for background jobs:

- **Wishlist Availability** — Automatic URL availability checks and run history
- **Wishlist Search Alerts** — Scheduled alert discovery for non-manual alerts
- **Auction Ending** — Reminders for watched/bidding auction lots ending soon
- **Auction Watch Bid Digest** — Digest notifications for watched auction bidding activity
- **Coin of the Day** — Daily featured coin notifications and manual run
- **Collection Health** — Daily health snapshots
- **Collection Valuation** — Scheduled valuation runs, manual trigger, and cancellation

## Valuation Runs Tab

**Scheduled Collection Valuations:**
- Enable/disable automated valuation runs
- Configure interval (default: 7 days)
- Set start time (default: 03:00 AM)
- Set max coins per run (default: 50)
- View run history:
  - Run timestamp
  - Coins processed
  - Updated values
  - Agent usage stats
- Manual trigger button
- Cancel in-progress runs

## Health Tab

Operational health views for scheduler/job runs and collection health snapshots.

## API Endpoints

```
GET    /admin/users                  # List users
DELETE /admin/users/:id              # Delete user
POST   /admin/users/:id/reset-password # Reset password
PUT    /admin/users/:id/role         # Update user role

GET    /admin/settings               # Get all settings
GET    /admin/settings/defaults      # Get default settings
PUT    /admin/settings               # Update settings

GET    /admin/logs                   # View logs
GET    /admin/test-anthropic         # Test Anthropic connection
GET    /admin/test-searxng           # Test SearXNG connection

GET    /admin/availability-runs      # View availability check history
GET    /admin/availability-runs/:id  # View one availability run

GET    /admin/valuation-runs         # View valuation run history
GET    /admin/valuation-runs/:id     # View one valuation run
POST   /admin/valuation-runs/trigger # Trigger valuation run
POST   /admin/valuation-runs/:id/cancel # Cancel run
GET    /admin/coin-of-day-runs       # View coin-of-day run history
POST   /admin/coin-of-day/run        # Trigger coin-of-day asynchronously
```

## Security Notes

- The first registered user becomes an admin automatically
- Admin users can promote/demote other users through role management
- Sensitive settings such as provider secrets are write-only after saving
- Most settings are per-instance; user-specific preferences remain in user/account settings

See also: [AI Coin Analysis](ai-analysis.md), [Auction Tracking](auction-tracking.md), [Coin Lookup](coin-lookup.md), [Authentication](../authentication.md)
