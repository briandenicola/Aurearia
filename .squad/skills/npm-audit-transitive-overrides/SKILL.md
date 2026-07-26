# Skill: npm audit transitive override remediation (frontend)

## When to use

Use this when `npm audit` fails in `src/web` on high-severity transitive vulnerabilities and `npm audit fix --force` proposes breaking/downgrade changes.

## Procedure

1. Run `npm audit` and identify the exact transitive chain.
2. Prefer targeted `overrides` in `package.json` to patch transitive packages without changing top-level dependency intent.
3. Run `npm install` to regenerate `package-lock.json`.
4. Re-run `npm audit`.
5. Run `npm run type-check` to verify frontend build parity.

## PR #531 example

- Override `@vue/test-utils -> js-beautify` to `2.0.3`
- Override `jake` to `12.10.1`
- Validate with `npm audit` (0 vulnerabilities) + `npm run type-check` (pass)
