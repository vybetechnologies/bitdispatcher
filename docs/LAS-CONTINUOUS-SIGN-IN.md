# LAS continuous sign-in (LAS13)

LAS CAD, MDT, call intake, records, viewer and administration share persistent sessions. LAS sessions have no idle or total-age expiry. Inactivity does not automatically lock the LAS UI or backend. Manual lock, manual sign-out, account disabling/removal and current duty/unit checks remain available.

Sessions are stored atomically as sessions.json in the protected persistent data directory, mode 0600. Tokens are never included in public session JSON, logs or the support bundle. Authentication and MFA proof, duty, console and assigned-unit context survive server restarts. Browser cookies remain Secure, HttpOnly, SameSite Strict, and are renewed during authenticated requests using the browser-supported 400-day maximum. Clearing browser storage, browser-enforced expiry after prolonged absence, or explicit revocation requires signing in again.

The first rollout cannot recover sessions held only in the old process's memory. Users may need to sign in once after this release; subsequent sessions survive deployments.

Regression coverage checks every role over a 20-year simulated interval, persistent restart recovery, MFA proof, manual logout and disabled-account revocation. Upstream non-LAS timeout behavior is unchanged.
