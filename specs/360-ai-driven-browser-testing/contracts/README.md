# Feature 360 Contracts

These contracts are the implementation source of truth for the exploratory
runner's external and cross-service interfaces.

| File | Purpose |
|---|---|
| `run-config.schema.json` | Secret-free input accepted by local/manual/nightly runs. |
| `exploration-agent.openapi.yaml` | Internal Go-to-Python model decision request/response. |
| `exploration-report.schema.json` | Machine-readable artifact published by every safely finalized run. |
| `issue-publication.schema.json` | Sanitized post-report request accepted by the optional publisher. |

## Compatibility

- Contract IDs are immutable. Breaking changes require a new `/v2` identifier
  and explicit compatibility handling.
- Producers reject unknown fields. Consumers must not infer fields omitted by
  the schema.
- Secrets are forbidden in all four contracts.
- Go DTOs, Python Pydantic models, TypeScript types, and schema examples must be
  checked for drift in automated tests.
- The OpenAPI endpoint is internal, enabled only in an ephemeral exploration
  stack, and authenticated with that run's internal service token.

## Report bundle

`report.json` validates against the report schema. Every `artifactPath` is
relative to the report directory and must remain beneath `evidence/`. The
human-readable `summary.md` is derived from the validated report and is not an
independent source of truth.
