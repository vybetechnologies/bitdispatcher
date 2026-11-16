# Administrative records and MDT defaults (LAS9)

Administration → Persons & vehicles (`#/admin/las-records`) creates and edits
standalone LAS person and vehicle records, independent of CAD incidents. Create
records individually or download the CSV/JSON templates and import files.
Imports require validation and a preview before the Import action. Limit: 500
records and 750 KB per file. JSON is an array of objects; CSV uses exact template
headers with aliases separated by `|`. Unknown fields and malformed values are
rejected. Source is required. VINs, DOBs, plate states and lengths are validated.
One invalid row prevents the whole batch from being saved.

Preview writes nothing. Commit rechecks the current data and writes one attributed
ledger event. Imports create only; they never overwrite. Duplicates are skipped
within the file and against existing records: agency/kind/external source ID,
otherwise VIN or plate/state, otherwise person name/DOB when recorded, otherwise
the exact supplied person record. A shared name alone is not an identity match.
The preview displays the first ten new entries and counts all new/duplicate rows.

Editing requires the current record version, so another administrator's changes
cannot be silently overwritten. Archive/restrict switches immediately exclude
records from MDT discovery. Restore by clearing those switches. There is no hard
delete: the ledger preserves changes, actors and provenance across restarts.
Manage/search the admin list by name, aliases, plate, VIN or source ID, with
100-row pages. All management routes and commands require Administrator.

The existing assigned-unit MDT search includes these records alongside eligible
incident records. Results say LAS directory and show their stable record ID;
they do not invent an incident. The same name/alias/DOB and plate/state/VIN
matching applies. Existing unit binding, search-purpose, hash audit, 50-result
limit and agency controls remain. Archived/restricted/known-underage persons are
excluded even for an owner's MDT session. Only the existing result DTO is sent;
import source IDs, audit identities and administrative fields are not officer
search results. Directory entries do not enter the public calls feed.

GET/POST `/api/admin/las-records`, PUT `/api/admin/las-records/{id}` and POST
`/api/admin/las-records/import` use the normal authenticated/CSRF API wrapper.

GPS now starts automatically on the MDT and requires normal device/browser
location permission. Stop GPS ends its watch; leaving MDT also clears it.
Coordinates retain the existing assigned-unit guard and server logging limits.
Voice starts automatically using local device voices and retains its mute,
repeat and settings controls. Browser-blocked audio has an explicit Enable audio
prompt, rather than silently pretending speech works.

The two-second emergency control now occupies its own non-shrinking footer,
outside the scrollable call/message/status column. The header has a bounded,
scrollable height, including expanded voice settings. Emergency access remains
visible in a short full-screen window; visual QA confirmed a 48px control fully
inside a 495px viewport, even with voice settings expanded. Pointer cancellation,
window blur, unmount and keyboard release cancel an incomplete hold. Repeated
key-down events cannot create multiple emergency submissions.

Validation covers management permissions/CSRF, preview-only behavior, atomic
import validation, duplicate/idempotent imports, stale edit refusal, real field
lookup, restricted/minor/agency exclusions, ledger replay, admin creation/import
controls, automatic local speech and emergency hold/cancellation. UI proof uses
only labeled local fixtures, never production personal records or emergency calls.
