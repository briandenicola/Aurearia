# Aurearia: Universal Agent Instructions

Aurearia is a Go API, Vue/TypeScript PWA and stateless Python agent service.
Read the [constitution](../.specify/memory/constitution.md) before editing. Its
Principles I–VI are the architecture, typing, security and UI rules; its
Working Rules say how to deliver a change.

- Load only the [scoped instructions](instructions) and [skills](skills/README.md)
  that match the files you are changing.
- Work on `beta` unless told otherwise. Leave unrelated changes alone.
- Use [testing](../docs/testing.md#6-running-tests-locally-vs-ci) for commands.
  Run the checks for the areas you touched before saying a change works.
- Report what you ran and what you could not run. Unrun checks are "not
  verified", never "passed".
- Do not merge, release, deploy or change repository settings unless the
  owner asks for that specific action.
- Keep PR descriptions short, using the [PR template](pull_request_template.md).
