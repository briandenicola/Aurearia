---
name: "go-sqlite-test-isolation"
description: "Ensuring SQLite in-memory databases are truly isolated per Go test when using glebarez/sqlite"
domain: "testing"
confidence: "high"
source: "earned"
---

## Context

In Go test files using `gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})`, multiple test functions that run in the same `go test` process **share the same underlying in-memory SQLite database**. Data inserted in one test leaks into subsequent tests that use a bare `:memory:` DSN.

This causes subtle failures where:
- A test that should return 0 results finds stale rows from an earlier test
- Auto-increment primary keys advance across tests (so data from test N ends up with IDs that test N+2 accidentally queries)
- The failure is non-deterministic if tests run in a different order

## The Fix

Use a **uniquely-named** in-memory database per test by encoding a test-specific name in the DSN:

```go
var testCounter uint64  // package-level

func setupTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    dbName := fmt.Sprintf("file:my_svc_%d_%d?mode=memory&cache=shared",
        time.Now().UnixNano(), atomic.AddUint64(&testCounter, 1))
    db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
    // ...
}
```

The `file:NAME?mode=memory&cache=shared` DSN creates a **named** in-memory database. Each unique name is a separate SQLite database. The `cache=shared` flag is required to allow GORM's connection pool (multiple Go connections) to see the same in-memory DB — without it, each pooled connection would see a different (empty) database.

## Counter-Example (Bug Pattern)

```go
// BAD: all tests in the same process share one anonymous in-memory database
func setupTestDB(t *testing.T) *gorm.DB {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    // ...
}
```

## Why It Was Working Before (False Negative)

Tests that used `:memory:` but happened to not overlap on `(coinID, userID)` pairs or to not read back stale rows appeared to pass. When scoring weights changed such that more items were written to the shared DB (and those items matched the target coin in a later test), the failure surfaced. This is a data-dependent flake, not a stable test.

## Signal

If a test that creates a fresh user with `userID=1` returns more results than expected, and those extra results have data characteristics consistent with a different test's setup, suspect `:memory:` sharing.

## When `:memory:` Is Safe

`:memory:` without a unique name is safe only when:
1. Each test opens and closes its own separate `*sql.DB` (not pooled)
2. OR the test is the only test in the package that writes to the relevant tables

In practice, neither condition holds in multi-test packages with GORM. Use named in-memory DBs everywhere.
