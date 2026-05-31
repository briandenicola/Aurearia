# Epic — Agentic Collection: Structured Data → AI Entry → Conversational Access

> **Status:** Backlog (epic)
> **Type:** Epic / theme
> **Member cards:** Structured Catalog References · Agentic Coin Entry ·
> Collection Chat · Collection Tool Server (External)

## Summary

A four-card arc that moves the app from *manual data entry + web-search AI*
toward *structured data, AI-assisted entry, and conversational access to the
collection itself* — including from external clients. Each card is shippable on
its own, but together they compound: better-structured data makes AI entry more
useful, AI entry produces richer records to query, and a shared tool layer lets
both the in-app agent and external clients (OpenWebUI/Ollama, Claude Desktop)
query and update the collection.

## Why these belong together

The existing AI agent only looks *outward* (web search for wishlist, auctions,
prices). This epic turns the intelligence *inward* — onto the collection the
user already owns — and raises the quality of the underlying data so that
inward-facing AI has something solid to work with.

The connective tissue is a **transport-agnostic collection tool layer** (Go):
query / filter / aggregate / propose-update / commit-update, scoped per user,
with the Go API as the only writer. In-app chat and the external server are both
adapters onto it. Designing this layer once is the central architectural bet of
the epic.

## Member cards

|#|Card                                 |One-liner                                                                                                                    |Primary value                                                                |
|-|-------------------------------------|-----------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------|
|1|**Structured Catalog References**    |Normalize attribution (RIC/RPC/Sear/KM…) into a `CoinReference` model + `era`, with per-catalog validation and authority URIs|Clean, portable, machine-usable reference data across ancient/medieval/modern|
|2|**Agentic Coin Entry**               |New `coin_intake` agent team drafts a fully-populated coin from photos / OCR / lookups; user reviews & confirms              |Cuts data entry to review-and-confirm; produces richer records               |
|3|**Collection Chat**                  |New `collection_chat` team answers questions and makes guarded updates over the user’s own collection via tool-calling       |Conversational read + confirm-gated write, in-app                            |
|4|**Collection Tool Server (External)**|Re-exposes the same tool layer to external clients (OpenWebUI/Ollama, MCP) with full read + write parity                     |Use the collection from outside the app                                      |

## Dependency graph

```
        ┌────────────────────────────┐
        │ 1. Structured Catalog Refs  │  (independent; soft-enables 2 & 3)
        └──────────────┬─────────────┘
                       │ soft (richer references)
                       ▼
        ┌────────────────────────────┐
        │ 2. Agentic Coin Entry       │  (independent; better with 1)
        └────────────────────────────┘

        ┌────────────────────────────┐
        │ 3. Collection Chat          │  ◄── establishes the shared TOOL LAYER
        │    (collection_chat team)   │       (the spine of cards 3 & 4)
        └──────────────┬─────────────┘
                       │ hard (re-exposes same tools over a transport)
                       ▼
        ┌────────────────────────────┐
        │ 4. Collection Tool Server   │  (external adapter; full read+write parity)
        └────────────────────────────┘
```

- **1 → 2, 3 (soft):** structured references make AI entry and reference-based
  queries materially better, but 2 and 3 can ship without 1.
- **3 → 4 (hard):** the external server re-exposes card 3’s tool layer. Build 3
  with transport-agnostic tools so 4 doesn’t force a refactor.
- **1 ↔ 2 (soft, bidirectional):** entry can populate candidate references;
  references give entry a target schema.

## Proposed build order

1. **Structured Catalog References (1).** Foundational, independent, low-risk,
   and lifts the data quality everything else builds on. Do first.
1. **Collection Chat (3) — read-only slice first.** Stand up the
   transport-agnostic tool layer + `get_coin` / `query_coins` / `aggregate`,
   scoped per user, in the existing chat drawer. This is the spine; getting it
   right de-risks card 4.
1. **Agentic Coin Entry (2).** Can be built in parallel with 3 (different agent
   team, different surface). Sequence by appetite; it benefits from 1 being done.
1. **Collection Chat (3) — write slice.** Add the two-phase
   `propose_update` / `commit_update` confirm-gated path once read + resolution
- disambiguation are solid.
1. **Collection Tool Server (4).** Once the tool layer is proven and writes
   work in-app, expose it externally over one transport (whichever OpenWebUI
   integrates with most cleanly), read-only first, then write parity.

> Parallelizable: card 2 is independent of the 3 → 4 spine and can slot in
> whenever. Cards 1 and the read-only slice of 3 are the highest-leverage
> starting points.

## Cross-cutting principles (apply to all member cards)

- **Agent proposes, user commits.** No silent writes anywhere — entry drafts,
  chat previews, external writes all keep a confirm step (in-app UI or
  protocol-level two-phase for external).
- **Go API is the only writer; agent stays stateless.** All AI surfaces act
  through the tool layer / API, never the DB directly (ADR 0002).
- **Server-side user scoping.** User identity comes from auth context / API key,
  never from agent- or client-supplied parameters. No cross-user or
  social-collection access via any AI surface.
- **One tool layer, many adapters.** Query/update logic lives once, transport-
  agnostic; in-app agent and external server are adapters. This is what keeps
  cards 3 and 4 from duplicating logic.
- **Auditability.** AI- and external-initiated updates are recorded in the coin
  activity journal, tagged by source.
- **Structured over prose.** Agent outputs that feed UI (entry drafts, chat
  results, references) are structured/typed, not freeform text.

## Epic-level open questions

- **Protocol for external access (card 4):** MCP, OpenAPI, or both? Verify
  OpenWebUI/Ollama’s current support before committing.
- **One chat surface or two (card 3):** merge collection mode into the existing
  agent drawer with intent detection, or a distinct entry point?
- **Confidence representation (card 2):** numeric vs. coarse buckets.
- **External write confirm model (card 4):** protocol-enforced two-phase vs.
  trusted single-call gated by a per-key capability flag.
- **Updatable-field allowlist (cards 3 & 4):** which fields can be changed
  conversationally / externally in v1 (value/grade/notes/tags safe; identity
  fields like category/era likely excluded initially).

## Definition of done (epic)

- [ ] All four member cards shipped and individually meet their acceptance
  criteria.
- [ ] A single transport-agnostic collection tool layer backs both in-app chat
  and the external server (no duplicated query/update logic).
- [ ] In-app chat and external clients have read + write/update parity.
- [ ] All AI/external write paths are confirm-gated and journaled.
- [ ] Threat model updated for the inward-facing AI surface and the external
  transport.
- [ ] Docs updated: features.md, api-reference.md, and a new doc for external
  tool-server setup (OpenWebUI/Ollama).