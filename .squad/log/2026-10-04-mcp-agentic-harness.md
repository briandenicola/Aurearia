# MCP Agentic Harness Completion Record

Date: 2026-10-04
Branch: `beta`
Base HEAD: `a4e16dcf2e01450e8373c5068023b194e484920b`
Rebased candidate commit: `acad3309af24c92e0502df1bdcfeed4282ac823d`
Post-rebase metadata delta SHA-256:
`96ffabc530f6119e2368ac71e9790fb3876bf9387e8f8c52a890267f81db139f`

The implementation candidate is the exact committed tree at `acad3309`,
bounded by `git diff a4e16dcf..acad3309`. The final candidate applies this
sorted metadata manifest to that commit; the completion log is excluded from
its own digest:

```text
ccbeb6576428176a574fc0532f5086a816dec76bd3302313656302b500e4f603  .squad/identity/now.md
224285ce2d9350668838e6300263fb37533899c4df23500f1978489351894751  docs/adr/0021-stateless-mcp-api-adapter.md
DELETE  specs/365-mcp-agentic-harness/plan.md
DELETE  specs/365-mcp-agentic-harness/spec.md
DELETE  specs/365-mcp-agentic-harness/tasks.md
0b4fee40613f967f56209479b140aaa965926df2fbf81b000c6f9c8da9b66533  specs/366-mcp-agentic-harness/plan.md
b1b6f517993d85902e77fcd56a0f8900b1dd257d2f705198a1693820dafc3472  specs/366-mcp-agentic-harness/spec.md
831deb0e1a2757de88de57b4815cc77268bd15d4aa7064c2cce3bb6e67cd36f2  specs/366-mcp-agentic-harness/tasks.md
```

The earlier dirty-tree manifests in the review history identify the same
implementation before the required rebase. The rebase required three metadata
reconciliations:

1. The MCP decision was renumbered from proposed ADR 0020 to proposed ADR 0021
   because the remote branch had accepted the voice-input ADR as 0020.
2. The remote release-candidate audit remained the current-work pointer, with
   the completed MCP delivery recorded as sibling work.
3. The MCP planning artifacts moved from Feature 365 to the next unused
   Feature 366 because the remote branch had assigned Feature 365 to desktop
   contextual headers.

## Criteria and workflows

| Criterion | Result | Evidence |
|---|---|---|
| Stateless MCP endpoint is default-off, authenticated, owner-scoped, read-capable, and rate-limited | Implemented and verified | `src/api/routes_tools.go`; full Go gate |
| Read-only collection, wishlist, auction, and stats tools | Implemented and verified | MCP discovery and owner-isolation tests |
| Copilot start/get/resume/cancel requires exact `copilot` capability | Implemented and verified | capability discovery test plus negative tamper |
| No mutation or arbitrary execution tools | Implemented and verified | MCP tool-surface negative test |
| API key management supports explicit `read,copilot` | Implemented and verified | model/repository/UI changes; Go and web gates |
| Portable skill and secret-safe setup guidance | Implemented and delivery-checked | `.github/skills/using-aurearia-mcp/SKILL.md` |
| Route and generated OpenAPI remain aligned | Implemented and verified | generation success and route drift test |

## Verification evidence

- Before rebase, `task check:go` passed using the exact cached Go 1.26.6
  toolchain with `GOTOOLCHAIN=local` and `GOPROXY=off`.
- After rebase, `task check:go` passed build and vet, then its formatting step
  reported every Go file because the Windows checkout had converted the full
  Go worktree to CRLF. No broad unrelated formatting rewrite was performed.
  `task test` then passed the complete Go suite on commit `acad3309`.
- `task check:web`: passed (zero-warning lint, strict type check, tests, and
  production build).
- `task check:delivery`: passed; governance reported only existing context
  warnings plus the expected Proposed ADR lifecycle warning.
- `task openapi`: passed using the exact cached Go 1.26.6 toolchain.
- `go test . -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1`:
  passed.
- `git diff --check`: passed.
- Before rebase, `task check:openapi` generated the intended candidate diff.
  After rebase, `task check:openapi` passed and the generated artifacts matched
  the committed candidate after Git's line-ending normalization.

## Negative control

The `models.HasAPICapability(capabilities, "copilot")` guard in
`handlers/mcp.go` was temporarily bypassed with `|| true`.
`TestMCPToolDiscoveryRequiresExactCopilotCapability` failed for `read`,
`read,write`, and malformed `read,copilotx` keys because all four Copilot tools
became discoverable. The bypass was removed and the targeted test and full Go
gate passed.

## Operational note

The owner authorized updating the machine's local Go installation from 1.26.1
to 1.26.6. The Winget MSI was cancelled before installation (exit 1602), so no
machine-wide update was completed. Validation instead used the already-cached,
exact 1.26.6 toolchain without changing the installed toolchain.

## Review

The first candidate-bound pass blocked because the digest was not accompanied
by its reconstructable sorted manifest. After the manifest was persisted, the
same reviewer found two blockers:

1. Native route-boundary coverage did not yet exercise the middleware and MCP
   transport with real API-key authentication.
2. The external-tool guide retained obsolete `mcpo` proxy instructions that
   could expose the broader OpenAPI write surface.

Both were repaired. Route-level tests now initialize the native endpoint,
discover and invoke all read and Copilot tools, and verify owner isolation,
safe errors, exact capability gating, forbidden write-shaped calls,
default-off behavior, invalid and revoked keys, the request-size limit and
per-key throttling. Successful Copilot start, get, queued-run cancellation and
paused-checkpoint resume are asserted through the MCP protocol, including their
persisted state transitions. The obsolete proxy guidance was removed. The
repaired targeted MCP tests, complete `task check:go`, and
`task check:delivery` pass.

The same independent reviewer returned `CLEAR` for candidate
`080505e0dcc06f6cb9b7760219bda573939672eb78b88d7d40f9e4137876e5b1`
after verifying the lifecycle success-path repair. Final completion metadata is
the only subsequent candidate change. The reviewer then returned `CLEAR` for
the final candidate
`ff9401e73042ce36e50070bc6a375b89f6ebec1899f14825a63e800f0b19fc47`.
The reviewer subsequently blocked the rebased candidate solely because remote
Feature 365 conflicted with the MCP planning identity. The MCP artifacts and
references were renumbered to Feature 366. The same reviewer returned `CLEAR`
for implementation commit `acad3309af24c92e0502df1bdcfeed4282ac823d`
plus metadata manifest
`96ffabc530f6119e2368ac71e9790fb3876bf9387e8f8c52a890267f81db139f`.
No deployment or release is authorized.
