"""Deep-identification pipeline graph (T067/T069).

`prepare_evidence` (vision) -> `router` -> bounded provider fan-out ->
`evaluator` -> `synthesizer` (contracts/agent-internal-contract.md §6).

Two entry points are exposed:

- `build_graph()` compiles a `StateGraph` using the exact same node
  callables the streaming runner uses, so graph *topology* (node names,
  edges, recursion-limit wiring) is independently testable.
- `run_deep_identification_stream(request)` is the actual production
  driver. It is a hand-written async generator (not `graph.astream`)
  because the internal SSE envelope (contract §3) needs a
  `provider_started`/`provider_result` frame *per provider* as fan-out
  proceeds — a single `provider_fanout` graph node only yields once it has
  fully completed, which cannot express that per-provider progress. Both
  entry points call the identical node functions below, so there is no
  behavioral drift between the tested topology and the runtime driver.
"""

import asyncio
import logging
from collections.abc import AsyncGenerator

from langgraph.graph import END, StateGraph

from app.config import settings
from app.llm.provider import get_chat_model
from app.models.hypothesis import CoinHypothesis
from app.models.requests import DeepIdentifyBounds, DeepIdentifyImage, DeepIdentifyRequest, LLMConfig, QuickEvidence
from app.models.responses import ProviderEvidence
from app.streaming import format_sse, sanitize_user_facing_payload
from app.teams.deep_identification.evaluator import evaluate
from app.teams.deep_identification.hypothesis import (
    build_hypothesis_from_quick_evidence,
    build_hypothesis_from_vision_traced,
)
from app.teams.deep_identification.providers import ngc as ngc_provider
from app.teams.deep_identification.providers import nomisma as nomisma_provider
from app.teams.deep_identification.providers import numista as numista_provider
from app.teams.deep_identification.providers import ocre as ocre_provider
from app.teams.deep_identification.providers import rpc as rpc_provider
from app.teams.deep_identification.query_terms import build_query_terms
from app.teams.deep_identification.router import route
from app.teams.deep_identification.state import DeepIdentificationState
from app.teams.deep_identification.synthesis import synthesize
from app.tools.provider_tools import ProviderToolsClient

logger = logging.getLogger(__name__)

_AUTOMATED_PROVIDER_NODES = {
    "numista": numista_provider.run,
    "nomisma": nomisma_provider.run,
    "ocre": ocre_provider.run,
}
_TRIVIAL_PROVIDER_NODES = {"rpc": rpc_provider.run}


async def prepare_evidence_node(state: DeepIdentificationState, llm_config: LLMConfig | None = None) -> dict:
    """Vision node: obverse+reverse only (FR-004 — hint images never enter
    the vision-prompt slots, they are context-only for provider queries).

    Builds the typed `hypothesis` output (contracts/vision-hypothesis.md
    §1) from the SAME single vision LLM call this node has always made —
    it no longer also produces a separate free-prose `image_analysis`
    string (that write-only field is deleted, see state.py). When
    `llm_config` is supplied, `build_hypothesis_from_vision` runs the real
    structured vision call with its full degrade ladder; when it is not
    (e.g. the topology-only `build_graph` compiled-but-never-invoked path),
    this falls back to the deterministic `quick_evidence` adapter with no
    LLM call at all.
    """
    from app.teams.coin_analysis import _build_image_contents

    quick_evidence = state.get("quick_evidence")
    images: list[DeepIdentifyImage] = state.get("images", [])
    face_images = [img.data_uri for img in images if img.role in ("obverse", "reverse")]
    image_contents = _build_image_contents(face_images)

    if llm_config is None:
        hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)
        source = "deterministic_fallback"
    else:
        hypothesis, source = await build_hypothesis_from_vision_traced(llm_config, image_contents, quick_evidence)
    return {"hypothesis": hypothesis, "hypothesis_source": source}



async def router_node(state: DeepIdentificationState) -> dict:
    decision = route(
        state.get("catalog", []),
        state.get("provider_override", []),
        state.get("bounds"),
        state.get("quick_evidence"),
        state.get("hypothesis"),
    )
    return {"selected": decision.selected, "skipped": decision.skipped, "router_rationale": decision.rationale}


