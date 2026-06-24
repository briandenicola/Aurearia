# OIDC setup

Ancient Coins supports OpenID Connect sign-in with Microsoft Entra ID, Pocket ID, or another standards-compliant provider. Keep at least one admin account with a local password so you can recover the app if OIDC is unavailable or misconfigured.

## Redirect URIs

Register both redirect URIs for each provider, replacing the host and provider ID:

- Login: `https://your-app.example/api/auth/oidc/{providerId}/callback`
- Account linking: `https://your-app.example/api/auth/oidc/{providerId}/link/callback`

Local development may use `http://localhost` redirect URIs. Production deployments should use HTTPS.

## Microsoft Entra ID

1. In Entra admin center, create or choose an App registration.
2. Add the two web redirect URIs above under Authentication.
3. Create a client secret and copy it once; Ancient Coins stores it write-only and never returns it from read APIs.
4. Enter the Tenant ID in Admin Settings and confirm the derived issuer URL shown under the field is `https://login.microsoftonline.com/{tenant-id}/v2.0`.
5. Configure scopes: `openid`, `profile`, and `email`.
6. Save and test the provider from Admin Settings before enabling it.

## Pocket ID

1. Create an OAuth/OIDC client in Pocket ID.
2. Add the two redirect URIs above.
3. Copy the client ID and client secret.
4. Use the Pocket ID issuer URL that serves `/.well-known/openid-configuration`.
5. Configure scopes: `openid`, `profile`, and `email`.
6. Save and test the provider from Admin Settings before enabling it.

## User linking

Existing users should sign in locally, start linking from Account Settings, and complete the provider flow. The backend blocks an external identity that is already linked to another account and blocks verified-email conflicts with a different local user; it does not silently merge accounts.

Users can unlink identities unless that would leave no usable sign-in method. A usable method is a local password, a passkey/WebAuthn credential, or another linked OIDC identity.

## Error categories

- Provider disabled or not found: the selected provider cannot be used.
- Provider misconfigured: discovery, redirect, client, or secret configuration needs admin attention.
- Provider denied access: the user cancelled or denied consent at the provider.
- Validation failed: state, nonce, issuer, audience, signature, expiry, subject, or verified-email checks failed.
- Account conflict: sign in locally and explicitly link the identity from Account Settings.

## Recovery admin guidance

OIDC-only admins do not count as recovery accounts. Before enabling OIDC-only workflows, confirm at least one admin has a working local password and can sign in without the provider. Final-local-admin protections block unsafe delete, demote, local-auth disable, and OIDC-only conversion operations.
