---
applyTo: ".github/**,.specify/**,scripts/delivery/**,Taskfile.yml,Dockerfile,src/agent/Dockerfile,docs/testing.md"
---

# CI, build and tooling changes

- CI is deliberately small (see [testing](../../docs/testing.md), §6). Each test
  workflow runs once per PR for the area it covers. Do not add jobs, gates or
  verifiers unless they catch a class of bug the existing tests cannot.
- Keep workflows calling the Taskfile `check:*` targets so local and CI
  behavior match. Recipes must not install tools or skip lint.
- Pin Actions and tool versions. When changing one, update every build, CI and
  setup file that uses it.
- After changing a workflow, say which jobs you ran and which only run on
  GitHub, so the owner knows what is unverified.
