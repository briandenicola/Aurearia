---
name: toolchain-pinning
description: "Review synchronized language, CI tool, and Docker pins before an authorized toolchain change."
---

# Toolchain Pinning Skill

Use this when a task changes CI-installed tools, language patch versions, or Docker base-image references.

1. Inventory all matching pins with `rg`: language version files, workflows, Taskfile targets, Dockerfiles, and directly related docs.
2. Replace mutable Go tool installs with reviewed module versions. Keep workflow installs and matching Taskfile installs identical.
3. For Docker base images, prefer `tag@sha256:<OCI index digest>` so multi-arch builds still work. Refresh the digest for the exact reviewed tag; do not switch image families or OS variants unless the task explicitly asks.
4. Update docs that describe the runtime/tool version and pin refresh cadence.
5. Validate the smallest relevant surface: OpenAPI generation when `swag` changes, `cd src/api; go test ./...`, and `govulncheck ./...` when available.
6. Leave locked specs/constitution entries alone unless an amendment is explicitly in scope.

Changing a pin is not permission to install or deploy it. Preserve the separate
`setup:*` and `check:*` boundary; use all applicable completion gates from
`docs/testing.md` section 6, not a single targeted test as final evidence.
