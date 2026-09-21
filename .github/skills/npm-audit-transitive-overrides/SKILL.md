---
name: npm-audit-transitive-overrides
description: "Resolve npm transitive advisories with narrow reviewed overrides and full frontend validation."
---

# Narrow npm remediation

1. Capture the advisory and dependency chain in the affected package.
2. Compare a compatible upstream fix with a narrow `overrides` entry. Do not run
   `npm audit fix --force` blindly or accept a breaking downgrade to clear a scan.
3. With authorization, change the manifest and regenerate its lock intentionally.
   `npm audit fix --package-lock-only` is still a mutating operation; inspect the
   resulting diff. Install/restore only under the approved setup boundary.
4. Pin an override to the reviewed patched version, scoped to the affected parent
   where possible. Historical versions below are not current recommendations.
5. Re-run the audit and authorized `task check:web`: zero-warning lint, strict
   type-check, full package tests and build. Type-check alone is not completion.
6. Record residual advisories/compatibility exceptions explicitly, not as success.

Use `npm.cmd` on Windows where PowerShell execution policy blocks npm.ps1.
Historical #530/#531 examples used `brace-expansion`, `js-beautify` under
`@vue/test-utils`, and `jake` overrides. Re-establish the current dependency chain
and patched releases rather than replaying those version numbers.
