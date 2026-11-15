# Lakes Area Security customization

This fork serves Lakes Area Security Corporation's North Dakota private-security operations.
The operating brand is Lakes Area Security. The company logo and Inter font are reused from
the company's existing website project. Navy is `#0a1a2f`, gold is `#c9a227`.
BioDream's Apache 2.0 license and attribution remain intact.

## Runtime and configuration

The service and `init-admin` seed LAS settings only when the ledger is empty.
An existing ledger is replayed unchanged. Deploying the binary does not replace persisted
call types, map settings, users, authenticator enrollment, or security policy.

An administrator can review **Administration → Lakes Area Security** and apply the LAS
starting configuration through `POST /api/admin/las`, with the current preview sequence,
explicit confirmation, session authorization and the CSRF header. Take a data-volume
snapshot first. Bulk application refuses incidents, reports, calls, units, response plans,
and customized call types. Use individual supported configuration screens on an operational
system. The action changes state through ordinary engine commands/events, never ledger
editing. The event group is not transactional: a storage failure may require retrying the
application. Both untouched upstream and LAS call definitions are recognized on retries.

The LAS catalog contains 19 starting call types and three security unit types. Run cards
must be reviewed against actual staffing and policy. No employees, stations or available
units are fabricated. Existing numbering, map, GIS, cameras, security policy and users
remain unchanged by profile application. The configured company phone is not moved or
published by the profile. The planned Fargo Office and Patrol 1 remain separate owner
configuration steps.

The internal `police` discipline and capability keys remain compatible with upstream
dispatch logic and historical events. The interface calls this **Security**; it does not
confer police authority. LAS units have an additional `security` capability used by the
new run cards. Fire/EMS choices are removed from LAS configuration controls. Government
interfaces (NCIC/VCIN, IBR, EIDO, radio pushes, NERIS/NEMSIS, IPAWS/public-alert issue and
PSC PIP) are blocked by the LAS runtime, including direct requests and the loopback hooks.
Government-only UI controls are not offered. Historical upstream source and documentation
remain for license compliance, compatibility and reference, not activation.

## Related company systems

Management remains authoritative for personnel, credentials, clients/sites, contracts,
billing and controlled records requests. NORTHSTAR remains employee self-service.
The Management link is navigation, not account synchronization or data integration.
BitDispatcher users remain local. The Neon records projection, central identity connection,
NDCOURTS and CLEAR are not implemented or claimed by this release. BitVoice is a separate
company-phone connection; no DID, SIP trunk, number port or paging provider is provisioned.

## Authenticator fix

The upstream multi-step sign-in consumed a valid TOTP code before asking a multi-duty
user to choose their duty. The browser's next submission reused that code and was refused.
Enrollment and last-used-step changes now commit only after a valid duty is selected and
the sign-in completes. Completed codes still cannot be replayed, including after restart.
Neither enrollment nor policy is reset. Regression tests cover enrollment and existing
authenticators, the intermediate prompt, completed replay and restart persistence.

## Build and release

Run `pnpm install --frozen-lockfile` and `pnpm build` in `web/`, then commit the generated
`internal/web/dist` alongside the source. Go embeds the LAS logo, font license, font and
assets. The Docker runtime version is `v0.1.0-las2`; the module selects Go 1.27.3.
`web/node_modules`, local environment files and test output are excluded from Docker context.
Shell files use LF line endings for Linux deployment from Windows.

Verify frontend tests, Go tests, profile safety/authorization tests and MFA flow tests before
deploying. Preserve the mounted `/var/lib/bitdispatcher` volume. A binary rollback cannot
undo configuration events: use supported application changes, or an owner-approved full
restore with explicit acknowledgement of data written after the backup.

The pre-release Fly snapshot requested during this work is
`vs_Qe0k689GmOmAfm8Om0AqY` on volume `vol_rnzqolegm7ewkz8r`; it was verified created.
Fly snapshot retention shown at creation is five days. It is not a permanent off-site backup.