# Providers with no free-text query — either a structured signal decode
# (ocre) or no automated call at all (ngc/rpc, terms-of-use/no-public-api).
# Kept as fixed, non-user-derived strings so `provider_started` still says
# *something* useful (FR-040) without duplicating each provider node's own
# field-decoding logic here.
_PROVIDER_STATIC_STARTED_NOTES = {
    "ocre": "Querying with decoded coin-type signals (ruler/denomination/mint/type id).",
    "ngc": "Link-out only; NGC terms of use prohibit automated queries.",
    "rpc": "Not automated; no public API is available.",
}

# Defensive bound on any query text placed on a provider_started frame —
# `build_query_terms` already bounds its own tiers, but this is a second,
# independent ceiling so a future upstream change can never make this
# frame's payload unbounded (spec FR-040 binding limit).
_PROVIDER_QUERY_STARTED_MAX_LENGTH = 300

# Nomisma's Go client rejects anything over 200 runes
# (nomisma_client.go::nomismaMaxQueryLength) — mirrors providers/nomisma.py's
# own `_MAX_QUERY_LENGTH` so the previewed query always matches what the
# provider node will actually send.
_NOMISMA_QUERY_MAX_LENGTH = 200


def _provider_started_detail(
    name: str,
    quick_evidence: QuickEvidence | None,
    notes: str,
    hypothesis: CoinHypothesis | None,
) -> dict:
    """Bounded, application-authored detail added to a `provider_started`
    frame (FR-040): the exact deterministic query text the provider node is
    about to use for numista/nomisma (the only two providers that build a
    free-text query via the shared `build_query_terms`), or a fixed
    descriptive note for providers with no free-text query. Never invents
    anything — every value here is the same input the corresponding
    provider node itself is about to consume.
    """
    if name == "numista":
        query = build_query_terms(quick_evidence, hypothesis, notes)
    elif name == "nomisma":
        query = build_query_terms(quick_evidence, hypothesis, notes, max_length=_NOMISMA_QUERY_MAX_LENGTH)
    else:
        query = ""

    if query:
        return {"query_terms": query[:_PROVIDER_QUERY_STARTED_MAX_LENGTH]}
    if name in ("numista", "nomisma"):
        # No precedence tier yielded usable terms — the provider node itself
        # will make zero upstream calls and report this same reason as its
        # error_kind (FR-011). Surface it here too so the owner sees *why*
        # a provider produced nothing, not just that it did.
        return {"skip_reason": "insufficient_query_evidence"}
    static_note = _PROVIDER_STATIC_STARTED_NOTES.get(name)
    return {"detail": static_note} if static_note else {}


async def _run_one_provider(
    name: str,
    catalog_by_name: dict,
    tools: ProviderToolsClient | None,
    quick_evidence: QuickEvidence | None,
    notes: str,
    bounds: DeepIdentifyBounds,
    semaphore: asyncio.Semaphore,
    hypothesis: CoinHypothesis | None = None,
) -> ProviderEvidence:
    entry = catalog_by_name.get(name)
    if entry is None:
        return ProviderEvidence(provider=name, status="skipped", automatable=name in _AUTOMATED_PROVIDER_NODES)

    if name == "ngc":
        return ngc_provider.run(entry, quick_evidence)
    if name in _TRIVIAL_PROVIDER_NODES:
        return _TRIVIAL_PROVIDER_NODES[name](entry)

    fn = _AUTOMATED_PROVIDER_NODES.get(name)
    if fn is None:
        return ProviderEvidence(
            provider=name, status="failed", automatable=True, error_kind="unconfigured", call_count=0
        )
    # An automatable provider needs the tools client to reach upstream; a
    # disabled one (e.g. OCRE with its flag off, automatable=False) still runs
    # its node purely so it can emit a zero-call not_automated row (FR-016).
    if tools is None and entry.automatable:
        return ProviderEvidence(
            provider=name, status="failed", automatable=True, error_kind="unconfigured", call_count=0
        )

    async with semaphore:
        try:
            return await asyncio.wait_for(
                fn(entry, tools, quick_evidence, notes, hypothesis), timeout=bounds.provider_timeout_s
            )
        except TimeoutError:
            return ProviderEvidence(
                provider=name, status="timed_out", automatable=True, error_kind="timeout", call_count=1
            )
        except Exception:
            logger.exception("[deep_identification.graph] provider node %s raised unexpectedly", name)
            return ProviderEvidence(
                provider=name, status="failed", automatable=True, error_kind="upstream", call_count=1
            )


