"""Team: Dynamic Set Builder — group-chat/magentic-style workflow that turns a
free-text prompt into a structured Set Proposal for human review.

Contract anchor: specs/011-dynamic-set-builder-correction-plan.md (Phase 2)
and `specs/spec - Dynamic Set Builder.md` (FR-001..FR-016).

Roles:
- Intent Analyst: extracts numismatic subject, scope, ambiguity, boundedness.
  Declines non-numismatic prompts and asks for clarification instead of
  guessing when a prompt can't be resolved to at least one concrete scope.
- Roster Researcher: enumerates candidate roster slots (using web search when
  external lookups are enabled), producing structured entries only.
- Collection Matcher: uses the collection summary Go passed in to estimate a
  filled/total preview. Never touches a database directly.
- Validator/Critic: annotates every slot verified/unverified instead of
  silently trusting or silently dropping unconfirmed entries.
- Orchestrator: enforces max-turn/budget limits and assembles the final
  structured proposal (or a clarification/rejection/failure result).

This module never creates or modifies a set, slot, or coin. It returns data
only — Go owns persistence, notifications, and the approval-gated creation
transaction (Phase 3+).
"""

import json
import logging
from typing import Any, TypedDict

from langchain_core.messages import HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model, get_search_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig, PortfolioSummary, SetBuilderRequest
from app.models.responses import (
    SetBuilderPrematchSummary,
    SetBuilderProposal,
    SetBuilderResponse,
    SetBuilderScopeOption,
    SetBuilderSlot,
)
from app.safety import with_safety
from app.teams.json_extraction import extract_json_payload

logger = logging.getLogger(__name__)

INTENT_PROMPT = with_safety("""You are the Intent Analyst for a numismatic (coin collecting) set-builder
workflow. Read the user's free-text prompt and determine what collectible
coin set, if any, they want built. Respond with a single JSON object only:

{
  "subject": "short description of the requested set",
  "is_numismatic": true,
  "clarification_needed": false,
  "clarification_question": "",
  "scope_summary": "one sentence describing the interpreted scope",
  "selected_scope": "short label for the recommended scope",
  "group_by": "e.g. decade, mint, ruler, category, or none",
  "scope_options": [
    {"label": "", "description": "", "recommended": true}
  ]
}

Rules:
- If the prompt is not about coin collecting, set "is_numismatic": false and
  "clarification_needed": false.
- Only set "clarification_needed": true when you cannot propose even one
  concrete, boundable scope option — e.g. the prompt is purely subjective
  ("cool coins") or gives no usable subject at all. Provide a specific
  "clarification_question" in that case.
- An unbounded-sounding prompt ("all Roman coins") is NOT automatically a
  clarification case — if you can decompose it into distinct, boundable scope
  options (e.g. by emperor, by denomination, by era), list them in
  "scope_options" and pick the most reasonable default as "selected_scope".
- Never fabricate a roster here — this step only analyzes intent and scope.
- Do not use emojis.""")

ROSTER_PROMPT = with_safety("""You are the Roster Researcher for a numismatic set-builder workflow.
Given the interpreted subject and scope, enumerate the complete candidate
roster of slots for the set. Respond with a single JSON array only:

[
  {
    "label": "short slot label, e.g. '1943 Lincoln Wheat Cent'",
    "criteria": {"year": "1943", "mint": "", "denomination": "Cent"},
    "group": "grouping value, e.g. decade or category label",
    "sort_order": 0,
    "source_note": "basis for this entry, or a note about uncertainty"
  }
]

Rules:
- Enumerate every slot the scope requires, up to the requested slot cap.
- Use accurate numismatic facts. If you are not confident about a specific
  detail (a catalog number, a rare mint mark), say so plainly in
  "source_note" rather than inventing a confident-sounding fact.
- Do not include any set, slot, or coin creation instructions — this is
  data only.
- Do not use emojis.""")

MATCH_PROMPT = with_safety("""You are the Collection Matcher for a numismatic set-builder workflow.
Given a proposed roster and a summary of the user's existing collection,
estimate how many roster slots the user's collection is likely to already
fill. Respond with a single JSON object only:

{"estimated_filled": 0, "estimated_total": 0, "notes": ""}

Rules:
- Base your estimate only on the collection summary provided — never assume
  coins the user hasn't been shown to own.
- "estimated_total" should equal the number of roster slots provided.
- Keep "notes" brief and factual.
- Do not use emojis.""")

