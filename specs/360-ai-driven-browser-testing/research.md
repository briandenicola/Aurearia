# Phase 0 Research: AI-Driven Browser Testing

All technical unknowns from the planning template are resolved below.

## Decision 1: Extend the existing three-service boundary

**Decision**: Use a TypeScript Playwright controller, an opt-in internal Go
proxy, and a strict Python-agent decision endpoint. The controller never calls a
model provider directly.

**Rationale**: Playwright 1.63.0 is already installed and F013 already owns the
browser workflow vocabulary. Constitution Principle II requires model inference
to stay in the Python agent and Go-to-agent calls to use the proxy boundary.
The existing `src/agent/app/llm/provider.py` already normalizes Anthropic and
Ollama structured output.

**Alternatives considered**:

- Node provider SDKs/direct REST: fewer hops, but duplicates provider behavior,
  exposes credentials to the browser harness, and violates the established AI
  service boundary.
- Python Playwright: keeps AI and browser code together but adds a second
  Playwright dependency/runtime and duplicates F013 TypeScript fixtures.
- A new agent framework: rejected as unnecessary dependency and abstraction.

## Decision 2: Dedicated provider credentials stay outside application settings

**Decision**: The run contract carries only provider and model identifiers.
Dedicated exploration credentials are supplied as protected environment
variables to the Go proxy for that ephemeral run and forwarded per request to
the agent. Production `AppSetting` values are never read.

**Rationale**: This satisfies FR-001 and existing stateless-agent semantics
without serializing a credential into configuration, reports, or browser state.
It also permits provider-specific credential names behind a common contract.

**Alternatives considered**:

- Reuse admin-configured production credentials: rejected by scope and privacy.
- Put keys in `run-config.json`: rejected because configuration is persisted.
- Store credentials in the agent: rejected because the agent is stateless.

## Decision 3: Reuse existing dependencies and pin exact resolutions

**Decision**: Add no npm/Python library. Pin the direct
`@playwright/test` declaration to `1.63.0`, keep the lockfile's Playwright
family at `1.63.0`, use Node `20.19.0`, and preserve `uv.lock` resolutions
LangChain `1.4.0`, LangChain Anthropic `1.7.2`, LangChain Ollama `1.1.0`, and
LangGraph `1.2.11`. Pin new GitHub Actions by full commit SHA.

**Rationale**: Playwright, Pydantic, structured provider adapters, SHA-256,
fetch, JSON, and process orchestration are already available. Exact versions
meet Constitution Principle VII and minimize supply-chain surface.

**Alternatives considered**:

- Add a JSON Schema validator: rejected initially; validate with a small typed
  runtime guard generated alongside schema fixtures and contract tests.
- Add Octokit: rejected; Node `fetch` is sufficient for two GitHub REST calls.
- Add a browser-agent package: rejected due to hidden actions and weaker bounds.

## Decision 4: Build a fresh Compose project from current source

**Decision**: Add `docker-compose.exploration.yml` that builds the root app and
agent images from the checkout, uses a unique project name, loopback-only random
port, new SQLite/upload volumes, internal-only agent networking, and
unconditional `down -v --remove-orphans`.

**Rationale**: The deployment Compose file uses mutable `latest` images,
persistent named volumes, fixed port 8080, and restart policies. It is not an
ephemeral test environment. Current-source builds and isolated volumes prove
the code under test and prevent production-data access.

**Alternatives considered**:

- Reuse deployment Compose: rejected due to mutable images/durable volumes.
- Mock APIs as F013 does: insufficient because Feature 360 requires full stack.
- Run against beta: explicitly prohibited.

## Decision 5: Seed through a one-shot Go command

**Decision**: Add a test-only command that creates the dedicated application
user and invokes `src/api/testutil.PersistGoldenCollection` against the
ephemeral database before exploration.

**Rationale**: It reuses the canonical F013 fixture builder, including
associations, without creating a competing fixture catalog or a production
seeding HTTP endpoint.

**Alternatives considered**:

- Copy frontend fixture JSON into SQLite: rejected as duplicate ownership.
- Expose a public seed endpoint: rejected as unnecessary production surface.
- Use production backup/data: prohibited.

## Decision 6: Centralize fail-closed budget enforcement

**Decision**: One immutable `RunLimits` and one `BudgetGuard` enforce
reserve/perform/reconcile around every counted operation. Attempted work counts.
Deadline checks use a monotonic clock and `AbortController`. Missing or invalid
provider token usage terminates before another model call.

**Rationale**: Distributed counters and post-run checks can overshoot during
timeouts/retries. A single guard is independently testable at exact boundaries.

**Alternatives considered**:

- Model instructions only: rejected; prompts are not enforcement.
- Check only after operations: rejected because it permits overrun.
- Estimate unavailable tokens: rejected by the spec's fail-closed assumption.

## Decision 7: Use an allowlisted action language

