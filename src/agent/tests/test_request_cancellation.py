"""A caller that gives up must stop agent work (#785)."""

import asyncio
import contextlib
import socket
import threading
import time

import httpx
import pytest
import uvicorn

from app import routes
from app.main import app
from app.request_cancellation import ClientDisconnectedError, cancel_on_disconnect

AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}
LLM = {"provider": "anthropic", "api_key": "k", "model": "m"}


class _FakeRequest:
    def __init__(self, disconnect_after: float | None):
        self._disconnect_after = disconnect_after

        class _URL:
            path = "/api/test"

        self.url = _URL()

    async def receive(self) -> dict:
        if self._disconnect_after is None:
            await asyncio.Event().wait()
        await asyncio.sleep(self._disconnect_after)
        return {"type": "http.disconnect"}


async def test_cancel_on_disconnect_returns_the_result_when_client_stays():
    async def work():
        await asyncio.sleep(0.01)
        return "done"

    assert await cancel_on_disconnect(_FakeRequest(None), work()) == "done"


async def test_cancel_on_disconnect_cancels_in_flight_work():
    cancelled = asyncio.Event()

    async def work():
        try:
            await asyncio.sleep(30)
        except asyncio.CancelledError:
            cancelled.set()
            raise

    with pytest.raises(ClientDisconnectedError):
        await cancel_on_disconnect(_FakeRequest(0.05), work())
    assert cancelled.is_set()


async def test_cancel_on_disconnect_propagates_work_errors():
    async def work():
        raise ValueError("boom")

    with pytest.raises(ValueError, match="boom"):
        await cancel_on_disconnect(_FakeRequest(None), work())


async def test_cancel_on_disconnect_lets_work_finish_when_the_watcher_fails():
    class _BrokenReceive(_FakeRequest):
        async def receive(self) -> dict:
            raise RuntimeError("Unexpected message received")

    async def work():
        await asyncio.sleep(0.05)
        return "done"

    assert await cancel_on_disconnect(_BrokenReceive(None), work()) == "done"


async def test_cancel_on_disconnect_lets_work_finish_when_the_watcher_is_cancelled():
    class _CancelledReceive(_FakeRequest):
        async def receive(self) -> dict:
            raise asyncio.CancelledError

    async def work():
        await asyncio.sleep(0.05)
        return "done"

    assert await cancel_on_disconnect(_CancelledReceive(None), work()) == "done"


async def test_cancel_on_disconnect_awaits_work_when_the_route_is_cancelled():
    started = asyncio.Event()
    finished_cleanup = asyncio.Event()

    async def work():
        started.set()
        try:
            await asyncio.sleep(30)
        finally:
            await asyncio.sleep(0.05)
            finished_cleanup.set()

    route = asyncio.ensure_future(cancel_on_disconnect(_FakeRequest(None), work()))
    await started.wait()
    route.cancel()
    with pytest.raises(asyncio.CancelledError):
        await route
    assert finished_cleanup.is_set()


@contextlib.contextmanager
def _running_server():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
    server = uvicorn.Server(uvicorn.Config(app, host="127.0.0.1", port=port, log_level="warning", lifespan="off"))
    thread = threading.Thread(target=server.run, daemon=True)
    thread.start()
    deadline = time.monotonic() + 5
    while not server.started and time.monotonic() < deadline:
        time.sleep(0.02)
    assert server.started, "uvicorn did not start"
    try:
        yield f"http://127.0.0.1:{port}"
    finally:
        server.should_exit = True
        thread.join(timeout=5)


def _slow_work(started: threading.Event, cancelled: threading.Event, finished: threading.Event):
    async def work(*_args, **_kwargs):
        started.set()
        try:
            await asyncio.sleep(10)
        except asyncio.CancelledError:
            cancelled.set()
            raise
        finished.set()

    return work


def _assert_client_timeout_cancels(base_url: str, path: str, payload: dict, events) -> None:
    started, cancelled, finished = events
    with pytest.raises(httpx.TimeoutException):
        httpx.post(f"{base_url}{path}", json=payload, headers=AUTH_HEADERS, timeout=0.5)
    assert started.is_set(), "agent work never started"
    assert cancelled.wait(3), "agent work kept running after the caller gave up"
    assert not finished.is_set()


def test_comparables_work_stops_when_the_go_caller_times_out(monkeypatch):
    events = (threading.Event(), threading.Event(), threading.Event())
    monkeypatch.setattr(routes, "search_comparables", _slow_work(*events))
    payload = {"llm": LLM, "query": "Trajan denarius", "dealer_search_sources": ["vcoins.com"]}
    with _running_server() as base_url:
        _assert_client_timeout_cancels(base_url, "/api/search/comparables", payload, events)


def test_analyze_work_stops_despite_the_route_fallback_handler(monkeypatch):
    events = (threading.Event(), threading.Event(), threading.Event())
    slow = _slow_work(*events)

    class SlowGraph:
        async def ainvoke(self, state):
            return await slow(state)

    monkeypatch.setattr(routes, "create_coin_analysis_team", lambda **_kwargs: SlowGraph())
    payload = {"llm": LLM, "coin": {"id": 1, "name": "Lookup Candidate"}}
    with _running_server() as base_url:
        _assert_client_timeout_cancels(base_url, "/api/analyze", payload, events)
