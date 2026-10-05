---
updated_at: 2026-10-05
focus_area: MCP admin enablement control
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: beta
baseline_commit: 4c484a6b
work_artifact: specs/366-mcp-agentic-harness/spec.md
tasks_artifact: specs/366-mcp-agentic-harness/tasks.md
---

# Current Work

The owner requested a UI control for the existing default-off
`ExternalToolServerEnabled` setting. The control now appears in **Admin → AI**,
uses the shared toggle primitive, and saves through the existing app-settings
contract. It gates both native MCP and the OpenAPI external-tool surface through
the existing backend middleware.

The in-app help and external-tool/MCP documentation now point to the actual UI
location and save action. Targeted component tests and strict Vue type-check
passed; the new regression test was tamper-proven. Full web validation was not
authorized locally and remains CI evidence.

## Next Action

Verify the governance-metadata repair through the delivery gate and beta CI.
Release or deployment requires separate owner authorization.
