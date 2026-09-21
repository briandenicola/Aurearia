---
name: python-locking
description: "Change Python dependency locks, prepared environments, or container installs without implicit tool upgrades."
---

# Locked Python changes

1. Treat `src/agent/pyproject.toml` and `src/agent/uv.lock` as the manifest/lock
   pair. Read the reviewed uv/Python pins from current workflows and Dockerfiles.
2. Do not install or upgrade uv/Python implicitly. Use an already prepared
   environment, or obtain setup approval.
3. For an authorized dependency change, refresh the lock intentionally with
   `uv lock`; use a targeted upgrade when needed rather than upgrading everything.
   Review unrelated resolution changes before accepting the lock.
4. Authorized `task setup:agent` synchronizes dev dependencies. Completion is
   `task check:agent`: verify the prepared lock/environment with non-mutating
   `uv sync --locked --check --offline --extra dev`, then no-sync/offline lint/tests.
5. Preserve the existing multi-stage runtime pattern: locked runtime-only
   dependencies, copied virtual environment, no dev tools or package installer
   in the final image. Validate container changes on the authorized runner.
6. Keep Dependabot's `uv` ecosystem and directory consistent with the manifest.
   Missing caches or a failed synchronization check are errors, not permission
   to download during a completion gate.
