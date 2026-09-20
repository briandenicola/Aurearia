# Data Model: AI-Driven Browser Testing

This feature stores no new production records. The entities below are
in-memory objects and versioned artifact records. All identifiers are strings
and all timestamps are RFC 3339 UTC unless noted.

## 1. RunConfiguration

Validated input for one run. Secrets are explicitly absent.

| Field | Type | Rules |
|---|---|---|
| `schemaVersion` | literal | `aurearia.browser-exploration-config/v1` |
| `trigger` | enum | `local`, `manual`, or `nightly` |
| `provider` | enum | Initially `anthropic` or `ollama`; adapter registry controls support. |
| `model` | string | 1–120 characters; provider model identifier, never a credential. |
| `workflowScope` | string[] | Non-empty unique IDs from the F013 inventory. |
| `limits` | ExplorationLimits | Every value positive and no greater than its hard maximum. |
| `createIssues` | boolean | Required; defaults to `false` at input construction. |
| `seededDefect` | string/null | Test-only registered seed ID; null for ordinary runs. |

Validation rejects unknown fields, secrets, raw URLs (except an explicitly
allowlisted local Ollama endpoint resolved by the launcher), empty workflow
scope, and values above a maximum.

## 2. ExplorationLimits

Immutable snapshot created before the stack starts.

| Field | Type | Hard maximum |
|---|---:|---:|
| `steps` | integer | 25 |
| `wallTimeSeconds` | integer | 900 |
| `modelCalls` | integer | 20 |
| `modelTokens` | integer | 60,000 combined input/output |
| `browserActions` | integer | 100 |
| `issueAttempts` | integer | 3 |

The supplied value is both the configured limit and effective limit after
validation. Values are rejected rather than silently increased or interpreted.
The report stores the exact snapshot.

## 3. ExplorationUsage

Mutable counters owned exclusively by `BudgetGuard`, copied to the report.

| Field | Type | Meaning |
|---|---:|---|
| `steps` | integer | Reserved decision cycles |
| `modelCalls` | integer | Attempted calls |
| `modelInputTokens` | integer | Provider-reported input tokens |
| `modelOutputTokens` | integer | Provider-reported output tokens |
| `modelTokens` | integer | Input plus output |
| `browserActions` | integer | Attempted primitive actions |
| `issueAttempts` | integer | Attempted issue creates |
| `elapsedMilliseconds` | integer | Monotonic elapsed duration |

**Invariant**: no reported counter may exceed its corresponding snapshot. A
reservation that would exceed a limit is refused before work begins.

## 4. ExplorationRun

Aggregate root for one execution.

| Field | Type | Rules |
|---|---|---|
| `id` | string | `aibr_<UTC compact time>_<12 lowercase hex>` |
| `trigger` | enum | Mirrors configuration. |
| `provider` | ProviderDescriptor | Provider/model only; no credential data. |
| `workflowScope` | string[] | F013 workflow IDs selected at start. |
| `environment` | EnvironmentDescriptor | Ephemeral identifiers and source revision. |
| `startedAt`, `endedAt` | timestamp | `endedAt >= startedAt`. |
| `status` | enum | `completed`, `bounded`, `failed`, `privacy_failed`, `infrastructure_failed`. |
| `termination` | Termination | Required even for normal completion. |
| `limits` | ExplorationLimits | Immutable snapshot. |
| `usage` | ExplorationUsage | Final counters. |
| `steps` | ExplorationStep[] | Ordered by index. |
| `routeHistory` | RouteVisit[] | Sanitized same-origin route templates. |
| `evidence` | BrowserEvidence[] | Sanitized evidence metadata. |
| `findings` | ExplorationFinding[] | Deduplicated findings. |
| `privacyCheck` | PrivacyCheck | Required before publication/upload. |
| `publication` | PublicationSummary | Present even when disabled. |

### State transitions