**Decision**: Model decisions may select only `navigate`, `click`, `fill`,
`select`, `upload_fixture`, `set_viewport`, `back`, `wait_for_ui`, `checkpoint`,
or `finish`, with bounded typed arguments. Routes must be same-origin and in the
selected F013 workflow. File uploads may reference only packaged synthetic
fixtures. There is no shell, arbitrary JavaScript, filesystem path, external
URL, or generic request action.

**Rationale**: This provides hard behavioral bounds and makes action counting
deterministic.

**Alternatives considered**:

- Free-form generated Playwright: rejected as code execution.
- Generic tool calling: rejected because its capability surface is too broad.

## Decision 8: Sanitize before prompts or persistence

**Decision**: Place one mandatory sanitizer immediately after collection and
before prompts, logs, screenshots, reports, traces, or publication. Do not
collect cookies, storage, headers, authorization, raw bodies, password fields,
binary payloads, or unrestricted DOM. Disable traces/video/HAR. Mask sensitive
selectors and gate screenshots on known canaries.

**Rationale**: Final-report redaction cannot protect earlier prompt, trace, or
log leakage. Preventing collection is stronger than regex cleanup.

**Alternatives considered**:

- Redact only artifacts: rejected because prompts/logs remain exposed.
- Retain Playwright traces: rejected because traces capture data outside the
  sanitizer boundary.
- Trust synthetic data: rejected; credentials and canaries still exist.

## Decision 9: Version one evidence-first report contract

**Decision**: Emit `report.json`, `summary.md`, and a content-addressed
`evidence/` tree conforming to
`aurearia.browser-exploration-report/v1`. Store observed facts separately from
model-generated triage. Validate schema and privacy before artifact upload.

**Rationale**: A versioned machine contract supports manual/nightly parity,
trend analysis, deterministic tests, and quick human review.

**Alternatives considered**:

- HTML-only report: rejected as difficult to validate and aggregate.
- Raw runner logs: rejected as unstructured and unsafe.
- Database persistence: rejected as unnecessary for an advisory first slice.

## Decision 10: Detect the seeded defect deterministically

**Decision**: Inject a named test-only HTTP 500 at the Playwright route boundary
for a known F013 edit request. Use the real full stack and a fake schema-valid
model adapter. Deterministic evidence rules create the finding; model triage
only annotates it.

**Rationale**: This proves route, screenshot, network evidence, triage, and
reporting without intentionally breaking production code or depending on live
model prose. Running it 20 times directly measures SC-002.

**Alternatives considered**:

- Plant a production code defect: rejected as unsafe.
- Require a live model in acceptance tests: rejected as nondeterministic/costly.
- Exact-match model prose: rejected by the feature non-goals.

## Decision 11: Fingerprint issues with an exact hidden marker

**Decision**: Hash normalized workflow ID, route template, category, and
evidence signature. Put
`<!-- aurearia-ai-finding:v1:<fingerprint> -->` in a sanitized issue body and
search open and closed issues for the exact marker before creation.

**Rationale**: It is stable, deterministic, provider-independent, and avoids
duplicate issues even when titles change. Within-run dedup uses the same
fingerprint.

**Alternatives considered**:

- Title matching: rejected as unstable.
- Model semantic similarity: rejected as nondeterministic and expensive.
- Comment on duplicates: rejected as an unnecessary side effect.

## Decision 12: Separate exploration and issue permissions

**Decision**: Report-only is the default. Exploration runs with read-only
repository permission. A conditional publication job receives `issues: write`
only after configuration, schema, and privacy checks succeed; it handles no raw
browser/provider output and attempts at most three creates.

**Rationale**: This creates a least-privilege boundary and makes an accidental
issue from ordinary tests impossible.

**Alternatives considered**:

- Give the runner `issues: write`: rejected due to excess privilege.
- Publish inline before final report: rejected because privacy/schema checks
  would not yet be final.

## Decision 13: Keep CI advisory and independent

**Decision**: Create a distinct manual/nightly workflow with no `pull_request`,
`push`, `workflow_run`, required-check, or deployment dependency. Publish
artifacts with `if: always()`. Operational/privacy failure remains visible in
that workflow but cannot block delivery.

**Rationale**: This meets FR-006/FR-007 while gathering reliability evidence for
the separate promotion decision in FR-027.

**Alternatives considered**:

- Add a Quality Gate job with `continue-on-error`: rejected because it still
  couples the workflows and can be misconfigured as required.
- Run nightly against deployed beta: rejected by isolation requirements.

## Decision 14: Additive, migration-free rollback

**Decision**: Keep all runtime output in artifacts/ephemeral volumes and add no
production tables or settings. Rollback disables/removes the advisory workflow,
revokes dedicated secrets, and removes test-only tooling. Issue publication and
individual providers can be disabled independently.

**Rationale**: Existing F013 and Quality Gate remain the regression authority,
so rollback has no application-data step.

**Alternatives considered**:

- Persist run history in the application DB: rejected as avoidable migration
  and privacy retention.
- Production feature flags: rejected because no user-facing runtime feature is
  added.