async def provider_fanout_node(
    state: DeepIdentificationState,
    tools: ProviderToolsClient | None,
    on_provider_event=None,
    on_result=None,
) -> dict:
    """Bounded fan-out over selected automatable providers plus every
    non-automatable catalog entry (which always runs, trivially).
    `asyncio.gather(..., return_exceptions=True)` guards against any node
    raising despite its own internal try/except (defense in depth).

    `on_result`, when supplied, is invoked synchronously with each
    `ProviderEvidence` row as soon as its task completes — *before*
    `asyncio.gather` has finished waiting for every provider. This lets the
    streaming driver accumulate partial evidence into `state["evidence"]`
    incrementally, so a total-timeout expiry mid-fanout (T069) still has
    every already-completed provider's evidence available for partial
    synthesis, even though one slow/hung provider is still pending.
    """
    catalog = state.get("catalog", [])
    catalog_by_name = {entry.provider: entry for entry in catalog}
    bounds: DeepIdentifyBounds = state["bounds"]
    quick_evidence = state.get("quick_evidence")
    notes = state.get("notes", "")
    semaphore = asyncio.Semaphore(max(1, bounds.max_concurrency))

    selected = list(state.get("selected", []))
    non_automatable = [entry.provider for entry in catalog if not entry.automatable]
    names_to_run = selected + non_automatable

    async def run_and_report(name: str):
        if on_provider_event:
            # FR-040: query terms ride on `provider_started` (already a
            # first-class event type the frontend groups live), not a bare
            # progress phase — Aurelia's stated preference, and it survives
            # to the owner-scoped stream verbatim (deep_identification_
            # pipeline_runner.go's onFrame has no reducing case for
            # provider_started, unlike provider_result).
            started_frame = {"type": "provider_started", "provider": name}
            started_frame.update(_provider_started_detail(name, quick_evidence, notes, state.get("hypothesis")))
            await on_provider_event(started_frame)
        result = await _run_one_provider(
            name, catalog_by_name, tools, quick_evidence, notes, bounds, semaphore, state.get("hypothesis")
        )
        if on_result:
            on_result(result)
        if on_provider_event:
            # Emit the full, application-owned ProviderEvidence (contract
            # §3/§4): the runner needs the typed, citation-backed `claims`
            # here to resolve the terminal synthesis' evidence_refs into
            # citation-bearing proposal evidence. Serialize via the Pydantic
            # model (never an ad-hoc dict) so field names/types/nullability
            # match the Go mirror exactly; every field is length-bounded by
            # the model and carries no raw provider payload, user notes, or
            # image data. `_emit`'s sanitizer still redacts any token-shaped
            # string defense-in-depth without stripping valid claims.
            await on_provider_event({"type": "provider_result", **result.model_dump(mode="json")})
        return result

    tasks = [run_and_report(name) for name in names_to_run]
    raw_results = await asyncio.gather(*tasks, return_exceptions=True) if tasks else []

    evidence: list[ProviderEvidence] = []
    for name, result in zip(names_to_run, raw_results, strict=False):
        if isinstance(result, BaseException):
            logger.exception("[deep_identification.graph] provider %s fan-out task failed", name, exc_info=result)
            evidence.append(
                ProviderEvidence(
                    provider=name, status="failed", automatable=name in selected, error_kind="upstream"
                )
            )
        else:
            evidence.append(result)

    for skip in state.get("skipped", []):
        evidence.append(ProviderEvidence(provider=skip["provider"], status="skipped", automatable=True))

    return {"evidence": evidence}


