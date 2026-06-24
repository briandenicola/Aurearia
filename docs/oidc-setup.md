# OIDC setup

Aurearia supports OpenID Connect sign-in with Microsoft Entra ID, Pocket ID, or another standards-compliant provider. Keep at least one admin account with a local password so you can recover the app if OIDC is unavailable or misconfigured.

## Redirect URIs

Register both frontend redirect URIs for each provider, replacing the host and provider ID. In Microsoft Entra ID, add these under the **Web** platform, not SPA, because the Go API redeems the authorization code with a client secret.

- Login: `https://your-app.example/auth/oidc/callback/{providerId}`
- Account linking: `https://your-app.example/settings/oidc/link/callback/{providerId}`

Both redirects land on branded frontend result pages, which then complete the secure callback with the Go API. The API callback endpoints (`/api/auth/oidc/{providerId}/callback` and `/api/auth/oidc/{providerId}/link/callback`) remain internal exchange endpoints for the Vue app and should not be the primary redirect URIs registered with Entra or Pocket ID.

If you have an older beta provider registration that uses `/api/auth/oidc/{providerId}/callback` or `/api/auth/oidc/{providerId}/link/callback`, keep those only until every beta deployment and provider is updated to the frontend URIs above.

Local development may use `http://localhost` redirect URIs. Production deployments should use HTTPS.

## Microsoft Entra ID

1. In Entra admin center, create or choose an App registration.
2. Add the two frontend Web redirect URIs above under Authentication.
3. Create a client secret and copy the **Value** column immediately; do not use the Secret ID. Aurearia stores the secret value write-only and never returns it from read APIs.
4. Enter the Tenant ID in Admin Settings and confirm the derived issuer URL shown under the field is `https://login.microsoftonline.com/{tenant-id}/v2.0`.
5. Configure scopes: `openid`, `profile`, and `email`.
6. Save and test the provider from Admin Settings before enabling it.

## Pocket ID

1. Create an OAuth/OIDC client in Pocket ID.
2. Add the two frontend redirect URIs above.
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
