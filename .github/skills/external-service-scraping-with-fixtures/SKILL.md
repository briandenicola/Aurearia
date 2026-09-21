---
name: external-service-scraping-with-fixtures
description: "Change an external-service scraper using sanitized fixtures, HTTP stubs, and explicit credential boundaries."
---

# Scraper evidence without live-service coupling

Read the approved provider scope and existing parser/client/test seams first.
This skill does not authorize auction subsystem changes or live account access.

1. Use minimal sanitized HTML/JSON under the existing testdata convention.
   Never commit raw HARs, cookies, account identifiers, bid tokens or credentials.
   Obtain permission for any live capture; respect provider access constraints.
2. Separate parser tests from HTTP orchestration. Inject an `httptest` server or
   equivalent transport seam into the actual client; merely constructing a stub
   without pointing the client at it does not prevent live traffic.
3. Cover public/authenticated variants, missing fields, malformed markup,
   sold/unavailable content returned with HTTP 200, 404, 429, 5xx and timeout.
   Assert extracted fields and typed failures, not just non-empty output.
4. Preserve service/repository boundaries, parameterized data access and owner
   scoping. Reuse sentinel errors rather than leaking upstream response bodies.
5. Bound timeouts/retries, propagate cancellation and release response bodies.
   Do not log credentials, raw sessions or sensitive query parameters.
6. Keep normal CI deterministic and offline with respect to external providers.
   Live integration checks require explicit owner approval and isolated test
   credentials; missing live evidence is pending, never an implicit CI pass.
7. Review fixture changes as contract changes: prove assertions still detect the
   original bug. Updating golden data does not automatically make behavior correct.
8. Use existing secret scanning; keyword-only hooks are not credential detection.

Run applicable authorized shared gates and sibling-provider regressions using
real discovered test names. Do not invent selectors that can run zero tests.
