# Deep Analysis

> Run an optional, resumable identification workflow that combines image
> evidence with bounded numismatic reference providers.

## Overview

Quick Identify remains the default fast path. Deep Analysis is an explicit
opt-in for cases where the collector wants more evidence before saving or
updating a coin. It accepts obverse and reverse images, optional notes, and
optional hint/reference images. Hint images are temporary analysis context and
are deleted after the job settles.

Deep Analysis runs as a persisted background job. Leaving the page does not
cancel it. The UI can reconnect to replayable Server-Sent Events, cancel active
work, retry a settled job, review a cited report, edit proposed fields, and
choose which accepted fields to apply.

No analysis result writes to a coin or intake draft automatically.

## Provider Boundaries

| Provider | Automated role | Attribution / boundary |
|---|---|---|
| Image model | Observes coin faces and normalizes visible evidence | Uses the configured Anthropic or Ollama model |
| Numista | Catalog candidates through configured API access | Subject to Numista API terms and quotas |
| Nomisma | Controlled-vocabulary reconciliation, including mint concepts | Nomisma.org, CC BY 4.0 |
| OCRE | Roman Imperial coin-type candidates through fixed-template Nomisma SPARQL | Online Coins of the Roman Empire, American Numismatic Society, ODbL 1.0 |
| NGC | Certification extraction and official verification link | No automated NGC data API or scraping |
| RPC | Not automated | Paused: no supported API or downloadable corpus is available; no RPC images or data are ingested |

OCRE queries interpolate only validated Nomisma identifier slugs into fixed
templates. Free text and legends are used for local scoring, not as executable
SPARQL.

## Workflow

1. Start from Coin Lookup/new intake or a saved coin with obverse and reverse
   images.
2. Optionally provide notes, hint images, or a provider override.
3. The router selects available providers within configured limits.
4. Provider progress streams to the UI and remains replayable after reconnect.
5. The pipeline evaluates contradictions and synthesizes a cited report.
6. Review and edit the proposal, accept individual fields, then explicitly apply
   them to a saved coin or intake draft.

Jobs settle as completed, partial, failed, or cancelled. One provider timing out
or being unavailable does not fail evidence returned by other providers.

## Configuration

Both controls default to off:

- `DeepIdentificationEnabled` enables the background workflow.
- `DeepIdentificationOCREEnabled` allows OCRE to be selected.

Admins can also configure worker count, per-user concurrency, queue depth,
timeout, retention, provider count, and provider call budgets. Disabling OCRE
is an immediate rollback: no new OCRE requests are made.

`DeepIdentificationRPCEnabled` remains false and does not make RPC automatable.

## Privacy and Security

- Jobs and artifacts are owner-scoped.
- The Go API owns persistence, provider HTTP calls, citation allowlists, and
  confirm-gated writes.
- Python remains stateless and calls Go provider tools with short-lived,
  job-scoped credentials.
- Full provider claims are consumed internally for synthesis; replayable public
  events contain bounded status and count data.
- Hint images are ephemeral and are not promoted into the coin image gallery.

## Related Decisions

- [ADR 0009 - Nomisma Authority Linking](../adr/0009-nomisma-authority-linking.md)
- [ADR 0010 - OCRE ODbL Provider](../adr/0010-ocre-odbl-provider.md)
- [Coin Lookup](coin-lookup.md)
- [AI Coin Analysis](ai-analysis.md)

