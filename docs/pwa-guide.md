# Progressive Web App (PWA) Guide

Aurearia is a Progressive Web App — you can install it on your phone, tablet, or desktop and use it just like a native app. Once installed, it launches in its own window (no browser toolbar), loads instantly from cache, and gives you quick access to your collection right from your home screen.

The app is built with the [VitePWA](https://vite-pwa-org.netlify.app/) plugin and works on iOS, Android, and desktop browsers.

---

## Installing the App

### iOS (Safari)

1. Open the app URL in **Safari**
2. Tap the **Share** button (the box with an upward arrow)
3. Scroll down and tap **"Add to Home Screen"**
4. Give it a name and tap **Add**

The app icon will appear on your home screen. Tap it to launch Aurearia in full-screen standalone mode.

### Android (Chrome)

1. Open the app URL in **Chrome**
2. Tap the **three-dot menu** (⋮) in the top-right corner
3. Tap **"Install app"** or **"Add to Home Screen"**

Chrome may also show an install banner automatically the first time you visit.

### Desktop (Chrome / Edge)

1. Open the app URL in Chrome or Edge
2. Look for the **install icon** (⊕) in the right side of the address bar
3. Click **"Install"**

The app will open in its own window without browser chrome. You can find it in your system's app launcher just like any other application.

---

## How the PWA Differs from the Desktop Experience

When you run Aurearia as an installed PWA, the interface adapts to give you a more mobile-friendly experience. Here's what changes:

### Gallery View

| Feature | PWA / Mobile | Desktop Browser |
| ------- | ------------ | --------------- |
| **Filters & sorting** | Hamburger menu (☰) next to the search bar opens a popover with all options | Inline toolbar with filters, sort, and view controls |
| **"My Collection" title** | Hidden for a more compact header | Visible |
| **Default view** | Swipe carousel | Grid |
| **Add Coin button** | `(+) Add` link in the navigation bar | "Add Coin" button in the toolbar |
| **Page-level pagination** | Hidden in swipe mode | Visible in grid mode |

### Swipe Gallery

The swipe gallery is a touch-friendly card carousel for browsing your coins one at a time:

- Swipe left/right (or use the **Prev / Next** buttons below the card) to move between coins
- Cards are sized at **315 × 399 px** on mobile with `object-fit: contain` so the full coin is always visible
- Your position is saved — if you tap into a coin's detail page and come back, you'll return to the same card
- Switch between swipe and grid view anytime from the hamburger menu

### Pull-to-Refresh

When you're at the top of the gallery, pull down to refresh your collection:

1. **Pull down** from the top of the page — a pill-shaped indicator appears
2. Keep pulling past the threshold — the text changes from *"Pull to refresh"* to *"Release to refresh"*
3. **Release** — the indicator shows a spinner and *"Refreshing..."* while your collection reloads

Pull-to-refresh only activates when you're scrolled to the very top of the page, so it won't interfere with normal scrolling.

### Camera Capture

On mobile and PWA, a **Photo** button appears on image upload sections (coin detail page, add/edit coin forms). Tapping it opens your device's rear camera directly so you can photograph a coin without switching to a separate camera app first.

### Biometric Login (Face ID / Touch ID / Fingerprint)

Aurearia supports WebAuthn/FIDO2 passkey authentication, letting you sign in with your device's biometrics instead of typing a password.

#### Setting Up Biometric Login

1. Sign in with your username and password
2. Go to **Settings → Account**
3. Tap **"Register Biometric"**
4. Follow your device's prompt to register your face, fingerprint, or device PIN

Your passkey is stored securely on your device:
- **iOS** — iCloud Keychain / Passwords app
- **Android** — Device fingerprint or face unlock

#### Signing In with Biometrics

1. On the login screen, enter your username (the app remembers your last username automatically)
2. If a biometric credential exists for that username, a **"🔐 Sign in with Biometrics"** button appears
3. Tap it and authenticate with Face ID, Touch ID, or your device's fingerprint sensor

> **Note:** Biometric login requires HTTPS in production. It uses your device's built-in authenticator (not external security keys).

---

## Offline Behavior

The service worker caches all static assets (HTML, JavaScript, CSS, and images), so the app shell loads instantly even when you're offline or on a slow connection.

API calls (loading your collection, saving coins, AI analysis) still require network connectivity. If you're offline, the app interface will load but data operations will fail until you're back online.

---

## Settings & Preferences

You can customize the PWA experience in **Settings → Appearance**:

| Setting | Description |
| ------- | ----------- |
| **Default View** | Choose whether the gallery opens in **swipe** or **grid** mode |
| **Default Sort** | Set your preferred sort order for the collection |

These preferences are saved in your browser's local storage and persist across sessions.

---

## Detecting PWA Mode

If you're curious whether the app is currently running as an installed PWA, it checks:

```js
window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true
```

When this is `true`, the app enables all the PWA-specific UI adaptations described above.
