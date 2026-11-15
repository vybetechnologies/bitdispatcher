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

## Deployment verification

On 2026-11-15 at 05:57 UTC, release v0.1.0-las2 was deployed to the existing
Fly machine and data volume. The Linux image build passed the complete Go test
suite before producing the binary; the web build and all 98 frontend tests passed.
The custom hostname returned healthy status, the new version and logo, and 403
for government query access. The sign-in page was visually verified with LAS
branding. The existing authenticator enrollment was preserved. An administrator
sign-in and the supported LAS configuration application remain pending; deployment
alone does not replace an existing ledger's catalog. See las-live-login.jpg.

## LAS3 role and setup changes

Compatible existing patrol/site/security-supervisor units may be retained when applying
LAS defaults before incidents, reports, calls or response plans exist. Unit definitions,
status, crews and stations are preserved. Unsupported unit types and custom catalogs
remain blocked. Empty capability arrays are tolerated by Units, Call types, Settings,
Personnel and the incident unit panel.

LAS administrators may close required checklist steps with a reason. An override is
attributed in the ordinary setup event, displayed as a warning, and does not alter
underlying diagnostic details or the go-live test result. This records an owner decision;
it does not certify training, backups, connectivity or operational readiness.

Management functional-access.md and platform-access-model.ts identify two owners and
separate CAD/RMS/MDT grants. Native BitDispatcher role presets cover owner/command admin,
dispatch supervisor, dispatcher, operations supervisor plus MDT, field officer, records
and observer. They are local duties, not Management synchronization or an implementation
of its independent property/shift scopes. HR, finance and client permissions remain in
Management. No non-owner accounts receive new access automatically.

Administrator and field duties may coexist. Each field session gets only field rights;
an administrator session remains administrative. MDT has a unit selector for authorized
console roles. Device GPS is shared only after the user chooses Share device GPS.
Selecting another unit stops GPS sharing. Generated account passwords remain masked
in the browser for secure owner handoff and are not written to documentation or logs.

## LAS4 privacy and MDT restrictions

Supersedes the earlier administrator MDT selector: every LAS MDT request now resolves the
account's explicit assigned unit, including Administrator sessions. Another unit in a query
is denied. Unassigned accounts are denied; assignment removal is immediate, assignments
are unique and persistent, and the change is attributed in the ledger. Administrative
configuration rights remain intact. Planned live bindings are admin -> 12-A-1901 and
dylan.sebesta -> 12-A-1902, through Users and positions.

The separate /api/public/las-calls endpoint publishes eligible call categories automatically
after five minutes, with 15-minute approximate received times, broad metro area and a
hundred-block street only when a clean street address is recognizable. It never copies
persons, units, narratives, clients, exact locations, coordinates or attachments. Sensitive
and unknown types are excluded; an attributed per-incident confidentiality exclusion is
available. Active/Closed updates automatically; history is bounded to 90 days / 1,000 entries.
The old upstream public incident endpoint and pusher are unavailable in LAS.

The live LAS profile was applied at ledger events 310-356. The subsequent real go-live
check passed with six passes, six warnings and zero failures, including ledger restore
and 16 dry-run call types at Fargo. Setup reached 16/16 by recording the owners-only
account rationale as an administrator override; diagnostic warnings remain visible.
Mason and Dylan's original accounts retain MFA and Administrator, with MDT duty added.

## Final live verification

LAS4 is running on the existing Fly machine with one passing health check. The
Linux build passed the complete Go suite and the web build passed all 98 tests.
Current account bindings were verified in the live Users page: Mason's admin
account -> 12-A-1901, dylan.sebesta -> 12-A-1902. Both retain Administrator and
MDT duties and their existing MFA. A real request by Mason for Dylan's MDT unit
was denied; Mason's own terminal loaded with GPS off and Administration available.
The owner's separately added Mavis account/unit was preserved.

The public calls page is live at https://lakesareasecurity.com/calls. Its public
API returned 200 with no eligible calls during verification. Website release
668c5a94-9234-43a2-bdf2-ec73230208c2 is documented in the Lakes Area Security
repository's codex/public-calls worktree. This is an automatic limited feed,
not a Management account/data integration.
