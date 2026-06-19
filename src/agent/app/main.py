"""FastAPI entry point for the Ancient Coins multi-agent service."""

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware

from app.config import settings
from app.logging_config import ring_handler, set_log_level, setup_logging
from app.routes import router
from app.security import InternalServiceAuthMiddleware

# Configure logging before anything else
setup_logging(settings.log_level)

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(application: FastAPI):
    logger.info("Agent service starting (log_level=%s)", settings.log_level)
    yield


app = FastAPI(
    title="Ancient Coins Agent Service",
    version="0.1.0",
    docs_url="/docs" if settings.debug else None,
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)
app.add_middleware(InternalServiceAuthMiddleware)

app.include_router(router)


@app.get("/health")
async def health():
    return {"status": "ok", "service": "agent"}


@app.get("/ready")
async def ready():
    return {"status": "ready", "service": "agent"}


@app.get("/logs")
async def get_logs(
    limit: int = Query(default=500, ge=1, le=2000),
    level: str = Query(default=""),
):
    """Return recent log entries from the in-memory ring buffer."""
    current_level = logging.getLevelName(logging.getLogger().level)
    logs = ring_handler.get_logs(limit=limit, level=level)
    return {"logs": logs, "count": len(logs), "logLevel": current_level}


@app.put("/log-level")
async def update_log_level(body: dict):
    """Dynamically update the agent service log level."""
    new_level = body.get("level", "")
    if not new_level:
        raise HTTPException(status_code=400, detail="level is required")
    applied = set_log_level(new_level)
    logger.info("Log level changed to %s", applied)
    return {"logLevel": applied}
