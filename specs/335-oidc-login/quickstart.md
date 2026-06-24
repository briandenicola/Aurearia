# Quickstart: OIDC Login for Entra ID and Pocket ID

## Prerequisites

- A local admin account exists and can sign in with username/password.
- `PublicAppURL` is configured to the externally reachable app URL for production.
- Each OIDC provider has registered frontend redirect URIs for login and account linking.

## Microsoft Entra ID setup

1. In Entra ID, create or select an app registration.
2. Add Web redirect URIs for the branded frontend callbacks, for example `https://your-app.example/auth/oidc/callback/1` and `https://your-app.example/settings/oidc/link/callback/1`.
3. Create a client secret and copy the **Value** immediately. Do not use the Secret ID.
4. In Aurearia Admin Settings, create an OIDC provider:
   - Provider type: `Microsoft Entra ID`
   - Display name: user-facing label such as `Microsoft`
   - Tenant ID: the Entra directory tenant ID; Aurearia derives `https://login.microsoftonline.com/{tenant-id}/v2.0`
   - Client ID: Entra app client ID
   - Client secret: Entra client secret
   - Scopes: `openid profile email`
5. Run the provider test and confirm discovery succeeds.
6. Enable the provider.

## Pocket ID setup

1. In Pocket ID, create an OIDC client/application.
2. Add redirect URIs for the branded frontend callbacks, for example `https://your-app.example/auth/oidc/callback/2` and `https://your-app.example/settings/oidc/link/callback/2`.
3. Copy the client ID and secret.
4. In Aurearia Admin Settings, create an OIDC provider:
   - Provider type: `Pocket ID`
   - Display name: user-facing label such as `Pocket ID`
   - Issuer URL: Pocket ID issuer base URL
   - Client ID: Pocket ID client ID
   - Client secret: Pocket ID client secret
   - Scopes: `openid profile email`
5. Run the provider test and confirm discovery succeeds.
6. Enable the provider.

## Validation workflow

1. Sign out and confirm local username/password login still works.
2. Confirm enabled OIDC providers appear on the login page.
3. Sign in with a linked OIDC identity and confirm the app opens normally.
4. Sign in locally, open Account Settings, link an OIDC identity, then sign out and sign back in with that provider.
5. Attempt to delete or demote the final local admin and confirm the app blocks the action.
6. Open Admin Security Events and confirm OIDC login/link/config events are recorded without tokens or secrets.

## Recovery guidance

Keep at least one admin account with a known local password. OIDC admin accounts are useful for convenience but do not satisfy recovery if the provider is unavailable or misconfigured.