async def evaluator_node(state: DeepIdentificationState, model) -> dict:
    result = await evaluate(model, state.get("evidence", []), hypothesis=state.get("hypothesis"))
    return {"disagreements": result.disagreements, "resolved_count": result.resolved_count}


async def synthesizer_node(state: DeepIdentificationState, model, partial_success: bool = False) -> dict:
    disagreements = state.get("disagreements", [])
    unresolved_questions = [f"Sources disagree on {d.field}." for d in disagreements]
    synthesis = await synthesize(
        model,
        state.get("evidence", []),
        disagreements,
        unresolved_questions,
        partial_success,
        hypothesis=state.get("hypothesis"),
    )
    return {"synthesis": synthesis.model_dump()}


def build_graph(model, tools: ProviderToolsClient | None, recursion_limit: int | None = None):
    """Compile a topology-testable `StateGraph` using the same node
    callables the streaming runner uses. Not used at request-serving time
    (see module docstring) — this exists so graph shape is independently
    verifiable without needing to drive the fine-grained SSE frames.

    When `recursion_limit` is provided (contract §6, `bounds.recursion_limit`),
    it is bound into the compiled graph's invocation config so any
    `.ainvoke`/`.astream` on the returned graph is capped at that iteration
    bound. The production SSE driver is a hand-written generator that does not
    invoke the compiled graph, so this binding governs graph-based callers and
    the topology tests rather than the streaming path.
    """

    async def _prepare(state: DeepIdentificationState) -> dict:
        return await prepare_evidence_node(state)

    async def _route(state: DeepIdentificationState) -> dict:
        return await router_node(state)

    async def _fanout(state: DeepIdentificationState) -> dict:
        return await provider_fanout_node(state, tools)

    async def _evaluate(state: DeepIdentificationState) -> dict:
        return await evaluator_node(state, model)

    async def _synthesize(state: DeepIdentificationState) -> dict:
        return await synthesizer_node(state, model)

    graph = StateGraph(DeepIdentificationState)
    graph.add_node("prepare_evidence", _prepare)
    graph.add_node("router", _route)
    graph.add_node("provider_fanout", _fanout)
    graph.add_node("evaluator", _evaluate)
    graph.add_node("synthesizer", _synthesize)

    graph.set_entry_point("prepare_evidence")
    graph.add_edge("prepare_evidence", "router")
    graph.add_edge("router", "provider_fanout")
    graph.add_edge("provider_fanout", "evaluator")
    graph.add_edge("evaluator", "synthesizer")
    graph.add_edge("synthesizer", END)

    compiled = graph.compile()
    if recursion_limit is not None:
        return compiled.with_config({"recursion_limit": recursion_limit})
    return compiled


def _classify_pipeline_error(exc: BaseException) -> str:
    """Map an unexpected pipeline exception to a typed contract §3 `error`
    frame code (`llm_unavailable` | `timeout` | `invalid_model_output` |
    `internal`). Deliberately narrow: only well-understood model-output
    parse failures and provider/model connectivity failures get a specific
    code; everything else stays `internal` rather than guessing.
    """
    from pydantic import ValidationError

    try:
        from langchain_core.exceptions import OutputParserException
    except Exception:  # pragma: no cover - langchain always present in practice
        parser_exc: tuple[type[BaseException], ...] = ()
    else:
        parser_exc = (OutputParserException,)

    if isinstance(exc, (ValidationError, *parser_exc)):
        return "invalid_model_output"
    name = type(exc).__name__
    if isinstance(exc, ConnectionError) or "Connection" in name or "Unavailable" in name:
        return "llm_unavailable"
    return "internal"


def _emit(frame: dict) -> str:
    """Redact any JWT/Bearer-shaped strings from an outgoing frame before
    it is serialized — defense in depth alongside the job-scoped token
    never being placed in a claim/citation/narrative field in the first
    place (Constitution §21 — no secrets in logs/user-facing output).
    """
    return format_sse(sanitize_user_facing_payload(frame))


