# #774 Listing title relevance check (2026-09-29)

Status: implemented, verified locally and accepted by the owner (fake-model evidence).

## Change
- `src/agent/app/teams/listing_relevance.py` (new): one structured LLM call labels
  dealer listing titles `match` / `related` / `unrelated`. Titles are marked untrusted
  data in the prompt and truncated to 300 chars. It is bounded by an 8s `asyncio.wait_for`,
  and each call logs `titles`, `outcome` and `elapsed_ms`. Any error, timeout, or
  incomplete or invalid answer returns None, and the existing ranking is kept.
- `order_by_relevance`: matches, then related (stable within each group); unrelated
  dropped. If every title is labelled unrelated, the labels are ignored (never empties a result).
- `coin_search.run_market_search`: when an LLM config is present, it searches with
  headroom (`MAX_ITEMS`), applies the check once after all providers return (no per-site
  latency), trims to the requested limit (updating truncation), and warns how many
  listings were left out. Without an LLM config, behaviour is unchanged.
- The module lives in `app/teams` (not `app/tools`) because tools do not import `app.llm`.
- `tests/conftest.py`: an autouse stub makes the check fail open so tests stay offline.
- Docs: `docs/features/how-coin-copilot-works.md` limits and diagrams.

## Evidence
- `tests/test_listing_relevance.py` (11 tests): Abbasid "Caligula of the Islamic world"
  is dropped; Germanicus "struck under Caligula" stays after matches; replacement up to
  the limit; raise/incomplete/all-unrelated keep the baseline; timeout bound; logging;
  no call without an LLM config; dict-parsed answers.
- Tamper tests (each reverted): all-unrelated guard, incomplete padding, timeout,
  exception guard, trim, headroom and ordering each failed at least one test.
- `task check:agent` (ruff, 756 passed) and `task check:delivery` passed.

## Review
- aurearia-reviewer: INCOMPLETE. No blocking bugs; status, budget, caller and injection checks passed.
  The gaps were real-model labelling evidence and a measured real `elapsed_ms`.
- Owner decision (2026-09-29): accept the fake-model tests instead of a live LLM call.
  The real latency will be seen in the `Listing relevance check ... elapsed_ms=` agent logs.
- Follow-ups applied: a cancellation check before the relevance call, and the dropped-count
  warning placed first. Follow-ups deferred: capping drops per call (seller-controlled titles),
  and the "offer more" wording when drops leave fewer than the limit.

## Next
Owner order: #779, #771, #766, #784. Confirm with the owner before starting #779.