VALIDATE_PROMPT = with_safety("""You are the Validator/Critic for a numismatic set-builder workflow.
Review the proposed roster for completeness and accuracy. Respond with a
single JSON array only, echoing every input slot with two added fields:

[
  {
    "label": "",
    "criteria": {},
    "group": "",
    "sort_order": 0,
    "source_note": "",
    "verification_status": "verified",
    "validation_notes": ""
  }
]

Rules:
- Every slot from the input roster MUST appear in your output — never
  silently drop a slot.
- Mark "verification_status": "unverified" for any slot whose criteria you
  cannot confidently confirm (e.g. an unverifiable catalog reference or a
  rare variety), and explain why in "validation_notes".
- Only mark "verified" when you are confident the slot's criteria are
  numismatically accurate.
- Do not add new slots that were not in the input roster.
- Do not use emojis.""")


class SetBuilderState(TypedDict, total=False):
    run_id: int | None
    prompt: str
    feedback: str
    collection_summary: str
    max_turns: int
    max_slots: int
    enable_external_lookup: bool
    turns_used: int
    transcript: list[str]
    intent: dict[str, Any]
    roster: list[dict[str, Any]]
    match: dict[str, Any]
    validated_slots: list[dict[str, Any]]
    status: str
    clarification_question: str
    failure_reason: str


def _record(state: SetBuilderState, note: str) -> list[str]:
    return [*state.get("transcript", []), note]


async def _call_json(model, system_prompt: str, human_text: str) -> Any:
    """Invoke the model and parse a JSON payload from its response."""
    messages = [SystemMessage(content=system_prompt), HumanMessage(content=human_text)]
    response = await ainvoke_with_retry(model, messages)
    content = response.content if isinstance(response.content, str) else str(response.content)
    payload = extract_json_payload(content)
    return json.loads(payload)


def _collection_summary_text(collection: dict[str, Any] | None) -> str:
    if not collection:
        return "The user's collection summary was not provided."
    total = collection.get("total_coins", 0)
    categories = collection.get("categories") or {}
    top_coins = collection.get("top_coins") or []
    lines = [f"Total owned coins: {total}."]
    if categories:
        lines.append("Categories: " + ", ".join(f"{k}={v}" for k, v in categories.items()))
    if top_coins:
        names = [c.get("name", "") for c in top_coins[:25] if isinstance(c, dict)]
        lines.append("Sample owned coin names: " + "; ".join(n for n in names if n))
    return " ".join(lines)


def _route_after_intent(state: SetBuilderState) -> str:
    if state.get("status"):
        return "finalize"
    if state.get("turns_used", 0) >= state.get("max_turns", 4):
        return "finalize"
    return "roster_researcher"


def _route_after_roster(state: SetBuilderState) -> str:
    if state.get("status"):
        return "finalize"
    if state.get("turns_used", 0) >= state.get("max_turns", 4):
        return "finalize"
    return "collection_matcher"


def _route_after_match(state: SetBuilderState) -> str:
    if state.get("status"):
        return "finalize"
    if state.get("turns_used", 0) >= state.get("max_turns", 4):
        return "finalize"
    return "validator"


