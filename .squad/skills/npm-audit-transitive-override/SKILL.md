# npm audit transitive override

Use this when `npm audit` fails on a deep transitive package and `npm audit fix` (or `--force`) either fails to clear advisories or introduces risky major downgrades.

## Steps

1. Run `npm.cmd audit` in the package directory and capture the full dependency chain.
2. Try `npm.cmd audit fix --package-lock-only` first.
3. If unresolved and force-fix proposes breaking changes, add the narrowest `overrides` entry in `package.json` for the vulnerable transitive package (pin to patched version).
4. Refresh lockfile with `npm.cmd install --package-lock-only`.
5. Re-run `npm.cmd audit` and the smallest frontend safety check (`npm.cmd run type-check`).

## Example (PR #530)

- Added `"overrides": { "brace-expansion": "5.0.8" }` in `src/web/package.json`.
- Ran `npm.cmd install --package-lock-only`.
- Result: `npm.cmd audit` passed with 0 vulnerabilities; type-check still passed.
