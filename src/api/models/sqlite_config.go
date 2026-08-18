package models

// SQLiteConcurrencyDSNParams is the SQLite driver DSN query string that
// eliminates the "database is locked (5) (SQLITE_BUSY)" defect on
// concurrent job claims: `_txlock=immediate` makes every GORM
// db.Transaction() acquire SQLite's write lock (RESERVED) at BEGIN instead
// of deferring and later trying to upgrade a read lock, and
// `_pragma=busy_timeout(5000)` makes competing lock acquisitions wait up
// to 5s instead of failing immediately. Both are per-connection SQLite
// driver options that must be encoded in the DSN (not exec'd as a one-off
// PRAGMA) so every pooled connection picks them up as it is opened. Full
// root-cause narrative lives in database.Connect (src/api/database/database.go).
//
// This constant is the single source of truth for those parameters. It
// lives in models/ (stdlib-only, per architecture_test.go's import matrix)
// so it can be shared between database/ (production, in database.Connect)
// and repository/'s concurrency regression test
// (deep_identification_repository_test.go's newDeepIdentificationFileTestDB
// caller) without repository/ importing database/, which
// TestNoDirectDatabaseImports forbids. Do not change this value without
// re-verifying TestDeepIdentificationRepository_ConcurrentClaimNoLockContention
// (repository/deep_identification_repository_test.go) - it is the
// regression guard for this exact fix.
const SQLiteConcurrencyDSNParams = "_txlock=immediate&_pragma=busy_timeout(5000)"
