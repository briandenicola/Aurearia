# Quickstart: AI-Driven Browser Testing

This is an implementation contract for the commands Feature 360 will add. The
feature is not implemented by this planning change.

## Prerequisites

- Docker Engine with Compose v2
- Task 3.x
- Node.js `20.19.0`
- Dependencies installed with `npm ci` under `src/web`
- Chromium installed for Playwright `1.63.0`
- One dedicated exploration-provider credential; never reuse an application
  user's production provider setting
- No production `.env`, database, upload volume, or deployed URL

The launcher rejects non-loopback browser origins and externally supplied
database paths.

## Credentials

Set only the variables for the selected provider:

```powershell
# Dedicated Anthropic test credential
$env:AI_BROWSER_ANTHROPIC_API_KEY = '<protected-test-key>'

# Or an explicitly allowed test-only Ollama endpoint
$env:AI_BROWSER_OLLAMA_URL = 'http://host.docker.internal:11434'
```

The launcher generates these per run and does not read them from a checked-in
file:

- `AI_BROWSER_TEST_USERNAME`
- `AI_BROWSER_TEST_PASSWORD`
- `JWT_SECRET`
- `AGENT_INTERNAL_SERVICE_TOKEN`

For opt-in issue publication in a manual run only:

```powershell
$env:AI_BROWSER_GITHUB_TOKEN = '<fine-grained-token-for-this-repository>'
```

The exploration process itself must not receive the GitHub token. Only the
post-validation publisher process receives it.

## Create a run configuration

Save as an ignored local file such as `.tmp/ai-browser/run-config.json`:

```json
{
  "schemaVersion": "aurearia.browser-exploration-config/v1",
  "trigger": "local",
  "provider": "anthropic",
  "model": "claude-sonnet-5",
  "workflowScope": [
    "login-session",
    "add-coin",
    "edit-one-field",
    "storage-location-change-clear",
    "tags-sets-edit",
    "image-upload-delete",
    "collection-search-filter",
    "mobile-edit"
  ],
  "limits": {
    "steps": 25,
    "wallTimeSeconds": 900,
    "modelCalls": 20,
    "modelTokens": 60000,
    "browserActions": 100,
    "issueAttempts": 3
  },
  "createIssues": false,
  "seededDefect": null
}
```

Use lower values for a smoke run. Values above a hard maximum are rejected.

## Run locally

From the repository root:

```powershell
task ai-browser-exploration -- --config .\.tmp\ai-browser\run-config.json
```

The command will:

1. Validate configuration and snapshot limits.
2. Build app and agent images from the current checkout.
3. Create a unique Compose project and fresh SQLite/upload volumes.
4. Generate a dedicated user and seed Feature 220/F013 golden fixtures.
5. Wait for app `/healthz` and agent `/ready`.
6. Run one Chromium exploration within all limits.
7. Validate and privacy-scan the report.
8. Tear down containers, network, and volumes even after failure.

Expected output:

```text
.artifacts/ai-browser/<run-id>/
├── report.json
├── summary.md
├── prompts/                 # sanitized request/response records only
└── evidence/
    ├── screenshots/
    ├── console/
    ├── network/
    └── accessibility/
```

Ordinary Playwright trace, video, and HAR files are disabled.

## Run deterministic validation

No live model or real GitHub issue is used:

```powershell
task test-ai-browser-exploration
task test-ai-browser-seeded-defect
```

The first command covers schemas, exact budget boundaries, malformed provider
responses, token-accounting failure, action allowlisting, redaction/canary
scans, finding deduplication, report-only default, and fake issue publication.

The seeded-defect command runs the same injected network defect 20 times. Every
run must detect it and link route history, a masked screenshot, and a relevant
network observation. The fake model supplies structured triage but cannot
create evidence. The fake publisher records a sanitized request and performs
no external call.

Confirm F013 remains authoritative:

```powershell
task test-critical-workflows
```

## Explicit issue opt-in

Change only:

```json
"createIssues": true
```

The publisher runs only after the report schema and privacy check pass. It:

1. Selects actionable findings in deterministic order.
2. Searches open and closed issues for the exact fingerprint marker.
3. Marks duplicates without creating or commenting.
4. Attempts no more than the configured limit (maximum 3).
5. Keeps every non-published finding in `report.json`.

Omitting `createIssues` at the workflow input layer or supplying `false`
results in zero issue calls. Tests always use the fake publisher.

## Manual CI

Run the planned **AI Browser Exploration** workflow from GitHub Actions:

1. Choose provider and model.
2. Choose an F013 workflow scope or the baseline scope.
3. Optionally lower limits.
4. Leave **Create issues** false unless publication is intentionally required.

The job uses protected provider secrets, builds a fresh stack, and uploads the
same report shape as the local command. It is not a required check.

## Nightly CI

The nightly schedule calls the same Taskfile target and default configuration.
It starts with a new stack and publishes a finite-retention artifact using an
immutable-SHA-pinned upload action. Findings and infrastructure failures are
visible in that workflow only; they do not affect Quality Gate or deployment.

Promotion to blocking is out of scope until a separate decision verifies at
least 20 consecutive nightly runs over at least 14 days meet every FR-027
threshold.

## Report review

Open `summary.md`, then follow finding links into `report.json` and `evidence/`.
For each finding verify:

- workflow and route;
- reproduction steps;
- expected versus observed behavior;
- deterministic observed facts;
- evidence references;
- separately labeled model triage;
- publication state.

`report.json` must validate against
`contracts/exploration-report.schema.json`.

## Troubleshooting

| Symptom | Expected safe behavior |
|---|---|
| Provider unavailable/rate-limited | Stop further calls, finalize evidence, report provider termination. |
| Missing/malformed usage | Stop before another model call with `provider_usage_invalid`. |
| App/agent readiness failure | Mark `infrastructure_failed`, publish safe summary if possible, tear down. |
| Login failure | Record sanitized UI evidence; do not expose credentials. |
| Navigation loop/no progress | Stop at budget/no-progress policy and preserve evidence. |
| Canary detected | Mark `privacy_failed`, block issue publication, do not upload offending payload. |
| Issue API unavailable | Preserve finding, record failed publication, obey three-attempt maximum. |

## Full quality gate for implementation

```powershell
# Repository root
task test
task test-race
task build
task test-agent
task lint-agent
task test-critical-workflows
task test-ai-browser-exploration
task test-ai-browser-seeded-defect
```

Also verify the report schemas and the Go/Python exploration contract have no
drift.

## Cleanup and rollback

Normal and failed runs execute:

```powershell
docker compose -f docker-compose.exploration.yml -p <run-project> down -v --remove-orphans
```

Operational rollback is to disable the advisory workflow, revoke dedicated
provider/issue secrets, remove the additive exploration targets, and expire
artifacts. There is no production database migration or data restoration.
