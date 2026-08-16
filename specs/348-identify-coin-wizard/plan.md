# Implementation Plan: Identify Coin Capture Wizard

## Scope

Replace only the Identify Coin capture state with a workflow-owned wizard.
Extend the existing lookup multipart request with optional bounded notes.

## Design

1. Extract capture UI into `CoinLookupCaptureWizard.vue`.
2. Store images by semantic role rather than array position.
3. Keep camera permission behind `Start Camera` and normalize gallery files
   through the existing Feature 347 helper.
4. Add optional `notes` and validated semantic image roles to the Vue-to-Go
   lookup contract.
5. Pass notes through `CoinDataProxy.Notes`; render them in the agent prompt as
   untrusted collector evidence, never as instructions.
6. Save collector notes and role-specific images to the Quick Capture draft.

## Validation

- Component tests for step progression, immediate analysis, optional steps,
  notes input, and explicit image roles.
- API client and Go handler/service tests for notes propagation and bounds.
- Agent test for safe notes context.
- Frontend build/tests and targeted Go/Python gates.