def create_set_builder_team(
    llm_config: LLMConfig,
    enable_external_lookup: bool = True,
):
    """Create the set-builder workflow graph.

    Uses `get_search_model`/`create_search_agent`-capable models for roster
    research when external lookups are enabled, and a plain chat model for
    the other roles. Kept as free functions (rather than closures over test
    doubles) so tests can monkeypatch `app.llm.provider.get_chat_model` /
    `get_search_model` without needing network access.
    """
    plain_model = get_chat_model(llm_config)
    research_model = get_search_model(llm_config) if enable_external_lookup else plain_model

    async def intent_node(state: SetBuilderState) -> dict:
        logger.info(
            "[set_builder] stage=intent_analyst run_id=%s provider=%s model=%s prompt=%.120s max_slots=%s",
            state.get("run_id"), llm_config.provider, llm_config.model, state.get("prompt", ""), state.get("max_slots"),
        )
        turns_used = state.get("turns_used", 0) + 1
        human_text = f"User prompt: {state.get('prompt', '')}"
        if state.get("feedback"):
            human_text += f"\n\nRegeneration feedback from a previous proposal: {state['feedback']}"
        try:
            intent = await _call_json(plain_model, INTENT_PROMPT, human_text)
        except Exception:
            logger.exception("[set_builder] intent_analyst failed")
            return {
                "status": "failed",
                "failure_reason": "Intent analysis failed to produce a valid response.",
                "turns_used": turns_used,
                "transcript": _record(state, "Intent analysis failed."),
            }

        update: dict[str, Any] = {
            "intent": intent,
            "turns_used": turns_used,
            "transcript": _record(state, f"Intent analysis: {intent.get('subject', 'unknown subject')}"),
        }
        if not intent.get("is_numismatic", True):
            update["status"] = "rejected"
            update["failure_reason"] = "The request does not describe a numismatic (coin collecting) set."
        elif intent.get("clarification_needed"):
            update["status"] = "clarification_needed"
            update["clarification_question"] = intent.get("clarification_question") or (
                "Could you clarify the specific coins or scope you'd like this set to cover?"
            )
        return update

    async def roster_node(state: SetBuilderState) -> dict:
        logger.info(
            "[set_builder] stage=roster_researcher run_id=%s provider=%s model=%s max_slots=%s",
            state.get("run_id"), llm_config.provider, llm_config.model, state.get("max_slots"),
        )
        turns_used = state.get("turns_used", 0) + 1
        intent = state.get("intent", {})
        human_text = (
            f"Subject: {intent.get('subject', '')}\n"
            f"Selected scope: {intent.get('selected_scope', '')}\n"
            f"Scope summary: {intent.get('scope_summary', '')}\n"
            f"Group by: {intent.get('group_by', 'none')}\n"
            f"Maximum roster entries: {state.get('max_slots', 200)}"
        )
        try:
            roster = await _call_json(research_model, ROSTER_PROMPT, human_text)
            if not isinstance(roster, list):
                raise ValueError("roster researcher did not return a JSON array")
        except Exception:
            logger.exception("[set_builder] roster_researcher failed")
            return {
                "status": "failed",
                "failure_reason": "Roster research failed to produce a valid roster.",
                "turns_used": turns_used,
                "transcript": _record(state, "Roster research failed."),
            }
        roster = [item for item in roster if isinstance(item, dict)][: state.get("max_slots", 200)]
        return {
            "roster": roster,
            "turns_used": turns_used,
            "transcript": _record(state, f"Roster research produced {len(roster)} candidate slots."),
        }

    async def match_node(state: SetBuilderState) -> dict:
        logger.info(
            "[set_builder] stage=collection_matcher run_id=%s provider=%s model=%s",
            state.get("run_id"), llm_config.provider, llm_config.model,
        )
        turns_used = state.get("turns_used", 0) + 1
        roster = state.get("roster", [])
        labels = [item.get("label", "") for item in roster]
        human_text = (
            f"Roster slot labels ({len(labels)} total): {json.dumps(labels)}\n\n"
            f"Collection summary: {state.get('collection_summary', '')}"
        )
        try:
            match = await _call_json(plain_model, MATCH_PROMPT, human_text)
        except Exception:
            logger.exception("[set_builder] collection_matcher failed — continuing with a zeroed estimate")
            match = {"estimated_filled": 0, "estimated_total": len(roster), "notes": "Matching estimate unavailable."}
        return {
            "match": match,
            "turns_used": turns_used,
            "transcript": _record(state, "Collection matching estimated filled/total slots."),
        }

    async def validate_node(state: SetBuilderState) -> dict:
        logger.info(
            "[set_builder] stage=validator run_id=%s provider=%s model=%s slots=%d",
            state.get("run_id"), llm_config.provider, llm_config.model, len(state.get("roster", [])),
        )
        turns_used = state.get("turns_used", 0) + 1
        roster = state.get("roster", [])
        human_text = f"Roster to validate: {json.dumps(roster)}"
        try:
            validated = await _call_json(plain_model, VALIDATE_PROMPT, human_text)
            if not isinstance(validated, list):
                raise ValueError("validator did not return a JSON array")
            validated = [item for item in validated if isinstance(item, dict)]
        except Exception:
            logger.exception("[set_builder] validator failed — marking roster unverified")
            validated = [
                {**item, "verification_status": "unverified", "validation_notes": "Validator step failed; unconfirmed."}
                for item in roster
            ]
        return {
            "validated_slots": validated,
            "turns_used": turns_used,
            "transcript": _record(state, f"Validator annotated {len(validated)} slots."),
        }

    async def finalize_node(state: SetBuilderState) -> dict:
        logger.info(
            "[set_builder] stage=orchestrator_finalize run_id=%s status=%s turns_used=%s max_slots=%s",
            state.get("run_id"), state.get("status"), state.get("turns_used"), state.get("max_slots"),
        )
        status = state.get("status")
        if status:
            return {"status": status}

        validated_slots = state.get("validated_slots")
        if not validated_slots:
            return {
                "status": "limit_reached",
                "failure_reason": state.get("failure_reason")
                or "Turn or budget limit reached before the roster could be validated.",
            }
        return {"status": "completed"}

    graph = StateGraph(SetBuilderState)
    graph.add_node("intent_analyst", intent_node)
    graph.add_node("roster_researcher", roster_node)
    graph.add_node("collection_matcher", match_node)
    graph.add_node("validator", validate_node)
    graph.add_node("finalize", finalize_node)

    graph.set_entry_point("intent_analyst")
    graph.add_conditional_edges(
        "intent_analyst", _route_after_intent, {"finalize": "finalize", "roster_researcher": "roster_researcher"}
    )
    graph.add_conditional_edges(
        "roster_researcher", _route_after_roster, {"finalize": "finalize", "collection_matcher": "collection_matcher"}
    )
    graph.add_conditional_edges(
        "collection_matcher", _route_after_match, {"finalize": "finalize", "validator": "validator"}
    )
    graph.add_edge("validator", "finalize")
    graph.add_edge("finalize", END)

    return graph.compile()


