# Image Operations

> Background removal, text extraction (OCR), and circle clipping

## Overview

Image operations are available from coin detail image galleries and upload flows. The app keeps original uploads, serves authenticated media through the API, and generates smaller variants for responsive loading.

## Key Features

- **Authenticated image serving** — Private uploaded media is served through authenticated API routes instead of public static paths.
- **Responsive image variants** — Uploads generate thumbnail and medium JPEG variants for faster card/detail rendering. Components choose thumbnail, medium, or full size based on context and network quality.
- **Background removal** — Coin detail image lightboxes can run client-side background removal using `@imgly/background-removal`.
- **Paste image URL** — Coin image sections can fetch an external image through the server proxy and attach it to the coin.
- **Camera capture** — PWA/mobile flows can capture images directly after an explicit user action.
- **OCR/text extraction** — Image tooling supports extracting text from store cards and certificates where configured.
- **Circle clipping** — Image tools support coin-focused circular presentation/cropping workflows.

Original uploads remain available; variants are derived assets and can be regenerated from the stored original if needed.

## Related Features

- [Collection Management](collection-management.md)
- [Wish List](wish-list.md)
- [Coin Details](coin-details.md)

See also: [features.md](../features.md)
