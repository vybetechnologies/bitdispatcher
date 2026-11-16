# MDT voice and NWS map warnings

LAS11 adds priority officer-safety speech when a completed MDT person search
returns a core caution flag or recorded safety concern. It reads the possible
match's name, warning and responder instructions, once for each search run.
An identical new search still speaks its caution; ordinary screen refreshes do
not. A no-caution result clears the earlier search notice. The visual core flags
and full search details remain available. Vehicle/status/weather/routine voice
uses the same queue; person cautions take priority over routine announcements.
New pending calls for service announce type, priority, location and call number.
Fresh completed person-search cautions still speak if the separate call-update
stream is reconnecting; stale call/status announcements remain paused.
Existing pending calls are seeded silently on terminal entry; subsequent new or
changed calls are deduplicated, and withdrawn calls are removed from the queue.

The MDT header has Voice on/off, Repeat, Stop speaking and Voice settings.
Voice starts automatically on each signed-in MDT page. Speech uses only installed local
English Web Speech voices (never a remote speech service), with adjustable speed
and volume. No microphone, automatic status changes or message acknowledgement.
Mute applies to the current MDT page. A new MDT page starts voice again. Browsers
that block automatic audio show Voice on / Enable audio until the officer taps
that control; the application does not bypass browser audio restrictions.
The browser's audio output must be connected and the terminal must remain running;
sleep, browser suspension and device audio restrictions can interrupt speech.

It announces the assigned unit's current status and call assignment, call status
and priority changes, changed locations, site hazards, flags, cautions, active
papers, priority notes, emergencies, dispatch messages, BOLOs and urgent/high
communications. Only existing authorized MDT/inbox data is consumed.
Routine inbox messages and incoming messages from
other units are spoken too; the officer's own outgoing messages are excluded.
Successful MDT status changes speak their confirmed status when the terminal
refresh returns it. Fresh active NWS alerts from the map's same feed are spoken
with event, affected area, headline and instructions. Updated weather instructions
are announced; expired or stale weather cannot start a new announcement. Feed
recovery does not replay unchanged alerts. Urgent speech interrupts routine speech
while preserving still-current queued status confirmations and messages.
Identical poll results do not repeat. Updated/withdrawn items replace queued speech; safety
warnings interrupt routine speech. The queue is bounded to 12 announcements and
each announcement to 600 characters, rather than accumulating old updates.
Speech stops on lost live terminal data, lock, sign-out, mute or leaving the MDT.
Web Locks prevents two MDT tabs in the same browser from speaking concurrently
where supported. Visual alerts and acknowledgement controls remain authoritative.

Both the CAD and MDT maps show an NWS badge, affected-area shading and a panel
with event, severity, area, onset/end, headline, description and instructions.
Warning = red, watch = gold, advisory/other = blue. Click a polygon or an alert
to read details; selecting an alert fits its boundary. The layer can be hidden.

The authenticated GET /api/weather/alerts route reads api.weather.gov only:
/alerts/active?zone=NDC017,MNC027,NDZ039,MNZ003. These are Cass County ND and
Clay County MN county/forecast zones. This intentionally covers the LAS Fargo–
Moorhead operating area, not all North Dakota/Minnesota. No CAD/person/unit
data or officer GPS is sent to NWS. The server sets a descriptive User-Agent,
uses bounded response sizes/timeouts and caches the shared feed for one minute.
For polygonless alerts it retrieves the official affected-zone boundaries, using
only the four allowlisted NWS zone paths. No estimated boundary is fabricated.
An alert whose boundary cannot be retrieved still appears in the panel.

Only Actual, effective and unexpired alerts are displayed; cancellation, test
and ended products are excluded. Expiry is also enforced in the browser every
15 seconds. Upstream failure is explicitly stale/unavailable, retaining still-
valid cached warnings at most 30 minutes. It never reports a successful zero-
alert result for an outage. Missing boundaries have their own visible notice.
The feed refreshes every minute and on connectivity restoration. NWS weather
receiving is independent of the disabled government alert-issuing integrations.

Source: https://www.weather.gov/documentation/services-web-api

Tests exercise deduplication, safety ordering, local-only speech, explicit
enablement/lock cancellation, expired data, outage display, boundaryless alerts,
official zone fallback, upstream path restriction, shared cache and authenticated
API access. Production audio output requires a check on the actual vehicle device.