def _build_proposal(state: SetBuilderState, max_slots: int) -> SetBuilderProposal:
    intent = state.get("intent", {})
    match = state.get("match", {})
    validated_slots = state.get("validated_slots", [])

    truncated = len(validated_slots) > max_slots
    slots_data = validated_slots[:max_slots]
    slots: list[SetBuilderSlot] = []
    for order, item in enumerate(slots_data):
        criteria = item.get("criteria")
        raw_sort_order = item.get("sort_order", order)
        sort_order = int(raw_sort_order) if str(raw_sort_order).lstrip("-").isdigit() else order
        slots.append(
            SetBuilderSlot(
                label=str(item.get("label") or f"Slot {order + 1}"),
                criteria={str(k): str(v) for k, v in criteria.items()} if isinstance(criteria, dict) else {},
                group=str(item.get("group") or ""),
                sort_order=sort_order,
                verification_status="verified" if item.get("verification_status") == "verified" else "unverified",
                source_note=str(item.get("source_note") or ""),
                validation_notes=str(item.get("validation_notes") or ""),
            )
        )

    scope_options = [
        SetBuilderScopeOption(
            label=str(opt.get("label") or ""),
            description=str(opt.get("description") or ""),
            estimated_slot_count=len(slots) if opt.get("recommended") else 0,
            recommended=bool(opt.get("recommended")),
        )
        for opt in intent.get("scope_options", [])
        if isinstance(opt, dict) and opt.get("label")
    ]

    prematch_notes = str(match.get("notes") or "")
    if truncated:
        prematch_notes = (
            f"{prematch_notes} Roster truncated to the {max_slots}-slot cap; consider narrowing scope."
        ).strip()

    return SetBuilderProposal(
        name=str(intent.get("subject") or "Proposed Set")[:300] or "Proposed Set",
        slug_hint="",
        description=str(intent.get("scope_summary") or ""),
        scope_summary=str(intent.get("scope_summary") or ""),
        selected_scope=str(intent.get("selected_scope") or ""),
        group_by=str(intent.get("group_by") or ""),
        scope_options=scope_options,
        slots=slots,
        prematch_summary=SetBuilderPrematchSummary(
            estimated_filled=max(0, int(match.get("estimated_filled", 0) or 0)),
            estimated_total=max(0, int(match.get("estimated_total", len(slots)) or len(slots))),
            notes=prematch_notes,
        ),
    )


async def run_set_builder_workflow(request: SetBuilderRequest) -> SetBuilderResponse:
    """Run the set-builder workflow and return a structured proposal (or a
    structured clarification/rejection/failure). Never creates or modifies a
    set — data only, per FR-003.
    """
    collection_dict: dict[str, Any] | None = None
    if isinstance(request.collection, PortfolioSummary):
        collection_dict = request.collection.model_dump()

    initial_state: SetBuilderState = {
        "run_id": request.run_id,
        "prompt": request.prompt,
        "feedback": request.feedback,
        "collection_summary": _collection_summary_text(collection_dict),
        "max_turns": request.max_turns,
        "max_slots": request.max_slots,
        "enable_external_lookup": request.enable_external_lookup,
        "turns_used": 0,
        "transcript": [],
    }

    try:
        graph = create_set_builder_team(request.llm, enable_external_lookup=request.enable_external_lookup)
        result = await graph.ainvoke(initial_state)
    except Exception:
        logger.exception("[set_builder] workflow execution failed")
        return SetBuilderResponse(
            status="failed",
            failure_reason="The set-builder workflow could not complete. Please try again.",
            transcript_summary="",
            turns_used=initial_state.get("turns_used", 0),
        )

    status = result.get("status", "failed")
    transcript_summary = " | ".join(result.get("transcript", []))
    turns_used = result.get("turns_used", 0)

    if status == "completed":
        proposal = _build_proposal(result, request.max_slots)
        return SetBuilderResponse(
            status="completed",
            proposal=proposal,
            transcript_summary=transcript_summary,
            turns_used=turns_used,
        )

    return SetBuilderResponse(
        status=status,
        clarification_question=result.get("clarification_question", ""),
        failure_reason=result.get("failure_reason", ""),
        transcript_summary=transcript_summary,
        turns_used=turns_used,
    )
