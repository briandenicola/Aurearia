---
name: "legacy-set-type-normalization"
description: "Handle mixed legacy/new set-type contracts without writing deprecated values."
domain: "frontend-contract"
confidence: "high"
source: "earned"
---

## Context

Use when backend enum values are renamed but legacy values may still appear during migration windows.

## Pattern

- Define separate frontend types for **write values** (current contract) and **read values** (current + legacy aliases).
- Add one shared normalizer helper (for example `normalizeCoinSetType`) and branch UI logic on normalized values.
- Keep all create/update payloads restricted to new values only.
- Apply the normalizer everywhere behavior branches on type (filters, editability/membership controls, completion loading, labels).

## Anti-Patterns

- Do not compare raw `setType` strings in multiple components once aliases exist.
- Do not allow deprecated enum values in write payload unions.
