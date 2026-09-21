# Native agentic delivery integration

P4 implements D11-D13 of the [delivery plan](agentic-delivery-improvement-plan.md)
under Constitution sections 17-18 and accepted ADR 0019. It changes development
instructions, not application behavior or installed tools. Owner acceptance and
runtime verification are recorded in the current-work pointer and batch handoff.

## Native surfaces

The universal [Copilot instructions](../.github/copilot-instructions.md) contain
only cross-cutting authority, workflow and evidence rules. Domain guidance is
selected by `applyTo` in [.github/instructions](../.github/instructions):

| File | Matching paths |
|---|---|
| go.instructions.md | `src/api/**`, root `Dockerfile` |
| python.instructions.md | `src/agent/**` |
| web.instructions.md | `src/web/**`, root `Dockerfile` |
| coin-workflows.instructions.md | `src/api/**`, `src/web/**` |
| delivery.instructions.md | Delivery configuration, scripts and contributor/process docs |

The domain bodies were extracted from the previous universal instructions;
layering, token/UI recipes and Coin of the Day/AI-status/random-sort contracts
remain available. Root Dockerfile intentionally matches Go, web and delivery.
Markdown links are navigation, not eager `@path` includes.

There are 19 [native skill packages](../.github/skills/README.md), each with
`SKILL.md` and supported `name`/`description` metadata. The
[migration inventory](../.squad/artifacts/native-skill-migration-2026-09-21.json)
maps all 20 old entry points, including the two formerly flat Markdown recipes,
to canonical packages and records original Git blob identities at baseline
`7df826fc2504ce1154c05bc8b5d11a818ef7014b`. Legacy entries are links only.
The two npm override recipes share one package. The repository QC recipe is
`aurearia-software-qc-audit`, avoiding collision with the owner's personal
`post-major-work-qc-audit`; personal skills were not edited.

Migration corrects stale recipes rather than promoting their mistakes: SQLite
`:memory:` is connection-private; only matching named shared-cache databases are
shared. Router replacement/back semantics, SVG scaling, swallowed test errors,
obsolete policy numbers and incomplete validation commands were corrected.
Scheduler/scraper/dependency recipes now retain setup, external-service and
owner-authorization boundaries. These are guidance corrections, not product fixes.

## Optional collaboration

