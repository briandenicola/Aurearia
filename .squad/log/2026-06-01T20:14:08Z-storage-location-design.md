# Session Log — Storage Location Design

**Timestamp:** 2026-06-01T20:14:08Z  
**Scribe:** Session Logger / Decision Merger  
**Scope:** `.squad/` handoff only  

## Summary

Recorded Maximus's design-only investigation for a user-definable **Storage Location** dropdown. The proposed approach is a per-user `StorageLocation` lookup table, a nullable FK on `Coin`, settings-style management, and coin-edit dropdown selection.

## Decisions Processed

- Merged Maximus storage-location proposal into `.squad/decisions.md`.
- Merged pending Aurelia store-prefix label decision into `.squad/decisions.md` if not already present.
- Cleared `.squad/decisions/inbox/` after merge.

## Follow-up

Brian still needs to confirm final delete behavior for locations currently assigned to coins.
