"""Stop route work when the Go caller gives up on the request (#785).

Go bounds agent calls with a context timeout. When it expires, Go closes the
HTTP connection, but a non-streaming FastAPI handler would otherwise keep
running provider, dealer and web-search calls whose result nobody reads.
"""

import asyncio
import logging
from collections.abc import Awaitable
from typing import TypeVar

from fastapi import Request
from fastapi.responses import Response

logger = logging.getLogger(__name__)

T = TypeVar("T")

CLIENT_CLOSED_REQUEST = 499


class ClientDisconnectedError(Exception):
    """The caller closed the connection, so the route's work was cancelled."""


async def _wait_for_disconnect(http_request: Request) -> None:
    # Once the body is consumed, the ASGI server's next message is the disconnect.
    # A blocking receive is used because Request.is_disconnected() cannot observe
    # it through BaseHTTPMiddleware (InternalServiceAuthMiddleware).
    while True:
        message = await http_request.receive()
        if message.get("type") == "http.disconnect":
            return


async def cancel_on_disconnect(http_request: Request, work: Awaitable[T]) -> T:
    """Await ``work``, cancelling it as soon as the client disconnects.

    Cancelling the task interrupts in-flight awaits (model, dealer and search
    calls), not just the checkpoints between steps.
    """
    task = asyncio.ensure_future(work)
    watcher = asyncio.ensure_future(_wait_for_disconnect(http_request))
    try:
        await asyncio.wait({task, watcher}, return_when=asyncio.FIRST_COMPLETED)
        if task.done():
            return task.result()
        if watcher.cancelled() or watcher.exception() is not None:
            # A receive failure is not evidence of a disconnect; let the work finish.
            reason = "cancelled" if watcher.cancelled() else repr(watcher.exception())
            logger.warning("Disconnect watcher for %s failed: %s", http_request.url.path, reason)
            return await task
        task.cancel()
        await asyncio.gather(task, return_exceptions=True)
        logger.info("Client disconnected from %s; cancelled agent work", http_request.url.path)
        raise ClientDisconnectedError
    finally:
        for pending in (task, watcher):
            if not pending.done():
                pending.cancel()
        await asyncio.gather(task, watcher, return_exceptions=True)


async def client_disconnected_handler(_request: Request, _exc: Exception) -> Response:
    return Response(status_code=CLIENT_CLOSED_REQUEST)