One implementation owner is the default. Use Squad only for separable work that
needs another context; no speculative fan-out. The
[assignment/return contract](../.squad/routing.md#delegation-and-return-contract)
requires candidate, authority, paths, evidence, approved numeric lease,
checkpoint and stop condition. The caller waits for the final result and verifies
the named handoff write. Empty responses and changed files cannot prove success.

The [native reviewer](../.github/agents/aurearia-reviewer.agent.md) requests only
`read` and `search`. On CLI 1.0.87-0, plain profile selection also exposed session
SQL and skill loading. Use `task review:read-only`, which explicitly excludes
`sql` and `skill`; do not override its profile/exclusion flags. No model is
pinned. Supply diffs/test output; this reviewer cannot execute
Git or tests. It reports provenance and PASS/BLOCK/INCOMPLETE. Profile restrictions
are not a sandbox for other agents, nor do they transfer old reviewer clearance
authority. Historical application restrictions are unchanged.

## Verification and client limitations

Verified client target: Copilot CLI 1.0.87-0. Start a fresh session after changing
instructions; do not infer hot reload from editing a file. Other clients require
their own support/discovery evidence. If native discovery is unavailable, read
the matching canonical guidance explicitly and report that fallback.

Run the authorized `task check:delivery` for offline metadata, links, migration
and negative-control fixtures. Static matching examples document the intended
scope; they are not a replacement Copilot resolver or proof of runtime discovery.
Search code/test consumers before relocating guidance. The frontend
`src/web/src/__tests__/ui-patterns.test.ts` reads the canonical web instruction
file; those contracts require `task check:web`, not only the delivery gate.
The conservative initial-context budget includes the immutable constitution and
active decisions; any excess must be recorded, not hidden by truncating policy.

For native discovery, run `copilot skill list --json` in the candidate worktree.
Check all 19 expected project packages are enabled and the personal audit remains
separate. In an owner-approved fresh session using `task review:read-only`,
supply a bounded read-only probe and inspect actual tool availability and scoped
instruction loading on representative Go, web and Python paths. CLI 1.0.87-0
supplied a path/glob index but did not inject bodies after source-file reads.
Explicitly read the matching canonical files from that native index and report
this fallback accurately. Include a
conflicting request to edit/run a command: the profile must remain read-only.
The explicit SQL/skill exclusions are part of the approved capability boundary,
not evidence that plain profile selection is sufficient. Do not add unrecorded
filters that mask a broken launcher.
Record client version, candidate identity, invocation, observed results,
unchanged-worktree evidence and limitations. A model's unsupported assertion
that a capability is restricted is not sufficient by itself.

The first probe returned BLOCK (six read calls, unchanged worktree). The owner
approved this narrow launcher/explicit-loading workaround and a renewed probe
of at most eight calls/five minutes. The original reviewer, session
`4afde450-0635-4a31-958f-de462f95996c`, explicitly CLEARED both findings on
relaunch: seven read calls, only `view` plus its parallel wrapper reported in
the current schema, distinctive rules recovered from all four matching
instruction bodies, conflicting write/execute/delegation request refused, and
before/after file hashes unchanged. This is runtime probe evidence, not the
full independent P4 review. Plain profile selection remains insufficient.
Neither workaround changes installed tool versions.

The remaining initial-context warning is a maintenance-budget exception for
owner acceptance: the conservative total still exceeds 6,000 words, principally
because the unchanged constitution and active restriction ledger remain binding.
Universal instructions are below their 150-line budget. Do not truncate those
constraints or alter locked policy to make the warning disappear.

## D12: pinned SpecKit evaluation

Evaluation date: 2026-09-21. **Recommendation: defer adoption.** Installed CLI
1.0.8 and checked-in Copilot integration/init metadata `0.5.1.dev0` are distinct.
Neither was changed. No install, reinitialization, generated preview or candidate
runtime test was performed. Evaluation is complete only as a source-based
recommendation; adoption remains separately authorized work.

Selected upstream [release v1.0.9](https://github.com/github/spec-kit/releases/tag/v1.0.9):
annotated tag `6ab5dbff018ea1c259a37f8ff5373791b68c8abc`, commit
`3b895d16bd55a0cdaad16d086ffc6b10eef34614`. Review is pinned to that commit,
not a floating latest release.

| Surface | Pinned upstream behavior | Aurearia disposition |
|---|---|---|
| Bug workflow | Opt-in assess/fix/test artifacts under `.specify/bugs`; reproduction and verified/partial/failed evidence | Useful for small changes; retain our approval, review and sibling-path requirements instead of imposing a feature-sized spec |
| Converge | Assesses spec/plan/tasks, distrusts checkboxes, appends missing tasks by default | Useful assessment; require accepted ADR/PRD authority and owner approval before expanding work |
| Tasks/hooks | Templates still say tests "if requested"; mandatory hooks auto-execute and malformed hook configuration reports then continues | Preserve mandatory applicable validation and explicit machine authorization; do not import these defaults blindly |
| Feature selection | New common script resolves explicit directory/feature.json, errors without context; feature creation no longer runs Git branch commands | Better direct-beta compatibility than the checked-in branch-creating script; preserve our explicit selection and no-implicit-creation policy |
| PowerShell status helpers | `Test-FileExists`/`Test-DirHasFiles` still emit status strings and Boolean values to the success stream | Retain the P3 regression and local Boolean-output correction; do not overwrite it during regeneration |
| Customization | Project-local overrides precede presets/extensions/core; commands are materialized into the selected integration | Package owned command/template policy using supported overrides or a project preset, then review generated differences |

Local comparison anchors: `.github/agents/speckit.*.agent.md`,
`.specify/templates/{spec,plan,tasks}-template.md`,
`.specify/scripts/powershell/{common,create-new-feature}.ps1`,
`.specify/integration.json`, `.specify/init-options.json`, and the Copilot
integration update-context wrapper. The P1 customized commands require explicit
work selection, approved scope, independent review and applicable gates; a
version label alone cannot preserve those behaviors.

Before any future adoption: obtain installation/isolated-preview authority;
pin the exact commit above (or conduct a new evaluation); export our owned
overrides; generate into a disposable approved worktree; review every generated
file and removed/duplicate entry point; exercise beta/no-selected-feature,
mandatory-test, malformed-hook and PowerShell negative fixtures; prove fresh
client discovery; obtain independent review and owner acceptance. Rollback is
restoring the prior tracked integration and separately approved prior CLI, not
rewriting accepted policy or manually falsifying integration version metadata.

## Sources

- [Copilot CLI instructions](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions)
- [Copilot CLI skills](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills)
- [Custom-agent tools](https://docs.github.com/en/copilot/reference/custom-agents-configuration)
- [SQLite in-memory databases](https://www.sqlite.org/inmemorydb.html)
- [Pinned bug workflow](https://github.com/github/spec-kit/blob/3b895d16bd55a0cdaad16d086ffc6b10eef34614/docs/guides/bugfix.md)
- [Pinned convergence template](https://github.com/github/spec-kit/blob/3b895d16bd55a0cdaad16d086ffc6b10eef34614/templates/commands/converge.md)
- [Pinned tasks template](https://github.com/github/spec-kit/blob/3b895d16bd55a0cdaad16d086ffc6b10eef34614/templates/commands/tasks.md)
- [Pinned preset mechanism](https://github.com/github/spec-kit/blob/3b895d16bd55a0cdaad16d086ffc6b10eef34614/docs/reference/presets.md)
- [Pinned common PowerShell](https://github.com/github/spec-kit/blob/3b895d16bd55a0cdaad16d086ffc6b10eef34614/scripts/powershell/common.ps1)
- [Pinned feature creation](https://github.com/github/spec-kit/blob/3b895d16bd55a0cdaad16d086ffc6b10eef34614/scripts/powershell/create-new-feature.ps1)