def _vision_completed_message(hypothesis: CoinHypothesis, source: str) -> str:
    """FR-040 `vision_completed` progress message: structural facts only
    (populated-field count, a confidence bucket derived from those fields'
    own bounded `[0,1]` scores) plus an honest degradation note when the
    structured vision call did not produce the result. Brian's core
    complaint was a silent nothing — a step that produced nothing must say
    so and why, not just move on to the next phase.
    """
    fields = hypothesis.fields()
    field_count = len(fields)
    if field_count:
        avg_confidence = sum(field.confidence for field in fields.values()) / field_count
        if avg_confidence >= 0.65:
            bucket = "high confidence"
        elif avg_confidence >= 0.45:
            bucket = "medium confidence"
        else:
            bucket = "low confidence"
        plural = "" if field_count == 1 else "s"
        base = f"Vision analysis produced {field_count} populated field{plural} ({bucket})."
    else:
        base = "Vision analysis produced no populated fields."

    if source == "no_images":
        return f"{base} No obverse/reverse images were available."
    if source == "deterministic_fallback":
        return (
            f"{base} The structured vision call did not produce a usable result; "
            "used deterministic quick-evidence data instead."
        )
    if source == "prose":
        return f"{base} The structured vision call failed schema validation; recovered from unstructured model output."
    return base


def _synthesis_started_message(evidence: list[ProviderEvidence], hypothesis: CoinHypothesis | None) -> str:
    """FR-040 `synthesis_started` detail: counts and structural facts only
    (contributing-provider count, whether image evidence also feeds
    synthesis) — never claim content itself.
    """
    contributing = sum(1 for row in evidence if row.status == "contributed")
    plural = "" if contributing == 1 else "s"
    message = f"Synthesizing report from {contributing} contributing source{plural}"
    if hypothesis is not None and not hypothesis.is_empty():
        message += " and image evidence."
    else:
        message += "."
    return message


def _clamp_bounds_to_ceilings(bounds: DeepIdentifyBounds) -> DeepIdentifyBounds:
    """Clamp Go-supplied per-run bounds to the deployment's configured
    `AGENT_DEEP_*` ceilings (T077). Callers must never be able to exceed
    the service-level maximums regardless of what `request.bounds`
    claims, so every field is `min(request_value, setting_value)`.
    """
    return DeepIdentifyBounds(
        max_providers=min(bounds.max_providers, settings.deep_max_providers),
        max_concurrency=min(bounds.max_concurrency, settings.deep_max_concurrency),
        provider_timeout_s=min(bounds.provider_timeout_s, settings.deep_provider_timeout),
        total_timeout_s=min(bounds.total_timeout_s, settings.deep_total_timeout),
        recursion_limit=min(bounds.recursion_limit, settings.deep_recursion_limit),
    )


