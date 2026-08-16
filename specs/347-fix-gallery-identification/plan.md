# Implementation Plan: Gallery Image Identification Regression

## Scope

Normalize only Coin Lookup gallery selections before upload. Reuse the existing API
error extractor and leave camera capture, API contracts, and backend limits unchanged.

## Design

1. Decode each selected image in the browser.
2. Draw it to a canvas at its original size or at a maximum 1920-pixel edge.
3. Encode the result as JPEG at 85 percent quality and preserve the source timestamp.
4. Add normalized files to the existing preview and lookup flow.
5. Reject an undecodable selection with an inline, user-readable error.

## Validation

- Unit-test image dimensions, output type, filename, and object URL cleanup.
- Component-test the gallery-to-lookup file contract and API validation message.
- Run the targeted Vitest files and the production frontend build.

