# MCP Admin Toggle

**Date:** 2026-10-05
**Owner:** Copilot CLI implementation owner
**Branch:** `beta`
**Base:** `4c484a6b`

## Outcome

Added the existing default-off `ExternalToolServerEnabled` setting to
**Admin → AI**. The shared toggle updates the existing string-valued app
setting and the established AI settings save action persists it. No API,
authorization, route, or setting semantics changed: the existing middleware
continues to gate both `/api/mcp` and `/api/v1/tools/*`.

Updated the in-app help, external-tool guide, MCP client setup guide, and API
reference to point administrators to **Admin → AI**. The in-app help now
describes the native Streamable HTTP MCP endpoint instead of the superseded
`mcpo` wrapper path.

## Evidence

- `node .\node_modules\vitest\vitest.mjs run src\components\admin\__tests__\AdminAISection.test.ts`
  passed: 2 tests.
- `node .\node_modules\vue-tsc\bin\vue-tsc.js --build` passed.
- Tamper proof: forcing the toggle update handler to retain `"false"` caused
  the new persistence test to fail; restoring the handler returned the test to
  green.
- `git diff --check` passed.

The owner authorized targeted validation only. Full lint, web suite, production
build, and browser checks were not run locally and remain CI evidence.

## Next Action

Commit and push the bounded change to `beta`. Release or deployment requires
separate authorization.
