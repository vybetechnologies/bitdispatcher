# LAS record details

Administration → Persons & vehicles supports optional subject photos, additional person and vehicle flags, local warrant entries, a registered sex offender marker and an official HTTPS registry profile link. The shared call-taker person editor exposes the same person fields. Vehicle entries support informational LAS-owned and employee-owned markers with an optional owner name.

Photos are resized to a maximum 640-pixel edge and saved as JPEG inside the private person record. JPEG/PNG imports are validated for type, dimensions and size. Existing core safety flags and employee affiliations remain independent.

Warrants retain number, issuing agency, charge, status, issued/expiration dates and instructions. MDT shows all entries but voices only active entries that have not expired by local calendar date. These are locally entered records; officers verify them with the issuing agency. No NCIC connection is implied.

MDT search and assigned-call cards show the new fields. Voice announces local active warrants, the registry marker, additional cautions and vehicle affiliations. Windows downloads these fields with the existing protected offline record snapshot. Refresh that snapshot online before relying on new entries offline.

CSV templates include the added fields. `cautions` and `warrants` cells contain JSON arrays (CSV-escaped); person JSON imports place these fields inside `person`, while vehicle cautions and ownership are top-level. Existing record version checks, import preview/create-only behavior, permissions and audit attribution are retained.

Validation: frontend unit suite; CAD/API suites; import-to-MDT and offline-snapshot API integration; synthetic web/Windows UI checks across six viewport sizes including emergency-button visibility. No production person searches or test incident creation are used for verification.

Vehicle records can optionally link to an existing local person using `registeredOwnerId`. Administration provides a local-only person picker and removal control. Links remain separate from affiliation, free-text owner names and vehicle cautions. MDT vehicle results include expandable current owner information; the Windows offline snapshot carries the same authorized information. Restricted, archived, under-18 and out-of-scope owners stay excluded under existing lookup rules. Stored links use record IDs, so identity corrections resolve automatically on the next online lookup/snapshot refresh.

Person records now support optional employer name, occupation/job title, workplace address, phone and notes through the shared person editor. Administrative person records also support typed links to other existing local people: spouse, partner, parent, child, sibling, relative, friend, associate, roommate, coworker, employer, employee, guardian, dependent and other. Relationship type describes the target relative to the current person; notes are optional. Save the person record to persist changes.

MDT search results provide “View person & associates”; registered owners can open the same profile browser. Each profile loads one direct neighborhood and can open the next person, with previous-person and breadcrumb navigation. Reverse parent/child, guardian/dependent and employer/employee relationships are derived automatically. Cycles return to the existing path rather than recursively expanding responses. Associate cautions do not become the current subject’s cautions; opening that associate loads their own safety information and voice warning. Existing own-unit, agency, archive/restriction and minor lookup rules apply on every open.

Profiles use the dedicated local-only audited `/api/mdt/records/person` endpoint; navigating relationships does not query EnformionGO or FBI. Windows snapshots include authorized direct associates and employer details for offline profile browsing. CSV `relationships` is a JSON array of `{personId,type,notes}` using existing local IDs; `employer` is a JSON object inside person JSON or a JSON-valued CSV cell. New references are validated before preview/commit.
