# Image Operations

> Background removal, text extraction (OCR), and circle clipping

## Overview

Image operations are available from coin detail image galleries and upload flows. The app keeps original uploads, serves authenticated media through the API, and generates smaller variants for responsive loading.

## Key Features

- **Authenticated image serving** — Private uploaded media is served through authenticated API routes instead of public static paths.
- **Responsive image variants** — Uploads generate thumbnail and medium JPEG variants for faster card/detail rendering. Components choose thumbnail, medium, or full size based on context and network quality. A Quick Capture draft's photos get theirs when the draft is promoted to a coin.
- **Background removal** — Coin detail image lightboxes can run client-side background removal using `@imgly/background-removal`.
- **Paste image URL** — Coin image sections can fetch an external image through the server proxy and attach it to the coin.
- **Camera capture** — PWA/mobile flows can capture images directly after an explicit user action.
- **OCR/text extraction** — Image tooling supports extracting text from store cards and certificates where configured.
- **Circle clipping** — Image tools support coin-focused circular presentation/cropping workflows.

Original uploads remain available; variants are derived assets made from the stored original.

**Photos without variants.** Photos uploaded before variants existed (July 2026) have none. Clients that cannot fetch media URLs, such as the MCP coin viewer in Domus, are sent a photo's thumbnail or medium variant inline, and an original only when it is 512 KB or less, so a larger photo without variants did not appear there at all. The image variant scheduler makes the missing variants: two minutes after the API starts, then daily at `ImageVariantCheckStartTime` (03:17 by default) in the schedule time zone. `ImageVariantCheckEnabled` (on by default) turns it off. Both are on the Admin page under Schedules, in the Image Variant Check panel. It checks every coin photo (attribution cards are left alone), one at a time, makes what is missing, and logs how many it checked and made, and each photo it could not resize with the reason (a corrupt file, or an original over 40 MB). A photo that already has both variants costs two file checks. A pass stops between photos when the API shuts down, and the next one carries on.

## Related Features

- [Collection Management](collection-management.md)
- [Wish List](wish-list.md)
- [Coin Details](coin-details.md)

See also: [features.md](../features.md)
