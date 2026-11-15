# MDT person and vehicle lookup

The MDT header has a Persons & vehicles action. The panel uses the existing Modal,
navy surfaces, gold selected states, typography and large touch controls. It stays
inside MDT; the session duty and assigned unit do not change. Unit status keyboard
shortcuts are suspended while the panel is open. Search state clears on close.

POST /api/mdt/records/search is available only to authenticated MDT users with a
configured assigned unit and the normal CSRF header. Another unit in the request
is denied by the common MDT guard, including for Administrator sessions. Results
are limited to the assigned unit's agency plus unassigned single-company records.
No NCIC, DMV, national/state police or external provider request is made.

Persons: all supplied name words may match the name, separated name parts or an
alias; optional YYYY-MM-DD date of birth is an exact additional filter. Vehicles:
plate and VIN normalize spaces/punctuation/case; provided plate, state and VIN
filters combine. Empty/overbroad inputs are rejected. The newest 50 matching
records are returned, with a refine-search notice if there are more.

The DTO includes identification basics, physical/vehicle description, source
incident reference/date and an existing recorded safety-concern indicator. It
does not return SSNs, driver-license/other ID numbers, addresses, phones, owner
contacts, notes, narratives, attachments or government query returns. Juvenile,
medical, patient, under-18 and removed person records are excluded. Vehicles on
juvenile or medical incidents and removed vehicles are excluded. Searches do not
create, alter, link or publish any person, vehicle or incident record.

Every successful query records the user/duty/position, assigned unit, operational
purpose, searched field names, SHA-256 query fingerprint and bounded result count.
Raw lookup terms are not included in this event. Responses are no-store. Results
are explicitly possible local matches, never an identity confirmation or a clear
warrant/history response.

Tests cover assigned-unit/role/CSRF denials, person and vehicle combined criteria,
identifier/notes redaction, restricted records, input rejection, hash-only audit,
request routing, distinct forms and failed-versus-empty results. No test records
are created in production.