```text
created -> provisioning -> ready -> exploring
exploring -> finalizing
provisioning/ready/exploring -> finalizing (on error or bound)
finalizing -> completed | bounded | failed | privacy_failed | infrastructure_failed
```

No terminal state transitions back to an active state.

## 5. EnvironmentDescriptor

| Field | Type | Rules |
|---|---|---|
| `kind` | literal | `ephemeral-full-stack` |
| `sourceRevision` | string | Full Git commit SHA being tested. |
| `composeProject` | string | Generated run-local name; contains no secret. |
| `appOrigin` | string | Loopback HTTP origin only; sanitized before report. |
| `services` | object | App/API/agent readiness outcomes and image IDs. |
| `fixtureSet` | literal | `feature-220-f013-golden-collection` |
| `testUserId` | string | Synthetic opaque ID; no username/password. |
| `productionDataUsed` | literal | Always `false`; validation fails otherwise. |

## 6. ExplorationStep

One model-directed cycle.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Stable within run, e.g. `step-001`. |
| `index` | integer | Starts at 1 and is strictly increasing. |
| `workflowId` | string | Member of selected scope. |
| `goal` | string | Sanitized, bounded text. |
| `observationEvidenceIds` | string[] | References existing evidence. |
| `decision` | ModelDecision/null | Null when model call failed. |
| `actionResults` | ActionResult[] | Results of allowlisted actions. |
| `usage` | ModelUsage/null | Provider-reported usage for this call. |
| `startedAt`, `endedAt` | timestamp | Ordered. |

## 7. ModelDecision

Strict response from the agent service.

| Field | Type | Rules |
|---|---|---|
| `action` | enum | `navigate`, `click`, `fill`, `select`, `upload_fixture`, `set_viewport`, `back`, `wait_for_ui`, `checkpoint`, `finish` |
| `target` | object/null | Action-specific, typed locator/route/fixture parameters. |
| `rationale` | string | Model-generated, max 1,000 chars, never persisted as fact. |
| `suspectedFindings` | ModelTriage[] | Interpretations referencing evidence IDs only. |
| `usage` | ModelUsage | Required and non-negative. |

Unknown actions/fields, external URLs, script content, filesystem paths, and
unreferenced evidence fail validation and cause safe termination.

## 8. BrowserEvidence

Discriminated union whose body has already crossed the sanitizer boundary.

### Common fields

| Field | Type | Rules |
|---|---|---|
| `id` | string | `ev_<kind>_<sequence>`. |
| `kind` | enum | `route`, `screenshot`, `console`, `network`, `accessibility`, `ui`. |
| `capturedAt` | timestamp | UTC. |
| `workflowId` | string | Current workflow. |
| `route` | string | Same-origin route template; query values removed/allowlisted. |
| `summary` | string | Sanitized and length-bounded. |
| `sha256` | string | 64 lowercase hex over persisted content/metadata. |
| `artifactPath` | string/null | Relative path under `evidence/`; no traversal. |
| `truncated` | boolean | Whether source exceeded its cap. |

### Kind-specific fields

- `screenshot`: viewport, masked selector count, PNG dimensions; no OCR/raw DOM.
- `console`: level and sanitized message; errors only by default.
- `network`: method, route template, status/error class, resource type, timing;
  never headers or raw body.
- `accessibility`: rule/check ID, impact, bounded target description, and
  sanitized message.
- `route`: from/to route templates and navigation reason.
- `ui`: bounded visible-state assertion/observation.

## 9. ExplorationFinding

