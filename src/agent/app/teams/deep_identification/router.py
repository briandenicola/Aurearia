"""Router node (contracts/agent-internal-contract.md §6, T058).

A single LLM call constrained to the supplied `provider_catalog` entries
selects which *automatable* providers to actually query this run, bounded
by `bounds.max_providers`. Non-automatable catalog entries (`ngc`, `ocre`,
`rpc`) are never subject to router reasoning — they always run (trivially,
with no network call) and are represented directly in the final evidence
list by their own provider nodes, so they are excluded from `selected`/
`skipped` here (those two lists describe only automatable candidates the
router did or didn't pick).

`provider_override`, when non-empty, replaces the LLM call entirely: only
catalog-listed automatable providers named in the override are selected,
and no other automatable provider runs. The override can never introduce a
provider absent from the catalog — Go controls the closed candidate list.
"""

import json
import logging

from app.llm.content import extract_text_content
from app.llm.retry import ainvoke_with_retry
from app.models.requests import DeepProviderCatalogEntry
from app.safety import with_safety
from app.teams.deep_identification.merge import PROVIDER_RANK
from app.teams.deep_identification.state import RouterSkip

logger = logging.getLogger(__name__)

ROUTER_PROMPT = with_safety("""You are the provider router for a numismatic deep-identification pipeline.
You will be given a list of automated catalog/reconciliation providers that
are legal to query for this run, plus brief context about the coin. Decide
which of the listed providers are worth querying, given the context.

Respond with a single JSON object only:

{"selected": ["numista"], "rationale": "one short sentence"}

Rules:
- "selected" MUST be a subset of the provider names you were given. Never
  invent a provider name that was not listed.
- If you have no strong reason to exclude a listed provider, include it —
  omitting a plausibly-useful provider is worse than an extra call.
- Do not use emojis.""")


class RouterDecision:
    """Result of the router step."""

    def __init__(self, selected: list[str], skipped: list[RouterSkip], rationale: str) -> None:
        self.selected = selected
        self.skipped = skipped
        self.rationale = rationale


def _automatable_names(catalog: list[DeepProviderCatalogEntry]) -> list[str]:
    return [entry.provider for entry in catalog if entry.automatable]


def _rank(name: str) -> int:
    try:
        return PROVIDER_RANK.index(name)
    except ValueError:
        return len(PROVIDER_RANK)


async def route(
    model,
    catalog: list[DeepProviderCatalogEntry],
    provider_override: list[str],
    max_providers: int,
    notes: str = "",
) -> RouterDecision:
    """Select which automatable providers to query this run.

    `model` is an already-constructed chat model (see app/llm/provider.py);
    passed in so tests can substitute a fake without touching LLM config.
    """
    automatable = _automatable_names(catalog)
    if not automatable:
        return RouterDecision(selected=[], skipped=[], rationale="no automatable providers in catalog")

    budget = max(0, max_providers)

    if provider_override:
        # Override wins outright — no LLM call, no free-form provider names.
        allowed_override = [p for p in provider_override if p in automatable]
        selected = sorted(allowed_override, key=_rank)[:budget] if budget else []
        skipped = [
            RouterSkip(provider=p, reason="not requested by provider_override")
            for p in automatable
            if p not in selected
        ]
        return RouterDecision(
            selected=selected, skipped=skipped, rationale="provider_override supplied by caller"
        )

    human_text = (
        f"Automated providers available this run: {json.dumps(automatable)}\n"
        f"Coin notes/context (may be empty): {notes[:500]}"
    )
    try:
        response = await ainvoke_with_retry(model, [_system_message(), _human_message(human_text)])
        raw = extract_text_content(response.content)
        decision = _parse_router_json(raw)
    except Exception:
        logger.exception("[deep_identification.router] router LLM call failed — selecting all automatable providers")
        decision = {"selected": automatable, "rationale": "router failed; defaulting to all automatable providers"}

    proposed = [p for p in decision.get("selected", []) if isinstance(p, str) and p in automatable]
    if not proposed:
        proposed = list(automatable)
    selected = sorted(dict.fromkeys(proposed), key=_rank)[:budget] if budget else []
    skipped = [
        RouterSkip(
            provider=p,
            reason="max_providers limit reached" if p in proposed else "not selected by router",
        )
        for p in automatable
        if p not in selected
    ]
    rationale = str(decision.get("rationale") or "")[:500]
    return RouterDecision(selected=selected, skipped=skipped, rationale=rationale)


def _parse_router_json(raw: str) -> dict:
    text = raw.strip()
    start = text.find("```json")
    if start != -1:
        start += len("```json")
        end = text.find("```", start)
        text = text[start:end if end != -1 else None].strip()
    parsed = json.loads(text)
    if not isinstance(parsed, dict):
        raise ValueError("router response is not a JSON object")
    return parsed


def _system_message():
    from langchain_core.messages import SystemMessage

    return SystemMessage(content=ROUTER_PROMPT)


def _human_message(text: str):
    from langchain_core.messages import HumanMessage

    return HumanMessage(content=text)
