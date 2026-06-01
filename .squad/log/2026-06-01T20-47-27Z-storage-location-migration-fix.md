# Session Log — Storage Location Migration Fix

**Timestamp:** 2026-06-01T20-47-27Z  
**Requested by:** Brian  
**Primary agent:** Cassius  
**Commit:** d2179f1

Cassius fixed the Storage Location startup migration failure by disabling physical SQLite FK constraint migration for the new nullable Coin.StorageLocation association while preserving the scalar nullable column and preload association. Cassius also verified no data loss on a disposable copy of the real database: row counts were unchanged, storage_locations was created, and coins.storage_location_id was added nullable.

Scribe actions:
- Wrote orchestration and session logs.
- Merged the decision inbox entry into .squad/decisions.md.
- Added cross-agent history notes for Maximus and Aurelia about the SQLite nullable-FK convention.
