"""Internal service authentication middleware."""

import secrets

from fastapi import Request
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import JSONResponse

from app.config import settings

PUBLIC_PATHS = {"/health", "/ready"}
INTERNAL_TOKEN_HEADER = "x-internal-service-token"


class InternalServiceAuthMiddleware(BaseHTTPMiddleware):
    """Require Go API's shared internal credential on non-health endpoints."""

    async def dispatch(self, request: Request, call_next):
        if request.url.path in PUBLIC_PATHS:
            return await call_next(request)

        expected = settings.internal_service_token
        if not expected:
            return JSONResponse({"detail": "Internal service credential is not configured"}, status_code=503)

        provided = request.headers.get(INTERNAL_TOKEN_HEADER, "")
        if not provided or not secrets.compare_digest(provided, expected):
            return JSONResponse({"detail": "Unauthorized"}, status_code=401)

        return await call_next(request)
