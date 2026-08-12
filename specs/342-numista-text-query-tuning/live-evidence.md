# Feature 342 Sanitized Live Query Evidence

**Observation date:** 2026-08-12  
**Provider:** Numista v3 text search (`category=coin`, `lang=en`, bounded to 10 results)  
**Credential handling:** `NUMISTA_KEY` was read from the process environment and was not printed, written, or retained.

This evidence contains query strings, candidate IDs, titles, and ranks only.
It contains no images, owner data, credentials, raw slab text, full user prose,
or raw provider payloads. Automated tests must replay committed fixtures and
must not call the live API.

| Case | Expected candidate | Verbose rank | V2 primary rank | Relaxed rank | Observation |
|---|---:|---:|---:|---:|---|
| Honorius, GLORIA ROMANORVM, SMNT | 208360 | — | 3 | 3 | Exact alias expansion produced `Nicomedia`; verbose returned no candidates. |
| Valentinian II, VOT V MVLT X, SMN | 332493 | — | — | — | Diagnostic query established the expected ID at rank 2, but the mint-constrained primary and relaxed query did not return it in the first 10. |
| Trajan, OPTIMO PRINC, Rome | 253021 | — | — | — | Diagnostic query established the Victory type at rank 1; both generated plans missed the expected ID in the first 10. |
| Athens owl tetradrachm | 373031 | — | — | 3 | The primary missed, while the relaxed title query returned the expected tetradrachm at rank 3. |
| Justinian I, M ANNO XII, Constantinople | 85834 | 1 | 1 | — | Both verbose and V2 primary returned the expected type at rank 1. |
| Aurelian, ORIENS AVG, unknown `XXIT` mintmark | 290869 | — | 2 | — | Unknown mintmark was omitted; V2 primary found the expected antoninianus. |

`—` means the expected candidate was absent from the bounded first ten results.

## Measured result

- Verbose-builder top-three inclusion: **1/6 (16.7%)**
- V2 primary top-three inclusion: **3/6 (50%)**
- Change: **+33.3 percentage points**
- Previously top-three verbose candidates lost: **0**
- Current comparison requests: **18** total (verbose, V2 primary, and relaxed
  query for each of six cases)

The six-case live sample exceeds SC-002's 10-point improvement threshold, but
it also shows that concise generation is not universally sufficient: the
Valentinian II and Trajan examples still missed the expected candidate in the
first ten, and Athens required relaxation. Results are a time-bounded provider
observation and may change as Numista's catalog/search ordering changes. The
deterministic fixture and 24-known-coin scorer gates remain the release-safe
automated evidence; CI does not call Numista.
