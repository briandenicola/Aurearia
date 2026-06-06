# PWA Features

> Progressive Web App with installable interface, offline read access, mobile-optimized controls, and camera integration.

## Overview

Ancient Coins is a fully-featured Progressive Web App installable on iOS, Android, and desktop browsers. It provides a native app-like experience with offline read access, touch-optimized controls, and camera integration.

## Installation

### Desktop (Chrome/Edge/Brave)
1. Visit the app in your browser
2. Click the **Install** button in address bar
3. App installs to your applications menu
4. Opens in standalone window (no browser UI)

### iOS (Safari)
1. Open app in Safari
2. Tap **Share**
3. Scroll and tap **Add to Home Screen**
4. Choose name and add
5. Launches in full-screen app mode

### Android (Chrome/Brave)
1. Open app in Chrome
2. Tap menu ⋮
3. Tap **Install app**
4. App adds to home screen
5. Launches in full-screen mode

## Features

### Offline Read-Only Access
- Service worker caches collection data
- Browse coins when offline
- View cached images
- No writes while offline; all changes require an active connection

### Mobile-Optimized UI
- **Swipe Gallery** — Touch-based card carousel for browsing
- **Pull-to-Refresh** — Refresh collection with downward swipe
- **Hamburger Menu** — Compact navigation in mobile header
- **Responsive Layout** — Adjusts for phones, tablets, desktops

### Camera Capture
- **Take Photos** — Use device rear camera to photograph coins
- **Available On** — Coin detail page, add coin form, image upload sections
- **Direct Integration** — Camera button appears in PWA mode
- **Auto-Save** — Photos attach directly to coin

### Touch Gestures
- **Swipe Left/Right** — Navigate between coins in gallery
- **Tap & Hold** — Context menu on cards
- **Double-Tap** — Zoom into images (in lightbox)
- **Pinch** — Zoom in/out on maps and charts

### Installation Badge
- Auto-badge appears when app is installable
- Persistent install option in app menu
- Standard browser or operating-system uninstall flow

## Performance

- **Fast Load** — Service worker caches resources for <1s startup
- **Minimal Data** — Optimized for cellular connections
- **Offline Reads** — Cached collection views work without network connectivity

## Background Removal

- **Client-Side ML** — Remove image backgrounds in-place
- **No Upload** — Privacy-preserving processing on your device
- **Available In** — Coin detail page image gallery

## Limitations

### Offline Write Access
- Offline writes are not supported
- All mutations require active connection
- Reads are fully cached

### Browser Support
- Desktop: Chrome 64+, Edge 79+, Brave, Firefox (limited)
- iOS: Safari 15.1+ (via Home Screen)
- Android: Chrome 40+

## Settings for PWA

Go to **Settings → Appearance**:

- **Default Gallery View** — Choose swipe (PWA) or grid (desktop)
- **Default Sort Order** — Affects both online and offline views
- **Time Zone** — Affects timestamps offline

## Related Features

- [Collection Management](collection-management.md) — Browsing and filtering
- [Camera Capture](camera-capture.md) — Photography integration
- [Image Operations](image-operations.md) — Background removal

## See Also

- [PWA Guide](../pwa-guide.md) — Detailed PWA documentation
- [Getting Started](../getting-started.md#pwa-installation)
