# Connected person records (LAS10)

Administration now uses the same PersonForm as call intake, not a reduced form.
It includes separate name parts, aliases, roles, location/sublocation, multiple
phones with type/source/verification/extensions, physical description, DOB and
approximate age, SSN, licence and other identification, notes and the core safety
concern fields. The same server normalizer validates both entry paths.

As explicitly requested by the owner, LAS authenticated person views show full
entered SSN/licence values. No new reveal, masking, approval or purpose step is
added for viewing those fields. The existing account/duty, unit and agency
authorization remains. The non-LAS upstream masking behavior is unchanged.
Government inquiry returns remain separate; these are LAS-entered person records.
Public calls contain none of this information.

Person.recordId connects incident entries to a standalone record. Existing
incident people are projected from their recorded ledger entries using stable
incident/person IDs when LAS starts. This does not append synthetic calls or
rewrite historical ledger events. It does not merge different people by name.
Repeated startup produces the same links and does not duplicate records.

Call-takers can Find record / Use this person, or Load current linked details
when editing an already connected entry. Full reusable fields are copied and
recordId retained; this call's role, location, clothing and incident safety
context remain. Saving a person updates its connected record. Prior incident
snapshots stay as recorded; loading current linked details is explicit when
reusing or refreshing a historical incident entry. Administration and MDT show
the related incident references. MDT results prefer the connected canonical
record rather than duplicating every occurrence on an incident.

The core officer-safety system remains in use. SafetyConcern retains title,
reason, weapons, instructions, source, reliability and incident/permanent scope.
Permanent warnings go through proposePersonFlag and the existing approval/review
rules. Core flag IDs are retained with the person identity; changing name/DOB
does not disconnect an already approved warning. Retired/expired flags are not
shown as active. MDT and Administration use the core FlagFor/ActiveAt rules and
show the existing warning and responder instructions. A different incident's
local safety context does not replace the shared record's established context.

JSON imports accept the full nested person object from the JSON template. CSV
adds all scalar person columns, aliases/roles separated by |, and optional phones
and safetyNote cells containing CSV-escaped JSON. Source remains required;
existing preview, atomic validation, duplicate and edit-version behavior remains.
Legacy LAS9 flat person records remain readable/editable through the full form.

Tests cover rich fields without redaction; shared identities on two incidents;
single MDT results with connected incident references; existing core flags after
reuse and identity correction; core safety validation on import; expanded CSV;
and deterministic old-incident linking/restart without name-based merges.
