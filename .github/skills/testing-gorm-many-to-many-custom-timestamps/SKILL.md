---
name: testing-gorm-many-to-many-custom-timestamps
description: "Protect custom join-table timestamps and memberships during GORM entity updates."
---

# Join-table preservation regressions

Read the actual join model, association configuration and repository methods.
GORM association updates do not automatically satisfy arbitrary custom fields
such as a required membership `AddedAt`.

1. Use an isolated test database, migrate both entities and the explicit join
   model, and fail immediately on every setup/query/write error.
2. Create membership through the repository path that owns its custom fields.
   Assert the original timestamp is non-zero and retain it as a baseline.
3. Update an unrelated parent field. Assert the field changed while membership,
   custom timestamp and other custom attributes remain unchanged.
4. Send an association field through a path intended to ignore it. Assert the
   original membership remains and the attempted replacement was not inserted.
5. Exercise the dedicated association-edit path separately, including owner
   scoping and failure/transaction behavior.
6. Where the approved contract preserves associations, verify the actual
   `Omit("Tags", "Sets")` or equivalent mechanism is load-bearing with a temporary
   negative control. Do not confuse a missing migration with successful omission.

Existing examples are in `src/api/repository/coin_repository_test.go` and the
coin/set repositories; rediscover test names rather than trusting old line
numbers. Use the `go-sqlite-test-isolation` skill and authorized `task check:go`.
