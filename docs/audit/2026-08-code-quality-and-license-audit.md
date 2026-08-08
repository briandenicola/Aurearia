# Code Quality & Open Source License Audit

**Date:** 2026-08-08
**Commit audited:** `7ee6941` (main)
**Scope:** `src/api` (Go), `src/web` (Vue/TS), `src/agent` (Python), build/CI/packaging
**Method:** static inspection plus live execution — all three test suites, ESLint, Ruff, `go vet`, `gofmt`, Go coverage profiling, a full production `vite build`, and license extraction from the resolved Go module cache, `package-lock.json`, and `uv.lock`.

---

## Executive summary

Aurearia is a large, unusually disciplined codebase for a solo project: ~99k lines of production code, ~50k lines of tests, all suites green (617 frontend, 189 agent, all Go packages), zero `any` in TypeScript, an enforced architectural layering test, and a hardened security-scan pipeline. The problems found are not sloppiness — they are **gaps between what the project documents and what it enforces**, plus one genuine legal conflict.

| Area | Verdict |
|---|---|
| Open source licensing — Go deps | ✅ Clean (100% MIT/BSD/Apache-2.0, 63 modules) |
| Open source licensing — Python deps | ✅ Clean (97 packages, all permissive) |
| Open source licensing — npm deps | ❌ **One AGPL-3.0 dependency shipped in an MIT product** |
| Attribution / notice compliance | ⚠️ No third-party notices in any distributed artifact |
| Code quality — structure | ⚠️ Two god-objects; documented modularity policy has drifted |
| Code quality — correctness tooling | ⚠️ Linter config exists but is unwired **and** unloadable |
| Code quality — test coverage | ⚠️ 40.5% Go; the social/privacy surface is at **0%** |
| Code quality — build output | ⚠️ 81% of the shipped bundle is one duplicated WASM blob |
| Code quality — docs accuracy | ⚠️ Version, changelog, and module-size docs all stale |

