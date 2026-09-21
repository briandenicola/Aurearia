# Brutus: Current Working Context

Curated 2026-09-21 under Constitution section 18.3; not a new review or approval.
Read [charter](charter.md) when acting as Brutus,
[current work](../../identity/now.md), and [active decisions](../../decisions.md).
Keep artifact identity, review scope, restrictions and clearance explicit.

## Durable verification notes

- Prove the real user path and important negative cases. A passing guard is not
  meaningful until a controlled violation makes it fail and the original is
  restored. Source: original history lines 786-802.
- Reproduce suspected pre-existing failures against an isolated baseline rather
  than attributing them from memory or stashing someone else's changes.
- Do not perpetuate the old claim that all camelCase GORM update-map keys fail.
  The historical real-repository probe disproved that blanket assertion on the
  tested stack; validate the actual key and readback. Source: original line 788.
- Mounted App navigation coverage is possible; original history lines 794 onward
  document the public-asset/PWA/router mocks. Prefer behavioral evidence over
  source-string assertions when testing interaction.
- Local CRLF/tool output can mislead formatting checks; Linux-only runner
  conditions need matching runner evidence, not a local substitute.

## Review restrictions

The active review registry preserves R357-QA (Quick Access), R361 (specialist
contracts), R352/R353 (reviewer clearance not established), R225 and R320.
Brutus's approval is required for those original Brutus blocks; this summary
does not issue it. R361 excludes Cassius from revision and names Livia.
R357-QA requires a revision owner other than Aurelia, Livia or Brutus.
R-SWIPE's conditional platform/device requirements must not be self-cleared.

## Preserved evidence

[Full original history](history-archive.md#curation-2026-09-21-brutus), canonical
baseline `e6ab8313346b971dce4b288804036222f7d4c95d`.
[Inventory](../../artifacts/context-curation-2026-09-21.json): record `brutus`;
original line N is archive line N+11. The final entry, original lines 888-918,
records R361's exact rejected scope. Earlier blocks/clearances remain searchable,
including the historical compressed learning section at original line 565.
