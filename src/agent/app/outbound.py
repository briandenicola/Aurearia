"""Outbound URL validation for agent-owned HTTP clients."""

import socket
from ipaddress import ip_address
from urllib.parse import urljoin, urlparse

import httpx

from app.config import settings

METADATA_IPS = {ip_address("169.254.169.254")}
_REDIRECT_STATUS_CODES = {301, 302, 303, 307, 308}


def _configured_origins() -> set[str]:
    return {
        origin.rstrip("/")
        for origin in (item.strip() for item in settings.trusted_outbound_origins.split(","))
        if origin
    }


def _origin(parsed) -> str:
    port = f":{parsed.port}" if parsed.port else ""
    return f"{parsed.scheme}://{parsed.hostname}{port}"


def _is_local_or_private(host: str) -> bool:
    lower = host.lower()
    if lower == "localhost" or lower.endswith(".localhost"):
        return True
    try:
        ip = ip_address(lower.strip("[]"))
    except ValueError:
        return False
    return (
        ip.is_loopback
        or ip.is_link_local
        or ip.is_private
        or ip.is_multicast
        or ip.is_reserved
        or ip.is_unspecified
    )


def _is_metadata_host(host: str) -> bool:
    try:
        return ip_address(host.strip("[]")) in METADATA_IPS
    except ValueError:
        return False


def _resolved_addresses_are_safe(host: str, port: int | None) -> bool:
    try:
        ip_address(host.strip("[]"))
        return True
    except ValueError:
        pass

    try:
        addresses = socket.getaddrinfo(host, port, type=socket.SOCK_STREAM)
    except socket.gaierror as exc:
        raise ValueError("host could not be resolved") from exc

    return all(not _is_local_or_private(address[0]) for *_, address in addresses)


def _validate_http_url(
    url: str,
    field_name: str,
    *,
    require_trusted_origin: bool,
    strip_trailing_slash: bool,
) -> str:
    value = url.strip()
    if strip_trailing_slash:
        value = value.rstrip("/")
    if not value:
        return ""

    parsed = urlparse(value)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        raise ValueError(f"{field_name} must be an http(s) URL with a host")

    origin = _origin(parsed)
    trusted = _configured_origins()
    is_trusted = origin in trusted
    if require_trusted_origin and not is_trusted:
        raise ValueError(f"{field_name} origin is not trusted")

    if _is_metadata_host(parsed.hostname):
        raise ValueError(f"{field_name} uses a metadata service address")

    local_dev_allowed = settings.allow_local_outbound and is_trusted
    if _is_local_or_private(parsed.hostname) and not local_dev_allowed:
        raise ValueError(f"{field_name} uses a local or private address")
    if not require_trusted_origin and not is_trusted and not _resolved_addresses_are_safe(parsed.hostname, parsed.port):
        raise ValueError(f"{field_name} resolves to a local or private address")

    return value


def validate_outbound_url(url: str, field_name: str) -> str:
    """Return a normalized URL after enforcing trusted-origin rules."""
    return _validate_http_url(
        url,
        field_name,
        require_trusted_origin=True,
        strip_trailing_slash=True,
    )


def validate_public_outbound_url(url: str, field_name: str = "url") -> str:
    """Return a URL safe for public internet fetches.

    Public fetches do not require a configured origin, but local/private targets
    remain blocked unless explicitly trusted for local development.
    """
    return _validate_http_url(
        url,
        field_name,
        require_trusted_origin=False,
        strip_trailing_slash=False,
    )


async def safe_get(
    url: str,
    *,
    field_name: str = "url",
    headers: dict[str, str] | None = None,
    params: dict[str, str] | None = None,
    timeout: httpx.Timeout | float | int | None = None,
    max_redirects: int = 5,
) -> httpx.Response:
    """GET a public URL, validating the initial URL and every redirect target."""
    current_url = validate_public_outbound_url(url, field_name)
    next_params = params

    async with httpx.AsyncClient(timeout=timeout, follow_redirects=False) as client:
        for _ in range(max_redirects + 1):
            response = await client.get(current_url, headers=headers, params=next_params)
            next_params = None
            if response.status_code not in _REDIRECT_STATUS_CODES:
                return response

            location = response.headers.get("location")
            if not location:
                return response

            redirect_url = urljoin(str(response.url), location)
            current_url = validate_public_outbound_url(redirect_url, f"{field_name} redirect")

        raise httpx.TooManyRedirects(
            f"Exceeded {max_redirects} redirects while fetching {field_name}",
            request=response.request,
        )
