# MCP Agentic Harness Completion Record

Date: 2026-10-04
Branch: `beta`
Base HEAD: `73d5750c7cdbd28938efff18f92dd4ba9994cb6a`
Candidate manifest SHA-256:
`ff9401e73042ce36e50070bc6a375b89f6ebec1899f14825a63e800f0b19fc47`

The candidate identity is the SHA-256 of the sorted UTF-8 manifest containing
`<file-sha256><two spaces><repository-relative-path>` for every modified or
untracked non-log file reported by:

```powershell
git ls-files -m -o --exclude-standard
```

Exact sorted candidate manifest:

```text
370cd771e73ab0c71d703670ce1a1e8bed599990bdc6a134c822485145ee7ed0  .github/skills/README.md
6ef0454bf633e6db9af2b5c65ade9009e7454d1b835856dcf4042fb54e0b9cf6  .github/skills/using-aurearia-mcp/SKILL.md
4d6dd44bfbc2bf304e8348fd4d6e3407bab23d553db8c1b22fcdc1cd418753ba  .squad/identity/now.md
4a2b6b59dd4258a9aa410c70de93896eec2bb258b58a6aea6b7e28a24d52ed8b  docs/adr/0020-stateless-mcp-api-adapter.md
68baf0da278547b2296a1881eb1e30f4c7775638681c45208e4b4fc8b61cbcd9  docs/adr/README.md
4106da6fb92b819107f242be663ca380134f7dc07eeadc0eedc85631d8a4fa16  docs/external-tool-server.md
031373c2057edd8f974665398d134b14cd6dc89eeec29efcd2c3c25b43f43178  docs/openapi.json
556781aebd98ae903d0ac988a27f03735b0241a1b540992046a41d0f41ac002a  specs/365-mcp-agentic-harness/plan.md
309fe4c0801b6238663c6d39ff6ed3c3bcd42a6be6550aed76de87cd553d684e  specs/365-mcp-agentic-harness/spec.md
39dbb09b46e094ad968cdd64c11036691e6cc0e67c796eccc261d114185364c2  specs/365-mcp-agentic-harness/tasks.md
9ae35438c68a86f85a17a0037a8cd8754ef658ed9ecd790d1b8a81c75814a606  src/api/architecture_test.go
c7e24ff185aea80fec3c5178d37ff85da5b6a55b1f12365b3df52478b4677572  src/api/deps.go
ffedba2cdc8975a77778845873019e9412c216a3905699dff8e96eb19fe55821  src/api/docs/docs.go
031373c2057edd8f974665398d134b14cd6dc89eeec29efcd2c3c25b43f43178  src/api/docs/swagger.json
94e6e60c1927508ac33225a6e09bc0a518dac9d398553fd852ce2cafe4dbe599  src/api/docs/swagger.yaml
4962c13e4dbe1fd78871d1bcecfece7e73a2fca8f27b67cdf0d3c68076ed4231  src/api/go.mod
b37f44b55f27c8ef74df54356f5d14cc4f2ce3143811bd019a5b65b81b208610  src/api/go.sum
17bf0226923b8f0c3a052663c35d94da3b24d0bbdfcf1cce39a9e261f566aa21  src/api/handlers/api_keys.go
bdf182d6e8cf0a4615381e83ce9c02b4c0b192d98b3b86a02771bc94eebea8da  src/api/handlers/mcp_test.go
b5bc4fd9e5395e6a4ac86abe659887747f22afc65403958b2e6a22f67e7cf5dd  src/api/handlers/mcp.go
f394b061140e314494d8bbe47bc04f5307b28f164128e766162f99cbcfcf4c76  src/api/models/api_key_test.go
df724a44909a6318ad9ebd104830e33ba8510de3f0e0f32602b0a7bc85e9bb64  src/api/models/api_key.go
b86370178898df3147ab19edf38d188e8bf4f8c8f562bd7f8abbedbe8c31846a  src/api/repository/api_key_repository.go
82be1c6205514b1adbb8c553e2d5c7414286d9a7c7d3bf3ffa4253a8bc45aba8  src/api/routes_tools.go
e1431aaec7376f5a0d2deb800ca31b850e70128ae77df190632aac11d77ac0d6  src/api/services/collection_tools_service.go
763d32d5930176c11504f7b9851d19faea023a686a3ab1656bc14a0ad91aae49  src/api/services/mcp_service_test.go
88cb98c1046e1d81fd16d6c4b0cecb44c59171ee84860a15a89ed0dbcb04119e  src/api/services/mcp_service.go
84f85fcbcf95427bffdb19417e8761af146f9d8f76e3bafe8ffeacb61e05b763  src/web/src/api/endpoints/auth.ts
659ebf3c991cd7c9481f25fb7fe560d833927efe7355fa4aff42a7e9d519dd4a  src/web/src/components/settings/SettingsApiKeysSection.vue
7e87de30ebfc3c6f61d6aad5b9efb8c3305eb3d1604d7459ed21a1a774de91af  src/web/src/types/auth.ts
```

The bounded tracked diff is the diff from the base HEAD limited to the paths
above. Untracked files are reviewed at the exact hashes recorded above.

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

- `task check:go`: passed using the exact cached Go 1.26.6 toolchain with
  `GOTOOLCHAIN=local` and `GOPROXY=off`.
- `task check:web`: passed (zero-warning lint, strict type check, tests, and
  production build).
- `task check:delivery`: passed; governance reported only existing context
  warnings plus the expected Proposed ADR lifecycle warning.
- `task openapi`: passed using the exact cached Go 1.26.6 toolchain.
- `go test . -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1`:
  passed.
- `git diff --check`: passed.
- `task check:openapi`: generator completed, then intentionally reported the
  uncommitted generated OpenAPI diff against HEAD. This is not schema drift;
  the generated files are part of the candidate.

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
No deployment or release is authorized.
