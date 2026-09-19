# Feature 363 Quality Evidence

## Focused URL-intake verification

Validated on 2026-09-19:

| Area | Command | Result |
|---|---|---|
| Go URL policy, duplicate, cleanup, outbound HTTP, handler, architecture | `go test ./services ./handlers . -run "WishlistURL|Outbound|Architecture" -count=1` from `src/api` | Passed |
| Go OpenAPI drift | `go test . -run "TestFeature363OpenAPISurface|TestRegisteredAPIRoutesAreDocumentedInOpenAPI" -count=1` from `src/api` | Passed |
| Python strict extraction | `python -m ruff check ...` and `python -m pytest tests/test_wishlist_url_extraction.py -q` from `src/agent` | Passed, 7 tests |
| Vue URL intake | `vue-tsc --build`, changed-file eslint, and focused Vitest suites | Passed, 15 tests |

Focused fixtures prove:

- canonical URLs remove tracking parameters but preserve identity query values
  and fragments;
- credentials, localhost, and IP literals are rejected, while the shared
  restricted dialer and redirect validator continue to reject non-public DNS
  results;
- duplicate lookup is owner-scoped and completes before retrieval;
- only one bounded HTML response is read and links are never crawled;
- storefront chrome, controls, related listings, and repeated text are removed;
- unsupported model fields and facts without verbatim page evidence are
  discarded;
- legends and descriptions remain separate and numeric values are normalized;
- sold, reserved, and withdrawn listing status remains explicit;
- rendering a proposal creates nothing; only explicit form confirmation calls
  canonical coin creation;
- cancellation creates nothing, duplicate results link to the existing coin,
  and image preview/upload failure does not undo coin creation;
- the architecture guard finds no auction-subsystem reference in URL intake.

## Controlled tamper evidence

Each guard was weakened temporarily, its focused test failed, and the original
implementation was restored:

| Guard weakened | Expected failure observed |
|---|---|
| Duplicate short-circuit disabled | Duplicate test attempted network retrieval |
| IP-literal rejection disabled | IPv4 and IPv6 loopback cases were accepted |
| Verbatim evidence filter disabled | Unsupported ruler and mint fields survived |
| Auction-isolation marker inserted | Architecture guard rejected the source |
| Explicit-confirmation boundary bypassed | Vue test observed coin creation before confirmation |

All restored focused suites passed afterward.

## Full quality gate

| Layer | Commands | Result |
|---|---|---|
| Go | `go build ./...`; `go vet ./...`; `go test ./... -count=1` | Passed all packages |
| Python | `python -m ruff check .`; `python -m pytest -q` | Passed, 649 tests; existing Python 3.14/LangChain warnings only |
| Vue | eslint; `vue-tsc --build`; full Vitest; Vite production build | Lint and type-check passed; 1,604 tests passed and four unrelated timing/navigation failures passed immediately in isolated rerun (27 tests); production/PWA build passed |
| OpenAPI | `swag init -g main.go -o ./docs --parseDependency --parseInternal`; route/schema assertions | Generated and passed |

The implementation adds no persistence, action lifecycle, generic crawler,
browser-to-Python route, or auction model/repository/service/route dependency.
The only write remains explicit browser confirmation through
`POST /api/coins`, followed by at most one best-effort image upload.
