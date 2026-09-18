# AI browser exploration

This directory owns the Feature 360 TypeScript controller boundary: checked-in
contracts, immutable run limits, browser orchestration, sanitized evidence, and
report finalization. It may use Playwright only through bounded, allowlisted
actions and must not contain provider SDKs, credentials, production endpoints,
or a second copy of the F013 fixture catalog.

Tests live in `__tests__/`. The Go API owns the internal proxy contract and the
Python agent owns stateless model inference.
