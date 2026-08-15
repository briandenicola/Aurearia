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

from app.llm.provider import get_chat_model
from app.models.requests import DeepIdentifyBounds, DeepIdentifyImage, DeepIdentifyRequest, QuickEvidence
from app.models.responses import ProviderEvidence
from app.safety import with_safety
from app.streaming import format_sse, sanitize_user_facing_payload
from app.teams.deep_identification.evaluator import evaluate
from app.teams.deep_identification.providers import ngc as ngc_provider
from app.teams.deep_identification.providers import nomisma as nomisma_provider
from app.teams.deep_identification.providers import numista as numista_provider
from app.teams.deep_identification.providers import ocre as ocre_provider
from app.teams.deep_identification.providers import rpc as rpc_provider
from app.teams.deep_identification.router import route
from app.teams.deep_identification.state import DeepIdentificationState
from app.teams.deep_identification.synthesis import synthesize
from app.tools.provider_tools import ProviderToolsClient

logger = logging.getLogger(__name__)

IMAGE_ANALYSIS_PROMPT = with_safety("""You are a numismatic expert briefly describing a coin image pair
(obverse and reverse) for a research pipeline. In 2-4 sentences, describe
only what is visibly present: portrait/design elements, visible
inscriptions/legends, material/color, and wear. Do not guess a specific
ruler, mint, or date unless legend text makes it unambiguous. Do not use
emojis.""")

_AUTOMATED_PROVIDER_NODES = {"numista": numista_provider.run, "nomisma": nomisma_provider.run}
_TRIVIAL_PROVIDER_NODES = {"ocre": ocre_provider.run, "rpc": rpc_provider.run}


async def prepare_evidence_node(state: DeepIdentificationState, model) -> dict:
    """Vision node: obverse+reverse only (FR-004 — hint images never enter
    the vision-prompt slots, they are context-only for provider queries).
    """
    from app.teams.coin_analysis import _build_image_contents

    images: list[DeepIdentifyImage] = state.get("images", [])
    face_images = [img.data_uri for img in images if img.role in ("obverse", "reverse")]
    image_contents = _build_image_contents(face_images)
    if not image_contents:
        return {"image_analysis": ""}

    from langchain_core.messages import HumanMessage, SystemMessage

    from app.llm.retry import ainvoke_with_retry

    human_content: list[dict] = [{"type": "text", "text": IMAGE_ANALYSIS_PROMPT}, *image_contents]
    try:
        response = await ainvoke_with_retry(
            model, [SystemMessage(content="You are an expert numismatist."), HumanMessage(content=human_content)]
        )
        content = response.content if isinstance(response.content, str) else str(response.content)
    except Exception:
        logger.exception("[deep_identification.graph] image analysis LLM call failed")
        content = ""
    return {"image_analysis": content}


async def router_node(state: DeepIdentificationState, model) -> dict:
    decision = await route(
        model,
        state.get("catalog", []),
        state.get("provider_override", []),
        state.get("bounds").max_providers if state.get("bounds") else 0,
        state.get("notes", ""),
    )
    return {"selected": decision.selected, "skipped": decision.skipped, "router_rationale": decision.rationale}


async def _run_one_provider(
    name: str,
    catalog_by_name: dict,
    tools: ProviderToolsClient | None,
    quick_evidence: QuickEvidence | None,
    notes: str,
    bounds: DeepIdentifyBounds,
    semaphore: asyncio.Semaphore,
) -> ProviderEvidence:
    entry = catalog_by_name.get(name)
    if entry is None:
        return ProviderEvidence(provider=name, status="skipped", automatable=name in _AUTOMATED_PROVIDER_NODES)

    if name == "ngc":
        return ngc_provider.run(entry, quick_evidence)
    if name in _TRIVIAL_PROVIDER_NODES:
        return _TRIVIAL_PROVIDER_NODES[name](entry)

    fn = _AUTOMATED_PROVIDER_NODES.get(name)
    if fn is None or tools is None:
        return ProviderEvidence(
            provider=name, status="failed", automatable=True, error_kind="unconfigured", call_count=0
        )

    async with semaphore:
        try:
            return await asyncio.wait_for(
                fn(entry, tools, quick_evidence, notes), timeout=bounds.provider_timeout_s
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
            await on_provider_event({"type": "provider_started", "provider": name})
        result = await _run_one_provider(name, catalog_by_name, tools, quick_evidence, notes, bounds, semaphore)
        if on_result:
            on_result(result)
        if on_provider_event:
            await on_provider_event({"type": "provider_result", "provider": result.provider, "status": result.status})
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
    result = await evaluate(model, state.get("evidence", []))
    return {"disagreements": result.disagreements, "resolved_count": result.resolved_count}


async def synthesizer_node(state: DeepIdentificationState, model, partial_success: bool = False) -> dict:
    disagreements = state.get("disagreements", [])
    unresolved_questions = [f"Sources disagree on {d.field}." for d in disagreements]
    synthesis = await synthesize(model, state.get("evidence", []), disagreements, unresolved_questions, partial_success)
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
        return await prepare_evidence_node(state, model)

    async def _route(state: DeepIdentificationState) -> dict:
        return await router_node(state, model)

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


async def run_deep_identification_stream(request: DeepIdentifyRequest) -> AsyncGenerator[str, None]:
    """Production streaming driver — the sole caller from `routes.py`
    (T068). Emits the internal envelope frame types from contract §3.
    Total-timeout partial-synthesis fallback (T069) lives here: on
    expiry, whatever evidence has been gathered so far is synthesized
    with `partial_success: true`; if nothing was gathered at all, a typed
    `error` frame is emitted instead.
    """
    bounds = request.bounds
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
        image_result = await prepare_evidence_node(state, model)
        state.update(image_result)
        await queue.put({"type": "progress", "stage": "image_evidence_ready"})

        router_result = await router_node(state, model)
        state.update(router_result)
        await queue.put({
            "type": "router_selected",
            "selected": router_result["selected"],
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

        await queue.put({"type": "synthesis_started"})
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
            synthesis = await synthesize(model, evidence, disagreements, unresolved_questions, partial_success=True)
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
