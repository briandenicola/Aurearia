---
name: go-sqlite-test-isolation
description: "Diagnose SQLite connection-pool isolation and shared-cache leakage in Go tests."
---

# SQLite test isolation

SQLite's anonymous `:memory:` database is private to a connection, not globally
shared by every Go test. A connection pool can therefore see different empty
databases. Named `file:NAME?mode=memory&cache=shared` connections share only when
they use the same name. See [SQLite's contract](https://www.sqlite.org/inmemorydb.html).

1. Inspect the actual DSN, GORM/database handles, pool settings and cleanup.
   Distinguish missing tables on a new connection from reused named/shared state.
2. Reuse a verified test helper. If a pool must share one in-memory database, use
   a unique test-specific name with `mode=memory&cache=shared`; prevent name
   collisions across parallel tests.
3. Alternatively, constrain a deliberately connection-local database to one
   connection where that matches the test's concurrency requirements.
4. Check open/migration errors, include required join models, and close the
   underlying `sql.DB` through test cleanup. Do not ignore failed setup.
5. Prove isolation using conflicting rows/identifiers in separate tests and
   verify pooled reads see the intended database. Exercise parallel/repeat
   behavior when relevant, not only the happy path.

Use authorized `task check:go` and relevant race evidence. Do not rewrite
application database architecture to fix an isolated test helper.
