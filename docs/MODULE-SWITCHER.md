# Shared workspace module switcher

One shared shell shows a Modules button on every signed-in page, including MDT,
CAD screens, call intake, RMS/records, administration and ancillary screens.
The navy/gold button and popup reuse the existing panel/button styling. The
whole navigation strip is absent when the server returns zero or one module.
It occupies layout space rather than covering an existing module's controls.

GET /api/auth/modules derives options from the account's current duties and a
valid MDT unit assignment. POST /api/auth/module accepts only a module ID and
revalidates current grants, CSRF and the session lock. Module IDs map to existing
working duties; the browser cannot supply a new role or duty. An owner in Field
work can return to Administration using the same authenticated identity.

An elevated working duty under an MFA policy requires that this session already
verified MFA. Newly granted privileges cannot use an older password-only session
to bypass that check. Switching to MDT selects Field duty where assigned and
uses only the account's bound unit. Returning to desk work restores its console.
Passwords, MFA enrollment, account grants, unit bindings and training mode are
unchanged. Duty transitions are attributed in the ledger.

Same-duty navigation remains in the SPA. A duty change reloads the active page
and other tabs for that account. Old offline board snapshots are paused, cleared
and invalidated before a new duty is rendered. Pending offline work is preserved.
Permission changes are rechecked when selecting a module, independent of stale
menu content. The menu also refreshes on focus and every minute.

Tests cover one-module hiding, current module display, same-duty navigation,
refused stale grants, real field/admin transitions, no field-session admin rights,
restoring desk position, CSRF, locked sessions, MFA step-up and removed MDT
assignments. The implementation does not provide cross-application SSO.
