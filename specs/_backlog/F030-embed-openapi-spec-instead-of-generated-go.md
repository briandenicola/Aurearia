---
id: F030
title: "Embed the OpenAPI spec instead of generating 24k lines of Go"
status: backlog          # backlog | triaged | promoted | dropped
priority: P3             # P0 (now) | P1 (next) | P2 (soon) | P3 (someday)
effort: S                # XS | S | M | L | XL
value: 2                 # 1 (low) – 5 (high) user/business value
risk: 2                  # 1 (low) – 5 (high) implementation / regression risk
owner: unassigned        # squad member or human handle
created: 2026-08-20
updated: 2026-08-20
---

# F030 — Embed the OpenAPI spec instead of generating 24k lines of Go

## Summary

`src/api/docs/docs.go` is a swaggo-generated file holding the entire Swagger 2.0
document as a single Go raw-string literal. At 24,311 lines it is 16% of all Go
in the repository and larger than any hand-written package — it distorts every
LOC measurement, and any change to a route annotation produces a diff of
hundreds to thousands of lines that no reviewer can meaningfully read.

Nothing about the current setup is broken: the file is committed on purpose, CI
regenerates it and fails on drift, and compiling the spec into the binary is what
lets the Alpine image serve `/swagger` as a single static binary with no docs
directory on disk. This card is about keeping all of those properties while
storing the spec as JSON that tools can diff, rather than as Go source.

"Done" looks like: `task openapi` emits `swagger.json`/`swagger.yaml` only, a
small hand-written `docs.go` embeds the JSON via `//go:embed` and registers it
with `swag.Register`, the CI drift gate still fails when a route and its
annotation disagree, and `/swagger` renders identically — including the correct
version string.

## Acceptance criteria

- [ ] `src/api/docs/docs.go` is hand-written and under ~40 lines; the generated
      artifact committed to the repo is `swagger.json` (+ `swagger.yaml`).
- [ ] `swag init` is invoked with `--outputTypes json,yaml` in both `Taskfile.yml`
      and `.github/workflows/ci.yml`.
- [ ] The CI "Verify OpenAPI snapshot" step still fails when a handler
      annotation changes without the artifact being regenerated (verify by
      deliberately editing one `@Summary` and confirming a red build).
- [ ] `GET /swagger/index.html` renders the full spec, and the displayed version
      matches the root `VERSION` file for a `docker run` of the release image.
- [ ] `docs/openapi.json` at the repo root is still produced and in sync.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./...` pass, including
      `route_openapi_drift_test.go` and `openapi_nullability_test.go`.

## Constitution alignment

- Principle I (Clear Layered Architecture) — the `docs` package stays a leaf
  with no new dependencies; only its representation changes.
- §17 Quality Gate — the OpenAPI drift gate is the load-bearing property here
  and must survive the change unweakened.
- §21 Definition of Done — regenerate-and-commit remains part of the workflow;
  only the committed artifact's format changes.

## Open questions

- [ ] Is the runtime version override in `main.go:108`
      (`docs.SwaggerInfo.Version = loadAppVersion()`) still needed? `task openapi`
      already rewrites the `//	@version` annotation from `VERSION` *before*
      running swag, so the value is baked correctly at generation time. If the
      override is redundant, embedding plain JSON is trivial. If it must stay
      (e.g. for `go run .` from a working tree where the artifact is stale), the
      embedded document needs to keep `{{.Version}}` as a placeholder, which
      means committing a template rather than the literal `swagger.json` — and
      that in turn means the committed file is no longer a valid OpenAPI
      document that external tools can consume directly.
- [ ] Does `swag` guarantee byte-stable JSON output across runs and platforms?
      The drift gate compares with `git diff --exit-code`, so any map-ordering
      instability would produce false failures. Worth confirming against the
      pinned `swag@v1.16.6` on both Linux CI and a Windows dev box before
      committing to the approach.
- [ ] Should `swagger.json` be marked `linguist-generated=true` in
      `.gitattributes` so GitHub collapses it in PR review? That alone captures
      much of the review-noise benefit and is a near-zero-risk change that could
      ship independently of this card.

## Notes

Discovered while measuring the Go comment-to-code ratio during the 2026-08-20
codebase review. Headline figure was "150k lines of Go"; the real shape is
50,664 lines of hand-written production code, 52,481 lines of tests, and 24,311
lines of generated `docs.go`.

Current wiring, for whoever picks this up:

- `docs.go` defines `docTemplate` (the spec, with `{{.Version}}`/`{{.Host}}`
  Go-template placeholders), a `SwaggerInfo *swag.Spec` referencing it, and an
  `init()` calling `swag.Register`.
- `main.go` imports the package for `docs.SwaggerInfo`, which triggers that
  `init()`; `bootstrap.go` serves `/swagger/*any` via
  `ginSwagger.WrapHandler(swaggerFiles.Handler)`, which reads the registry.
- `Taskfile.yml` `openapi:` regenerates and copies `docs/swagger.json` to
  `docs/openapi.json` at the repo root.

Cheapest partial win, if the full change is not worth it: the
`linguist-generated` attribute in the third open question. It is one line and
solves the review-noise half of the problem without touching the build.

## History

- 2026-08-20: created (status: backlog)
