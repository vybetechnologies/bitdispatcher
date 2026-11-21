# LAS record details

Administration → Persons & vehicles supports optional subject photos, additional person and vehicle flags, local warrant entries, a registered sex offender marker and an official HTTPS registry profile link. The shared call-taker person editor exposes the same person fields. Vehicle entries support informational LAS-owned and employee-owned markers with an optional owner name.

Photos are resized to a maximum 640-pixel edge and saved as JPEG inside the private person record. JPEG/PNG imports are validated for type, dimensions and size. Existing core safety flags and employee affiliations remain independent.

Warrants retain number, issuing agency, charge, status, issued/expiration dates and instructions. MDT shows all entries but voices only active entries that have not expired by local calendar date. These are locally entered records; officers verify them with the issuing agency. No NCIC connection is implied.

MDT search and assigned-call cards show the new fields. Voice announces local active warrants, the registry marker, additional cautions and vehicle affiliations. Windows downloads these fields with the existing protected offline record snapshot. Refresh that snapshot online before relying on new entries offline.

CSV templates include the added fields. `cautions` and `warrants` cells contain JSON arrays (CSV-escaped); person JSON imports place these fields inside `person`, while vehicle cautions and ownership are top-level. Existing record version checks, import preview/create-only behavior, permissions and audit attribution are retained.

Validation: frontend unit suite; CAD/API suites; import-to-MDT and offline-snapshot API integration; synthetic web/Windows UI checks across six viewport sizes including emergency-button visibility. No production person searches or test incident creation are used for verification.