| Field | Type | Rules |
|---|---|---|
| `id` | string | Stable within run, `finding-001` sequence after sort. |
| `fingerprint` | string | SHA-256 of normalized workflow, route, category, evidence signature. |
| `category` | enum | `functional`, `console`, `network`, `accessibility`, `visual`, `navigation`, `performance`, `security`, `other`. |
| `severity` | enum | `critical`, `high`, `medium`, `low`, `info`. |
| `confidence` | number | 0–1 inclusive. |
| `workflowId` | string | F013 workflow ID. |
| `route` | string | Sanitized route template. |
| `title` | string | Concise and sanitized. |
| `reproductionSteps` | string[] | 1–12 sanitized steps. |
| `expectedBehavior` | string | Sanitized, bounded. |
| `observedBehavior` | string | Sanitized, bounded and evidence-grounded. |
| `evidenceIds` | string[] | Non-empty references to existing evidence. |
| `observedFacts` | string[] | Deterministic facts derived from evidence. |
| `modelTriage` | ModelTriage/null | Explicitly labeled interpretation. |
| `publication` | FindingPublication | Publication outcome. |

**Invariant**: a finding cannot exist without evidence. Model triage cannot add
an evidence reference or observed fact that is absent from the run.

## 10. ModelTriage

| Field | Type | Rules |
|---|---|---|
| `generatedByModel` | literal | `true` |
| `summary` | string | Sanitized interpretation. |
| `suggestedCategory` | category enum | Advisory. |
| `suggestedSeverity` | severity enum | Advisory; deterministic policy may cap it. |
| `confidence` | number | 0–1 inclusive. |
| `evidenceIds` | string[] | Existing IDs only. |

## 11. PrivacyCheck

| Field | Type | Rules |
|---|---|---|
| `status` | enum | `passed` or `failed`. |
| `checkedChannels` | string[] | Must contain prompts, screenshots, report, traces, logs, and proposed issues. |
| `canaryCount` | integer | Number of registered canaries, never their values. |
| `violations` | PrivacyViolation[] | Channel/path/rule only; never leaked content. |
| `checkedAt` | timestamp | After report staging, before publication/upload. |

If failed, run status becomes `privacy_failed`, all issue requests become
`blocked_privacy`, and the artifact is restricted to a minimal diagnostic
manifest that contains no offending content.

## 12. IssuePublicationRequest

Derived only after report and privacy validation.

| Field | Type | Rules |
|---|---|---|
| `schemaVersion` | literal | `aurearia.browser-exploration-issue/v1` |
| `runId` | string | Source run. |
| `findingId` | string | Source finding. |
| `fingerprint` | string | Finding fingerprint. |
| `repository` | string | Exact `owner/name` allowlist. |
| `title` | string | Sanitized, max 160 chars. |
| `body` | string | Sanitized reproduction, expected/observed, severity, run/evidence refs, hidden marker. |
| `labels` | string[] | Fixed allowlist only. |

### FindingPublication states

`not_requested`, `not_actionable`, `queued`, `duplicate`, `created`, `failed`,
`blocked_limit`, or `blocked_privacy`.

## 13. Termination

| Field | Type | Rules |
|---|---|---|
| `reason` | enum | `completed`, `step_limit`, `wall_time_limit`, `model_call_limit`, `model_token_limit`, `browser_action_limit`, `issue_attempt_limit`, `provider_unavailable`, `provider_usage_invalid`, `model_output_invalid`, `browser_failure`, `stack_failure`, `privacy_failure`, `cancelled`, `internal_error`. |
| `reachedLimits` | string[] | Every simultaneously reached limit. |
| `message` | string | Sanitized operator-facing summary. |
| `lastCompletedStepId` | string/null | Existing step only. |

## Relationships

```text
RunConfiguration 1 ──creates── 1 ExplorationRun
ExplorationRun    1 ──snapshots 1 ExplorationLimits
ExplorationRun    1 ──records── 1 ExplorationUsage
ExplorationRun    1 ──contains─ * ExplorationStep
ExplorationRun    1 ──contains─ * BrowserEvidence
ExplorationRun    1 ──contains─ * ExplorationFinding
ExplorationStep   * ──references * BrowserEvidence
ExplorationFinding * ──references 1..* BrowserEvidence
ExplorationFinding 1 ──produces 0..1 IssuePublicationRequest
```

Referential validation occurs before a report is considered valid.
