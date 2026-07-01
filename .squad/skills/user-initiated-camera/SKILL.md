# User-Initiated Camera Start

Use this pattern for iOS/PWA camera surfaces that should not request permission on page load.

## Pattern

- Do not call `navigator.mediaDevices.getUserMedia()` from `onMounted()`, route entry, default mode watches, or retake/reset flows.
- Show the custom camera frame with a placeholder message and a visible `Start Camera` button.
- Wire `Start Camera` directly to `startCamera()` so permission is requested only from a user tap.
- Keep library upload controls available before camera start.
- Keep shutter/capture controls disabled or hidden until `cameraReady` is true.
- Stop active tracks on unmount and whenever the user leaves the camera mode.

## Reference files

- `src/web/src/pages/AddCoinPage.vue`
- `src/web/src/pages/CoinLookupPage.vue`
- `src/web/src/pages/__tests__/CoinLookupPage.test.ts`
