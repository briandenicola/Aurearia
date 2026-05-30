# Scale Planning Spike (2026-05-30)

## Source

- Task ID: `cec55dbd-0b3f-4ee5-8254-f5a20bb4717d`
- Session ID: `e33975b9-c90c-45dd-aec0-1bd7292bd70a`
- Origin: Private Copilot task output copied into repository for team visibility and long-term retention.

## Overview

The application is currently designed as a personal-scale, self-hosted tool (PRD: "depth over scale"). This spike investigates and plans what architectural changes are needed to support **2,000–10,000 coins per user** with a **user base of 100 or more people** (about 1M total coins).

## Identified Bottlenecks

| Layer | Current State | Problem at Scale |
|---|---|---|
| Database | Single SQLite file, WAL mode | Write serialization, no connection pooling, no horizontal reads |
| Image storage | Local filesystem (`uploads/coin-{id}/`) | Unbounded disk growth; at 2 images x 10K coins x 100 users = 2M files, inode/directory limits hit |
| AI analysis | Synchronous Ollama/Anthropic calls per request | Analysis queue saturates at high coin counts and concurrent users |
| Valuation runs | Scheduler iterates all coins per user | O(n) per user per run, blocking at high coin counts |
| Search/filtering | Full-table GORM queries with OFFSET pagination | SQLite OFFSET degrades significantly past around 50K rows |
| Coin of the Day | In-memory `map[uint]string` idempotency cache | Lost on restart, does not scale across processes |

## Proposed Work Areas

### Phase 1 — Database Layer

- Add composite indexes on `coins` table: `(user_id, created_at)`, `(user_id, category)`, `(user_id, material)`, `(user_id, status)`
- Replace OFFSET pagination with keyset (cursor) pagination on coins list endpoint
- Evaluate SQLite headroom vs. PostgreSQL migration for concurrent write scenarios
- Add connection pool limits (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`)

### Phase 2 — Image Storage

- Move image storage off local filesystem to an object store (S3-compatible / MinIO for self-hosted)
- Implement lazy thumbnail generation (serve originals from object store, generate 300px thumbs on first access)
- Add `Cache-Control` headers on image responses plus CDN/nginx caching layer

### Phase 3 — AI / Agent Workload

- Move AI analysis calls to an async job queue (pgq/River or Redis-backed Asynq)
- Add per-user rate limiting and global concurrency cap on AI endpoints
- Ensure cached analysis columns (`ai_analysis`, `obverse_analysis`, `reverse_analysis`) are checked before re-requesting

### Phase 4 — Search and Discovery

- Add SQLite FTS5 virtual table for inscription, name, and description text search (replaces `LIKE '%term%'` full scans)
- Add indexed denormalization or short-TTL cache for portfolio summary aggregations (category counts, total value)

### Phase 5 — Application Server

- Add nginx/Caddy reverse proxy for connection limiting, TLS, gzip, and static asset caching
- Tune Go runtime (`GOMAXPROCS`, pprof admin endpoint)
- Externalize Coin of the Day idempotency to the `featured_coins` DB table (add `last_picked_date` per user)

## Scale Estimates

| Change | Covers |
|---|---|
| Indexes + keyset pagination | 1K–100K coins/user, 100–500 users |
| SQLite → PostgreSQL | 100K+ coins/user, 1000+ concurrent users |
| Object store for images | Any scale beyond around 50K total images |
| FTS5 | Text search on 10K+ coins without latency regression |
| Async AI queue | 50+ concurrent AI requests |

## What Does Not Need to Change

- Three-service architecture (Go API / Vue SPA / Python agent) handles this scale without re-platforming
- GORM repository layer abstracts the database; most changes are config plus query tuning
- JWT auth and WebAuthn are stateless and scale horizontally without changes
- Social features (follows, comments) already have composite unique indexes

## Next Steps

1. Decide SQLite vs. PostgreSQL at target scale.
2. Decide object store strategy (MinIO self-hosted vs. cloud S3).
3. Decide async queue approach.
4. Convert outcomes into updated `spec.md`, `plan.md`, and `tasks.md`.
