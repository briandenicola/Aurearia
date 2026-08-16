# Feature Specification: Gallery Image Identification Regression

**Feature Branch**: `347-fix-gallery-identification`
**Created**: 2026-08-15
**Status**: In Progress
**Input**: User report: "I get a 400 error when I use images instead of the camera to start the identification process."

## User Story

As a mobile collector, I can choose an existing photo from my device and start
Quick Identify with the same reliability as a camera capture.

## Requirements

- **FR-001**: Gallery-selected images MUST be normalized to a provider-compatible
  JPEG before Quick Identify uploads them.
- **FR-002**: Normalization MUST preserve orientation, aspect ratio, and avoid
  upscaling while bounding the longest dimension to 1920 pixels.
- **FR-003**: Camera captures MUST retain their existing behavior.
- **FR-004**: If preparation or the API request fails, the UI MUST show a useful
  validation message rather than Axios's generic status text.
- **FR-005**: Existing backend body-size and image-validation limits MUST remain
  unchanged.

## Success Criteria

- **SC-001**: A gallery image using an iPhone/browser-decodable format reaches
  `POST /api/coins/lookup` as JPEG.
- **SC-002**: Gallery JPEG/PNG images larger than the analysis dimensions are
  reduced before upload.
- **SC-003**: Exact gallery-selection and error-message regressions are automated.