async def run_deep_identification_stream(request: DeepIdentifyRequest) -> AsyncGenerator[str, None]:
    """Production streaming driver — the sole caller from `routes.py`
    (T068). Emits the internal envelope frame types from contract §3.
    Total-timeout partial-synthesis fallback (T069) lives here: on
    expiry, whatever evidence has been gathered so far is synthesized
    with `partial_success: true`; if nothing was gathered at all, a typed
    `error` frame is emitted instead.
    """
    bounds = _clamp_bounds_to_ceilings(request.bounds)
    model = get_chat_model(request.llm)
    tools = (
        ProviderToolsClient(request.tools_base_url, request.internal_token, timeout_s=bounds.provider_timeout_s)
        if request.tools_base_url and request.internal_token
        else None
    )

    state: DeepIdentificationState = {
        "job_id": request.job_id,
        "images": request.images,
        "notes": request.notes,
        "quick_evidence": request.quick_evidence,
        "catalog": request.provider_catalog,
        "provider_override": request.provider_override,
        "bounds": bounds,
        "evidence": [],
    }

    queue: asyncio.Queue = asyncio.Queue()
    SENTINEL = object()

    async def on_provider_event(frame: dict) -> None:
        await queue.put(frame)

    def on_result(row: ProviderEvidence) -> None:
        # Incremental accumulation so a total-timeout expiry mid-fanout
        # (T069) still has every already-completed provider's evidence.
        state["evidence"] = [*state.get("evidence", []), row]

    async def pipeline() -> dict:
        image_result = await prepare_evidence_node(state, request.llm)
        state.update(image_result)
        await queue.put({"type": "progress", "stage": "image_evidence_ready"})
        await queue.put({
            "type": "progress",
            "stage": "vision_completed",
            "message": _vision_completed_message(
                image_result["hypothesis"], image_result.get("hypothesis_source", "")
            ),
        })

        router_result = await router_node(state)
        state.update(router_result)
        await queue.put({
            "type": "router_selected",
            "selected": router_result["selected"],
            "skipped": router_result["skipped"],
            "rationale": router_result["router_rationale"],
        })

        await queue.put({"type": "progress", "stage": "provider_fanout_started"})
        state["evidence"] = []  # reset before incremental on_result accumulation begins
        fanout_result = await provider_fanout_node(
            state, tools, on_provider_event=on_provider_event, on_result=on_result
        )
        state["evidence"] = fanout_result["evidence"]  # canonical final list (includes skipped rows)

        await queue.put({"type": "progress", "stage": "evaluation_started"})
        evaluator_result = await evaluator_node(state, model)
        state.update(evaluator_result)
        await queue.put({
            "type": "evaluation",
            "disagreement_count": len(evaluator_result["disagreements"]),
            "resolved_count": evaluator_result["resolved_count"],
        })

        await queue.put({
            "type": "synthesis_started",
            "message": _synthesis_started_message(state.get("evidence", []), state.get("hypothesis")),
        })
        synthesizer_result = await synthesizer_node(state, model)
        state.update(synthesizer_result)
        return synthesizer_result["synthesis"]

    pipeline_task = asyncio.ensure_future(pipeline())

    async def drain_queue() -> AsyncGenerator[dict, None]:
        while True:
            item = await queue.get()
            if item is SENTINEL:
                return
            yield item

    async def watch_pipeline() -> None:
        try:
            await pipeline_task
        finally:
            await queue.put(SENTINEL)

    watcher = asyncio.ensure_future(watch_pipeline())

    try:
        async with asyncio.timeout(bounds.total_timeout_s):
            async for frame in drain_queue():
                yield _emit(frame)
            synthesis = pipeline_task.result()
            yield _emit({"type": "synthesis", "report": synthesis})
    except TimeoutError:
        pipeline_task.cancel()
        watcher.cancel()
        evidence = state.get("evidence", [])
        if not evidence:
            yield _emit({
                "type": "error",
                "code": "timeout",
                "message": "Deep analysis timed out before any evidence was gathered.",
            })
            return
        try:
            disagreements = state.get("disagreements", [])
            unresolved_questions = [f"Sources disagree on {d.field}." for d in disagreements]
            # T056: the timeout partial-synthesis path must thread the same
            # hypothesis the happy path uses — it must not silently fall
            # back to a hypothesis-less synthesis just because the run was
            # cut short. `state["hypothesis"]` is set in `prepare_evidence`,
            # the very first pipeline step, so it is available here even
            # when the timeout struck mid-fanout/evaluation.
            synthesis = await synthesize(
                model,
                evidence,
                disagreements,
                unresolved_questions,
                partial_success=True,
                hypothesis=state.get("hypothesis"),
            )
            yield _emit({"type": "synthesis", "report": synthesis.model_dump()})
        except Exception:
            logger.exception("[deep_identification.graph] partial-synthesis fallback failed after timeout")
            yield _emit({
                "type": "error",
                "code": "timeout",
                "message": "Deep analysis timed out and partial synthesis failed.",
            })
    except Exception as exc:
        logger.exception("[deep_identification.graph] pipeline failed unexpectedly")
        pipeline_task.cancel()
        watcher.cancel()
        yield _emit({
            "type": "error",
            "code": _classify_pipeline_error(exc),
            "message": "Deep analysis failed unexpectedly.",
        })