**One finding requires action before the next public image push: [L1](#l1--critical--agpl-30-dependency-shipped-in-an-mit-licensed-product).**

### Codebase at a glance

| Component | Production LOC | Test LOC | Test suite |
|---|---:|---:|---|
| Go API (`src/api`) | 51,653 | 34,448 | 126 test files, all passing |
| Vue web (`src/web`) | 40,840 | 13,068 + 708 e2e | 102 files / 617 tests, all passing |
| Python agent (`src/agent`) | 6,533 | 2,904 | 21 files / 189 tests, all passing |
| Generated (`docs/docs.go`) | 21,557 | — | swaggo output, committed |
| Markdown documentation | 63,331 | — | 407 files |

---

# Part 1 — Open source license audit

## Dependency license inventory

**Go — 61 modules in the build graph, extracted from the resolved module cache.** No copyleft anywhere.

| License | Count |
|---|---:|
| MIT | 31 |
| BSD-3-Clause | 20 |
| Apache-2.0 | 10 |

**Python — 97 locked packages** (`uv.lock`), including the `pip-audit`/`cyclonedx` dev tree. All MIT / BSD / Apache-2.0 / ISC. No LGPL or GPL transitives (notably, `charset-normalizer` is used rather than the LGPL `chardet`).

**npm — 756 packages** (`package-lock.json`):

| License | Count | Note |
|---|---:|---|
| MIT | 626 | |
| ISC | 33 | |
| Apache-2.0 | 27 | |
| BSD-3-Clause / BSD-2-Clause | 34 | |
| BlueOak-1.0.0 | 12 | permissive |
| MPL-2.0 | 12 | all `lightningcss` build binaries — not shipped in `dist/` |
| `(MPL-2.0 OR Apache-2.0)` | 1 | `dompurify` — Apache-2.0 may be elected |
| CC-BY-4.0 / CC0-1.0 / Python-2.0 / MIT-0 | 5 | dev-only data packages |
| **`SEE LICENSE IN LICENSE.md`** | **1** | **`@imgly/background-removal` — see L1** |

---

## L1 — CRITICAL — AGPL-3.0 dependency shipped in an MIT-licensed product

**Package:** `@imgly/background-removal@1.7.0` (a production `dependency`)
**Actual license:** **GNU Affero General Public License v3.0**, or a paid commercial license from IMG.LY.

The npm metadata only says `SEE LICENSE IN LICENSE.md`, which is why no automated tool in this repo has ever flagged it. Retrieving the upstream file resolves it:

- `LICENSE.md` in `imgly/background-removal-js` is the verbatim AGPL-3.0 text.
- The project README states: *"The software is free for use under the AGPL License. Please contact support@img.ly for questions about other licensing options."*

### This is not a theoretical exposure — the code is in the shipped artifact

Verified by running a real production build:

1. `src/web/src/utils/backgroundRemoval.ts` statically imports `removeBackground` from the package.
2. That util is statically imported by `useImageProcessor.ts` and `ImageLightbox.vue`.
3. `npm run build` emits **`dist/assets/backgroundRemoval-CDxFGOPx.js` (82.47 kB)** containing the AGPL code, and it appears in the Workbox precache manifest.
4. `Dockerfile:13` runs `npm run prepare-background-removal-assets`, which downloads IMG.LY's model/runtime data into the image.
5. `.github/workflows/docker-publish.yml` pushes `${DOCKERHUB_USERNAME}/ancient-coins` to Docker Hub with `push: true`.

So the AGPL code is **conveyed** (published image, served to browsers) and **used over a network** (AGPL §13). Both triggers fire.

### Why this conflicts with the repo's own LICENSE

`LICENSE` is MIT. AGPL-3.0 §5(c) requires the entire combined work to be licensed under AGPL-3.0 to anyone who receives it. Offering the combined work under MIT terms grants recipients rights — sublicensing, proprietary redistribution — that AGPL-3.0 forbids. An MIT grant over an AGPL-derived work is not a grant the project is in a position to make.

Additionally, AGPL §13 obliges any operator of a modified version reachable over a network to offer the Corresponding Source to remote users. Every self-hoster who deploys this image inherits that obligation — silently, because nothing in the README or `docs/deployment.md` mentions it.

> **A note on the "self-hosted, personal use" defence.** Purely private use — never conveying the software to anyone — genuinely does not trigger the AGPL. But this project publishes public Docker images, documents public-facing deployment, and ships social/showcase features designed to be reached by other people. That is conveying and remote network interaction. The defence does not apply here.

### Remediation options, in order of preference

1. **Replace the dependency.** Background removal is one feature (`removeCoinBackground`, one call site behind a composable — a clean seam). Permissively-licensed options exist: run the ISNet/U²-Net ONNX model directly through `onnxruntime-web` (MIT) — the project **already self-hosts that exact model and runtime** under `public/imgly-background-removal/`, so this is mostly re-implementing pre/post-processing, not adding infrastructure. This preserves the MIT license and is the recommended path.
2. **Relicense the project to AGPL-3.0.** Legally clean and cheap to execute, but changes the terms for every existing user and every downstream fork.
3. **Buy a commercial license from IMG.LY** (`support@img.ly`) permitting MIT redistribution.
4. **Move it server-side or make it optional** — *does not fix this on its own.* AGPL specifically closes the "run it on a server" loophole, and an optional-but-bundled dependency is still conveyed.

Whichever path is chosen, do it before the next image push, and add an SPDX-based license gate (see [L4](#l4--medium--no-license-gate-in-ci)) so this class of dependency can never land unnoticed again.

---

## L2 — HIGH — No third-party attribution notices in any distributed artifact

`git ls-files` finds no `NOTICE`, `THIRD_PARTY_LICENSES`, or equivalent anywhere in the repository, and neither Dockerfile copies license texts into the final image.

MIT, BSD-2-Clause, and BSD-3-Clause all require that the copyright notice and permission text be reproduced **in binary distributions**, not just in source. The published images distribute:

- 61 Go modules compiled into `ancient-coins-api` (all MIT/BSD/Apache-2.0),
- the entire npm production dependency tree, minified into `dist/assets/*.js`,
- the resolved Python runtime tree inside `/app/.venv` in the agent image.

None of their notices travel with the artifact. This is the single most common OSS compliance gap in container-based distribution, and it is straightforward to fix.

**Remediation.** Generate notices at build time and `COPY` them into both images:

```dockerfile
# api-build stage
RUN go install github.com/google/go-licenses@latest \
 && go-licenses save ./... --save_path=/licenses/go

# web-build stage
RUN npx --yes license-checker-rseidelsohn --production \
        --files /licenses/npm --out /licenses/npm-summary.json

# final stage
COPY --from=api-build /licenses /app/licenses/go
COPY --from=web-build /licenses /app/licenses/npm
```

Serve or document the path. Add the equivalent (`pip-licenses`) to the agent image.

---

## L3 — MEDIUM — Apache-2.0 NOTICE files not propagated

Two dependencies in the Go build graph ship a `NOTICE` file:

- `gopkg.in/yaml.v2@v2.4.0/NOTICE`
- `gopkg.in/yaml.v3@v3.0.1/NOTICE`

Apache-2.0 §4(d) requires that a `NOTICE` file's attribution content be carried into derivative works and distributions. Fold these into the aggregated notice bundle from L2.

---

## L4 — MEDIUM — No license gate in CI

`.github/workflows/security-scan.yml` is genuinely strong: Gitleaks, `govulncheck`, `npm audit --audit-level=high`, and `pip-audit` failing closed on any finding. All four cover **vulnerabilities**. None covers **licenses** — which is precisely why an AGPL dependency has been sitting in `package.json` unremarked.

**Remediation.** Add a `license-check` job that fails on any non-allowlisted SPDX identifier, and explicitly rejects the `SEE LICENSE IN …` / `UNKNOWN` cases that hide real terms:

```yaml
  license-check:
    name: License allowlist
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@... # pin as elsewhere
      - name: npm licenses
        working-directory: src/web
        run: |
          npm ci
          npx --yes license-checker-rseidelsohn --production --excludePrivatePackages \
            --onlyAllow "MIT;ISC;Apache-2.0;BSD-2-Clause;BSD-3-Clause;BlueOak-1.0.0;0BSD;MIT-0;CC0-1.0;Unlicense;(MPL-2.0 OR Apache-2.0)"
      - name: go licenses
        working-directory: src/api
        run: |
          go install github.com/google/go-licenses@latest
          "$(go env GOPATH)/bin/go-licenses" check ./... \
            --disallowed_types=forbidden,restricted,reciprocal,unknown
```

Also worth generating an SBOM per release — `cyclonedx-python-lib` is already in the agent's dev tree, so the tooling habit is half-established.

---

## L5 — LOW — Nominatim usage does not meet the OSM usage policy

`src/api/services/geocode_service.go` calls the **public** Nominatim instance at `https://nominatim.openstreetmap.org/search`. Two policy gaps:

1. **User-Agent lacks contact information.** The UA is `Aurearia/1.0 (self-hosted coin collection app)`. The OSM Nominatim Usage Policy requires an identifying UA *with a contact email or website* so operators can reach the maintainer before blocking. Add one, ideally sourced from an admin setting so self-hosters supply their own.
2. **No rate limiting.** The policy caps absolute maximum use at 1 request/second. `GeocodeService` sets an 8-second timeout but no throttle. Add a token-bucket or minimum inter-request interval.

Also note that geocoding results are ODbL-licensed OSM data. Displayed geocoded coordinates should carry OSM attribution.

**Credit where due:** the Leaflet map layers get this right — `MintMapLeaflet.vue:9` and `CreateMintModal.vue:198` both set the correct `&copy; OpenStreetMap contributors` attribution on the tile layer.

---

## L6 — LOW — Google Fonts CDN contradicts the self-hosted / offline promise

`src/web/src/assets/styles/main.css:1` loads Cinzel and Inter from `fonts.googleapis.com`.

The fonts themselves are SIL OFL 1.1 and free to use; because they are linked rather than redistributed, there is no OFL notice obligation. The problem is product-level, not legal:

- The README promises the app "runs entirely on your infrastructure" and offers "offline read access". A CDN font request breaks both claims.
- Every user's browser makes a request to Google on page load, from an app whose selling point is data sovereignty.
- Fonts will not render offline or on an air-gapped LAN.

**Remediation.** Self-host both families under `public/fonts/` with `@font-face` and `font-display: swap`, and include their OFL copies in the notice bundle. `woff2` is already in the Workbox `globPatterns`, so precaching works with no config change.

---

## L7 — INFO — Unverified asset provenance

`src/web/public/coin-logo.jpg` (1.2 MB) has no recorded source or license. It is used as the favicon, the apple-touch-icon, and a PWA `includeAssets` entry. If it is not original work, it needs attribution and a license check; if it is original, record that in the notice bundle. (It is also 1.2 MB for something displayed at 192×192 — see [Q8](#q8--medium--81-of-the-shipped-bundle-is-one-duplicated-wasm-blob).)

No embedded third-party source was found in the repository: a scan for `SPDX-License`, `Copyright (c)`, `Licensed under`, and `adapted from` across all Go/TS/Vue/Python/CSS files returned **zero** matches. There is no vendored or copy-pasted foreign code to attribute — the licensing exposure is entirely at the dependency boundary.

---

## Adjacent, non-OSS legal note

Outside the scope of open source licensing but worth a deliberate decision: the app scrapes and imports third-party auction data (NumisBids, CNG), queries Numista, and links NGC certification lookups. Those are governed by each site's Terms of Service and, potentially, database rights — not by any OSS license. `docs/threat-model.md` covers security risk; there is no equivalent record of data-source terms. A short `docs/data-sources.md` recording what each integration accesses and under what terms would close that gap.

---

# Part 2 — Code quality & maintainability

## What this codebase does well

These are not filler — they are above the norm and worth protecting:

- **Enforced architecture.** `src/api/architecture_test.go` fails the build if handlers touch GORM or raw SQL, or if any package outside `main` imports `database` directly. Boundary debt is explicitly allowlisted with a written reason per file. This is a genuinely good pattern.
- **TypeScript discipline.** **Zero** occurrences of `: any`, `as any`, or `<any>` across 40,840 lines. Two `@ts-expect-error`/`eslint-disable` total. `noUncheckedIndexedAccess` is on.
- **Consistent XSS handling.** Every one of the 11 `v-html` bindings routes through `DOMPurify.sanitize`. No exceptions found.
- **No TODO debt.** A repo-wide scan for `TODO|FIXME|HACK|XXX` found 5 hits, all of which are false positives (regex literals, prompt text). For a codebase this size that is remarkable.
- **Supply-chain hygiene.** All GitHub Actions pinned by commit SHA; Docker base images pinned by digest; Dependabot active; four security scanners failing closed.
- **Clean toolchain state.** `go vet` clean, Ruff clean, ESLint zero errors, all 806 tests across three suites passing.

---

## Q1 — HIGH — `main()` is a 749-line wiring god-function

`src/api/main.go` is 852 lines, of which `func main()` is **749**. Inside it: **177 constructor calls** and **300 route registrations** across 8 route groups, in one linear block.

Measured against the rest of the codebase, this is a clear outlier — the next-longest function in the entire Go API is 245 lines, and only 18 of 2,034 functions exceed 100 lines.

**Why it matters.** Every new feature touches this function, making it a permanent merge-conflict hotspot. It is also the least-tested code in the project at **1.1% coverage** for `package main`, because a 749-line `main()` cannot be unit tested.

**Remediation.** Split along the seams that already exist:

```
main.go              → config load, DB connect, Run()
wire_repositories.go → newRepositories(db) *Repositories
wire_services.go     → newServices(repos, cfg) *Services
wire_schedulers.go   → registerSchedulers(reg, svcs)
routes_public.go     → registerPublicRoutes(r, h)
routes_protected.go  → registerProtectedRoutes(api, h)
routes_admin.go      → registerAdminRoutes(api, h)
routes_tools.go      → registerToolRoutes(api, h)
```

Each becomes independently testable, and the route-registration functions become natural homes for route-table assertions. The `SchedulerRegistry` type at the top of `main.go` shows the pattern is already understood — it just needs applying to the other 170 constructions.

---

## Q2 — HIGH — The Go linter config is both unwired and unloadable

`src/api/.golangci.yml` exists and enables 10 linters: `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused`, `bodyclose`, `gocritic`, `gofmt`, `misspell`. `docs/CHANGELOG.md` lists it as a delivered feature.

**It has never run.** `grep -rn golangci .github/ Taskfile.yml .pre-commit-config.yaml` returns nothing. CI runs only `go build`, `go vet`, and `go test`.

And it cannot run as written. Executed against the current release:

```
$ golangci-lint run ./...
Error: can't load config: unsupported version of the configuration: ""
```

The file uses the golangci-lint **v1** schema (`linters-settings`, `issues.exclude-dirs`, and the `gosimple` linter, since merged into `staticcheck`). golangci-lint v2 requires `version: "2"` and a restructured layout.

A second blocker surfaced while testing: `go.mod` declares `go 1.26.5`, and current golangci-lint releases (v2.6.2, v2.12.2) are built with Go ≤ 1.25 and refuse to analyse a module targeting a newer Go. Note that `go 1.26.5` is a **patch-level** go directive, which is unusual and tightens toolchain requirements unnecessarily. Convention is `go 1.26` plus a separate `toolchain go1.26.5` line — worth changing regardless, as it eases every third-party analyser.

**Concrete evidence the linters would find real work:** `gofmt -l` currently reports **13 unformatted files** in `src/api`, including production files `services/content_guard.go`, `services/image_variants.go`, `services/notification_service.go`, `services/ollama_service.go`, `handlers/coins.go`, `handlers/catalog_registry.go`, `models/refresh_token.go`, and `capture/circle.go`. The `.pre-commit-config.yaml` `go fmt` hook would catch these locally, which tells us the hooks are not consistently installed — and nothing in CI backstops them.

**Remediation (in order):**
1. Change `go.mod` to `go 1.26` + `toolchain go1.26.5`.
2. Migrate `.golangci.yml` to the v2 schema (`golangci-lint migrate` handles it once network access to the schema is available; otherwise hand-migrate — it is a ~25-line file).
3. Add a `gofmt -l` check to CI immediately — it needs no new tooling and would have caught all 13 files:
   ```yaml
   - name: Format check
     run: test -z "$(gofmt -l . | grep -v '^docs/')"
   ```
4. Add the `golangci-lint` job once 1–2 land.

---

## Q3 — HIGH — The social/privacy surface has zero test coverage

Measured Go coverage: **40.5% overall.**

| Package | Coverage |
|---|---:|
| `capture` | 97.5% |
| `testutil` | 86.9% |
| `middleware` | 68.9% |
| `services` | 55.8% |
| `database` | 46.2% |
| `repository` | 27.1% |
| `handlers` | 24.8% |
| `main` | 1.1% |

`docs/testing.md` sets out a "confidence over coverage" philosophy and explicitly declines a repo-wide percentage gate. That is a defensible position and this finding is not an argument against it. The problem is that the codebase violates the doc's **own** stated rule — *"every new service method must have at least one unit test"* — in exactly the places where the risk is highest.

Files where **100% of functions are uncovered** (≥10 functions each):

| File | Uncovered / total |
|---|---:|
| `repository/social_repository.go` | 30 / 30 |
| `repository/price_alert_repository.go` | 20 / 20 |
| `handlers/social.go` | 19 / 19 |
| `handlers/agent.go` | 15 / 15 |
| `repository/security_repository.go` | 13 / 13 |
| `repository/availability_repository.go` | 13 / 13 |
| `services/coin_intake_service.go` | 12 / 12 |
| `repository/showcase_repository.go` | 12 / 12 |
| `repository/api_key_repository.go` | 12 / 12 |
| `repository/admin_repository.go` | 12 / 12 |
| `services/valuation_scheduler.go` | 10 / 10 |
| `handlers/sets.go` | 21 / 23 |

The pattern is not random. **The entire social feature — follows, follower galleries, comments, ratings, public profiles, showcases — is untested end to end**, across `handlers/social.go` (833 lines), `repository/social_repository.go`, and `repository/showcase_repository.go`. That is the widest authorization and privacy surface in the product, and the one where a regression leaks another person's collection rather than merely breaking a page. `repository/api_key_repository.go` and `repository/security_repository.go` are in the same category.

**Remediation.** No blanket coverage target. Instead, treat authorization-bearing paths as a named tier in `docs/testing.md` and require table-driven owner/non-owner/public/private tests for each. Start with `social_repository.go` and `showcase_repository.go`; the repository test infrastructure already exists (`repository/*_repository_test.go` covers 15 other repositories with a working DB-backed harness), so this is filling in a proven pattern, not building one.

---

## Q4 — HIGH — No `context.Context` propagation anywhere in the data or service layer

| Measurement | Count |
|---|---:|
| GORM `WithContext(` calls in `repository/` | **0** |
| Service functions accepting `ctx context.Context` | 48 of ~1,458 (~3%) |
| Handlers using `c.Request.Context()` | 14 |
| Outbound `http.NewRequestWithContext` | 19 |
| Outbound `http.NewRequest` (no context) | 8 |

**Not a single database query in the entire application is cancellable.** Neither are ~97% of service calls.

For a CRUD app this would be a minor inefficiency. For this app it is not: requests fan out into LLM analysis, agent-service proxying, auction-site scraping, carrier APIs, and Nominatim — operations measured in seconds to minutes. When a client disconnects or a gateway times out, every downstream call keeps running to completion, holding a DB connection and a goroutine. Under the scheduler load already present (10 background schedulers), that compounds.

**Remediation.** Thread `context.Context` from `c.Request.Context()` down through services to repositories, and use `r.db.WithContext(ctx)` in the repository layer. This is mechanical but touches many signatures — do it package by package, starting with the paths that call outbound HTTP (`agent_proxy.go`, `coin_lookup_service.go`, `numisbids_service.go`, `cng_auction_service.go`, `availability_service.go`, `shipment_carrier_client.go`), where the payoff is immediate. Convert the 8 remaining `http.NewRequest` call sites at the same time. The `architecture_test.go` harness is the natural place to add a guard against new context-free repository methods once the conversion is done.

---

## Q5 — MEDIUM — The error-response helper has ~15% adoption

`handlers/helpers.go` defines `respondError(c, status, clientMsg, err)`, which sends a consistent JSON body *and* logs server-side detail. It is a good helper.

- `respondError(...)` call sites: **122**
- Raw `c.JSON(status, gin.H{"error": ...})` in `handlers/`: **719**

So ~85% of error responses bypass the helper and log nothing. Two knock-on inconsistencies:

- `parseID` — in the same file, three lines below `respondError` — emits its own raw `gin.H{"error": "Invalid ID"}` instead of calling it.
- `respondError` logs via `log.Printf`, while the rest of the app uses the structured `services.Logger` (362 call sites). Errors therefore land in a different stream, at a different format, from everything else — including the admin log viewer that `services.Logger` backs.

**Remediation.** Point `respondError` at `services.Logger`, make `parseID` use it, then migrate the 719 call sites (largely mechanical, and a good candidate for `gofmt -r` or a small AST rewrite). Consider a `gocritic`/custom check forbidding `gin.H{"error"` outside `helpers.go` once migration completes.

---

## Q6 — MEDIUM — The frontend modularity policy has drifted, in both directions

`docs/frontend-modularity.md` is a thoughtful document: it names five oversized modules, records their line counts, and sets a deliberate policy of *not* pre-emptively splitting them. The policy is sound. The data behind it is now wrong:

| Module | Doc claims | Actual | Change |
|---|---:|---:|---|
| `AddCoinPage.vue` | 1,307 | 671 | ▼ 636 |
| `CoinLookupPage.vue` | 1,097 | 505 | ▼ 592 |
| `App.vue` | 819 | 645 | ▼ 174 |
| `AdminSchedulesSection.vue` | 1,134 | **1,330** | ▲ 196 |
| `api/client.ts` | 780 | **1,118** | ▲ 338 (+43%) |

Three modules were successfully refactored and the doc never recorded the win. Two grew *past* the size that got them listed — which is the exact outcome the policy was written to prevent. And the largest frontend file in the repo, **`types/index.ts` at 1,927 lines**, is not in the document at all.

### The concrete refactor hiding inside `AdminSchedulesSection.vue`

It is 786 lines of template and 541 of script, and the reason is textbook duplication: **seven near-identical "Run History" blocks**, one per scheduler (Availability, Auction Ending, Auction Price Alert, Auction Watch Bid Digest, Valuation, Collection Health, Coin of the Day) — roughly 100 template lines each — plus seven parallel `useRunHistoryPagination<T>(...)` call sites.

Notably, the **logic** was already extracted into `useRunHistoryPagination`. Only the markup was left behind. Extracting a `<RunHistoryTable :columns :rows :pagination>` component would remove ~600 lines from this file and make the eighth scheduler a five-line addition rather than a hundred-line copy-paste. This is the single highest-value frontend refactor available, and it fits the policy's own "extract during workflow changes" rule the next time a scheduler is touched.

**Remediation.** Refresh the table (a ~10-line script in CI could keep it honest), add `types/index.ts` to the list, and extract `<RunHistoryTable>`.

---

## Q7 — MEDIUM — `types/index.ts` and `api/client.ts` are undifferentiated monoliths

`src/web/src/types/index.ts` holds **226 exported types in a single 1,927-line file** — the whole domain, flat. `src/web/src/api/client.ts` is 1,118 lines and opens with a **single 3,800-character import statement** pulling ~170 type names across three lines.

That import line is the symptom worth acting on: it means every domain concept is coupled to every other through one module, no feature can be reasoned about in isolation, and the file is a guaranteed conflict point.

**Remediation.** Split both by domain, mirroring the component folders that already exist (`coin`, `sets`, `auction`, `social`, `admin`, `quick-capture`, `wishlist-alerts`, `shipments`):

```
types/
  index.ts        # re-exports only, keeps `@/types` working
  coin.ts  sets.ts  auction.ts  social.ts  admin.ts  ...
api/
  client.ts       # axios instance, interceptors, error formatting
  coins.ts  sets.ts  auctions.ts  social.ts  admin.ts  ...
```

Keeping `index.ts` as a barrel means zero call-site churn — this can be done incrementally without a big-bang change.

---

## Q8 — MEDIUM — 81% of the shipped bundle is one duplicated WASM blob

Measured from a real production build (`npx vite build`):

| Artifact | Size |
|---|---:|
| **`dist/assets/ort-wasm-simd-threaded.jsep-*.wasm`** | **22.8 MB** |
| `dist/coin-logo.jpg` | 1.2 MB |
| `dist/assets/ort.webgpu.bundle.min-*.mjs` + `.js` | 0.8 MB |
| `dist/assets/ort.bundle.min-*.mjs` + `.js` | 0.8 MB |
| Everything else | ~2.4 MB |
| **Total `dist/`** | **28 MB** |

Three separate problems here:

1. **The ONNX runtime is shipped twice.** Vite emits the 22.8 MB WASM into `dist/assets/` because `@imgly/background-removal` pulls in `onnxruntime-web`. Meanwhile `npm run prepare-background-removal-assets` downloads *the same runtime* into `public/imgly-background-removal/onnxruntime-web/`, and `backgroundRemoval.ts` sets `publicPath` to point there. At runtime the imgly loader builds its URL from `publicPath` — so the bundled copy is very likely never fetched. Worth confirming with a browser network trace before deleting, but the duplication itself is certain: both copies land in every Docker image.
2. **The WebGPU variant cannot be reachable.** `backgroundRemovalConfig` pins `device: 'cpu'`, yet `ort.webgpu.bundle.min` ships in both `.js` and `.mjs` form (~0.8 MB).
3. **`coin-logo.jpg` is 1.2 MB** and serves as favicon, apple-touch-icon, and a PWA `includeAssets` entry — displayed at 192×192 at most. Re-encoding to a ~30 kB WebP/PNG saves ~1.2 MB from the precache on every install.

**Remediation.** Confirm the runtime fetch path, then exclude the redundant ONNX assets from the Vite build (`build.rollupOptions.external` or an alias stub), drop the WebGPU bundle given `device: 'cpu'`, and downsize the logo. Expected result: **28 MB → ~3 MB**, which shrinks the image, the deploy, and the PWA install for every self-hoster.

Also: the build warns `caniuse-lite is 6 months old`. Add `npx update-browserslist-db@latest` to the Dependabot cadence.

---

## Q9 — LOW — Four `markdown-it` instances with three different configurations

`composables/useMarkdown.ts` exports `renderSafeMarkdown()` — the correct shared helper. Three other modules ignore it and construct their own renderer:

| Location | Config |
|---|---|
| `composables/useMarkdown.ts` | `{ html: false, linkify: true, breaks: true }` |
| `composables/useCoinSearchChat.ts:45` | `{ html: false, linkify: true, breaks: true }` |
| `components/coin/FeaturedCoinModal.vue:83` | `{ html: false }` |
| `components/coin/CoinAIAnalysis.vue:116` | `{ html: false }` |

Beyond four copies of the same bundle-level dependency, the configs diverge: the same AI-generated markdown renders with autolinking and line breaks in the chat surface but not in the analysis panel — a real, user-visible inconsistency.

More importantly this is a **security-relevant** duplication. All four currently call `DOMPurify.sanitize` correctly, but if `renderSafeMarkdown` is ever hardened (a stricter `ALLOWED_TAGS`, a link-rel policy), three call sites silently miss the fix. `useCoinSearchChat.ts:436` already passes custom `DOMPurify` options, showing the drift has started.

**Remediation.** Route all four through `renderSafeMarkdown`, extending it with an options parameter for the chat case. Consider an ESLint `no-restricted-imports` rule blocking direct `markdown-it` and `dompurify` imports outside `useMarkdown.ts`.

---

## Q10 — LOW — CI lets warnings and formatting drift accumulate

ESLint currently reports **169 warnings, 0 errors**. Because `npm run lint` has no `--max-warnings 0`, CI is green regardless.

| Rule | Count |
|---|---:|
| `vue/html-indent` | 100 |
| `vue/multiline-html-element-content-newline` | 52 |
| `@typescript-eslint/no-unused-vars` | 8 |
| `vue/one-component-per-file` | 5 |
| other | 4 |

152 of these are pure formatting — the real signal is that **no formatter is configured** (no Prettier, no `eslint --fix` gate), so whitespace drifts and buries the 8 warnings that are genuine dead code, e.g. `getDimensionFillClass` unused in `CollectionHealthScorecard.vue:82` and an unused `ref` import in `useCoinDetailContext.ts:4`.

**Remediation.** Run `eslint --fix` once to clear the 154 auto-fixable warnings, fix the ~15 real ones by hand, then set `--max-warnings 0` so the count cannot creep back. Pair with the `gofmt` gate from [Q2](#q2--high--the-go-linter-config-is-both-unwired-and-unloadable) and `ruff format --check` for the agent.

Minor related item: `src/web/src/utils/options.spec.ts` is the only `.spec.ts` among 102 test files (all others are `__tests__/*.test.ts`). Because it sits outside a `__tests__` directory, `tsconfig.app.json`'s `exclude` does not catch it and it is type-checked as application code. Move it to `utils/__tests__/options.test.ts`.

---

## Q11 — LOW — Build and deployment configuration inconsistencies

**`Taskfile.yml` — the default task is broken.** Indentation places `cmds` as a sibling of `default` rather than a child:

```yaml
tasks:
  default:      # ← parses as null
  cmds:         # ← parses as a task literally named "cmds"
    - task --list
```

Parsed keys confirm it: `['default', 'cmds', 'up', 'build-api', ...]`, with `default` resolving to `None`. Running bare `task` does nothing useful; `task cmds` is what actually lists tasks.

**Three different Docker image names across three files:**

| File | Value |
|---|---|
| `Taskfile.yml` | `DOCKER_REPO: bjd145` |
| `docker-compose.yaml` (app) | `${DOCKERHUB_USERNAME:-briandenicola}/ancient-coins` |
| `docker-compose.yaml` (agent) | `${DOCKERHUB_USERNAME:-ancient-coins}/ancient-coins-agent` |

The agent's fallback is plainly wrong — it defaults to an org named `ancient-coins`. A user running `docker compose up` without `DOCKERHUB_USERNAME` set gets a working app and a broken agent pull.

**Node version divergence.** CI (`ci.yml:59`, `security-scan.yml:71`) builds and tests on **Node 20.19.0**; the `Dockerfile` builds the production bundle on **node:24-alpine**. CI therefore never exercises the toolchain that produces the shipped artifact. Node 20 also reached end-of-life in April 2026 — CI is running an unsupported runtime. Align both on Node 24 and update `engines` accordingly.

**Other smaller items:**
- `docker-compose.yaml` defines a healthcheck for `agent` but none for `app`, despite `/health` and `/healthz` existing.
- `Taskfile.yml`'s `build-web` uses `npm install` (non-reproducible) rather than `npm ci`, and omits `prepare-background-removal-assets` — so a local `task build-web` produces a bundle whose background-removal feature cannot load its model, diverging from the Dockerfile.
- No `lint-web` / `test-web` tasks, though `lint-agent` and `test-agent` exist. The asymmetry pushes contributors toward raw npm commands.

---

## Q12 — LOW — Documentation has drifted from the code

407 markdown files / 63,331 lines is a serious documentation investment, and the deep-dive docs (`ARCHITECTURE.md`, `threat-model.md`, `testing.md`, the ADR set) are genuinely good. The drift is concentrated in the metadata:

| Drift | Detail |
|---|---|
| **Version disagreement** | `VERSION` says `3.7`; `README.md:115` announces "**v2.0 (Latest)**" |
| **Changelog never released** | Every entry sits under `## [Unreleased]` — no version sections at all, despite 3.7 and 599 merged PRs |
| **Changelog claims unshipped tooling** | Lists "golangci-lint config — errcheck, gocritic, misspell, bodyclose, staticcheck" as delivered; it has never executed (see [Q2](#q2--high--the-go-linter-config-is-both-unwired-and-unloadable)) |
| **Stale repository name** | README CI badge, clone instructions, and links in `frontend-modularity.md` and `threat-model.md` still point at `briandenicola/coin-collection-app`; the repo is now `briandenicola/Aurearia`. The CI badge is therefore broken on the README |
| **Stale module sizes** | See [Q6](#q6--medium--the-frontend-modularity-policy-has-drifted-in-both-directions) |

**Remediation.** Fix the repo-name references (a one-line `sed`, but it un-breaks the README badge); reconcile `VERSION` with the README; cut `[3.7]` in the changelog and adopt the release-tagging discipline the "Keep a Changelog" header already promises.

---

## Q13 — LOW — Agent-process artifacts are committed to the repository

**222 tracked files** under `.squad/` and `.specify/`, including runtime output that `.gitignore` was clearly meant to exclude:

- `.squad/orchestration-log/` — 53 markdown files
- `.squad/log/` — 31 markdown files
- `.squad-phase2-commit-msg.txt` — a stray commit message at the repository root

`.gitignore` already lists `.squad/orchestration-log/` and `.squad/log/`, but these files were committed before those rules were added, so the ignore has no effect on them.

Roughly 84 of the 407 markdown files are agent run logs rather than documentation, which dilutes search and makes the docs tree harder to navigate.

**Remediation.** `git rm -r --cached .squad/orchestration-log .squad/log && git rm --cached .squad-phase2-commit-msg.txt`. Keep the durable parts (`.squad/agents/`, `.squad/skills/`, `.specify/memory/constitution.md`) — those are real project artifacts.

---

## Q14 — LOW — LangGraph API deprecation will break on the next major

The agent test run surfaces:

```
LangGraphDeprecatedSinceV10: create_react_agent has been moved to `langchain.agents`.
Please update your import to `from langchain.agents import create_agent`.
Deprecated in LangGraph V1.0 to be removed in V2.0.
```

Two call sites: `app/llm/provider.py:71,78` and `app/teams/collection_chat.py:19,109`.

`pyproject.toml` pins `langgraph>=1.2.10,<2.0`, so nothing breaks today — but the upper bound is the only thing holding it, and Dependabot will eventually propose the 2.0 bump. Migrating now is a two-line change per file.

---

# Prioritized roadmap

### P0 — Before the next public image push
| # | Action | Effort |
|---|---|---|
| [L1](#l1--critical--agpl-30-dependency-shipped-in-an-mit-licensed-product) | Resolve the AGPL-3.0 conflict — replace `@imgly/background-removal`, relicense, or license commercially | Days |
| [L4](#l4--medium--no-license-gate-in-ci) | Add an SPDX license-allowlist gate to CI | Hours |

### P1 — Next sprint
| # | Action | Effort |
|---|---|---|
| [L2](#l2--high--no-third-party-attribution-notices-in-any-distributed-artifact) / [L3](#l3--medium--apache-20-notice-files-not-propagated) | Generate and ship third-party notices in both images | Hours |
| [Q2](#q2--high--the-go-linter-config-is-both-unwired-and-unloadable) | `go 1.26` + `toolchain`; add `gofmt -l` gate; migrate and wire `.golangci.yml` | 1 day |
| [Q3](#q3--high--the-socialprivacy-surface-has-zero-test-coverage) | Cover the social/showcase/api-key authorization paths | 2–3 days |
| [Q11](#q11--low--build-and-deployment-configuration-inconsistencies) | Fix Taskfile `default`, compose image defaults, Node 20 → 24 | Hours |

### P2 — This quarter
| # | Action | Effort |
|---|---|---|
| [Q1](#q1--high--main-is-a-749-line-wiring-god-function) | Split `main()` into wiring + route modules | 2–3 days |
| [Q4](#q4--high--no-contextcontext-propagation-anywhere-in-the-data-or-service-layer) | Thread `context.Context` through outbound-HTTP paths first, then repositories | 3–5 days |
| [Q8](#q8--medium--81-of-the-shipped-bundle-is-one-duplicated-wasm-blob) | Cut the duplicated ONNX runtime and WebGPU bundle; downsize the logo (28 MB → ~3 MB) | 1 day |
| [Q6](#q6--medium--the-frontend-modularity-policy-has-drifted-in-both-directions) | Extract `<RunHistoryTable>`; refresh the modularity table | 1–2 days |
| [L5](#l5--low--nominatim-usage-does-not-meet-the-osm-usage-policy) / [L6](#l6--low--google-fonts-cdn-contradicts-the-self-hosted--offline-promise) | Nominatim contact UA + rate limit; self-host fonts | 1 day |

### P3 — Opportunistic
[Q5](#q5--medium--the-error-response-helper-has-15-adoption) error-helper migration · [Q7](#q7--medium--typesindexts-and-apiclientts-are-undifferentiated-monoliths) type/client split · [Q9](#q9--low--four-markdown-it-instances-with-three-different-configurations) markdown consolidation · [Q10](#q10--low--ci-lets-warnings-and-formatting-drift-accumulate) `--max-warnings 0` · [Q12](#q12--low--documentation-has-drifted-from-the-code) doc/version reconciliation · [Q13](#q13--low--agent-process-artifacts-are-committed-to-the-repository) untrack agent logs · [Q14](#q14--low--langgraph-api-deprecation-will-break-on-the-next-major) LangGraph import migration

---

## Appendix — Reproducing these results

```bash
# License inventories
cd src/api && go mod download all          # then inspect $(go env GOMODCACHE)/**/LICENSE
cd src/web && python3 -c "import json;d=json.load(open('package-lock.json'))"   # license field per package

# Quality measurements
cd src/api && gofmt -l . | grep -v '^docs/'                     # → 13 files
cd src/api && go test ./... -coverprofile=cover.out             # → 40.5% total
cd src/api && go tool cover -func=cover.out                     # → per-function coverage
cd src/web && npx eslint . --ext .vue,.ts,.tsx                  # → 169 warnings, 0 errors
cd src/web && npx vite build && du -sh dist                     # → 28 MB
cd src/agent && uv run ruff check app/ tests/ && uv run pytest -q

# Confirm the linter config is unloadable
cd src/api && golangci-lint run ./...
# Error: can't load config: unsupported version of the configuration: ""
```
