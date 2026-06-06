# Admin Settings

> Configure AI providers, manage users, enable scheduled tasks, and manage system-wide settings.

## Overview

Admin Settings are accessed by the first registered user (admin) and provide configuration for AI providers, user management, logging, and scheduled tasks.

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

## Logs Tab

**Real-Time Logging:**
- View application logs as they occur
- Filter by log level
- Auto-refresh toggle
- Export logs to file

## Availability Checks Tab

**Automatic Wishlist Availability Checking:**
- Enable/disable checks
- Set daily start time (e.g., 2:00 AM)
- Set repeat interval (e.g., every 120 minutes)
- View run history with per-coin drill-down:
  - URL checked
  - Status (Available/Unavailable/Unknown)
  - HTTP code and reason
  - Whether AI agent was used

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
```

## Security Notes

- Only first registered user can access Admin
- No multi-admin support currently
- All settings are per-instance (not per-user)

See also: [AI Coin Analysis](ai-analysis.md), [Auction Tracking](auction-tracking.md), [Authentication](../authentication.md)
