# LAS MDT for Windows

Independent Windows 10/11 x64 application: WPF window, native menu and status bar, bundled shared MDT interface in Microsoft WebView2, current-user installer, Start menu registration, Windows Apps uninstall entry, and built-in independent updates. There is no Microsoft Store dependency and no separately installed .NET runtime. Microsoft WebView2 Evergreen Runtime is required.

## Offline operation

The Windows application is MDT-only. Sign-in requests the Security officer (field) duty; previously saved broader sessions must switch to that duty before opening the terminal. Other module routes redirect to MDT, the native window blocks other CAD module navigation, and no module switcher or administration links appear. Officer reports remain within `/mdt/reports`. Browser module access and account grants are unchanged.

Sign in online with an account's assigned MDT unit once. The app downloads authorized local person/vehicle records (including normal officer-safety flags and full existing person fields), call details, messages/BOLOs, report metadata and own drafts. The recorded map database is exported consistently from CAD and verified with SHA-256. The same Go map engine then supplies offline vector tiles, fonts, road routes, address search/reverse lookup and saved road closures.

A visible Windows connection/outbox bar identifies offline data. Status changes, mark-outs, call notes, messages/read receipts, BOLO open/ack receipts, own-unit attachment/crew changes, GPS samples, photos/files and report drafts are stored durably in a Windows DPAPI-protected outbox before transmission. Reports can have temporary OFFLINE identifiers until the server assigns their permanent number. Saved work and sign-in survive application updates.

Sync checks the authenticated account and unit again and runs operations through the normal server validators. Command IDs and payload hashes are recorded in the same ledger events as mutations, and reconstructed on replay, so a lost response/restart cannot create duplicate notes, messages, reports or files. Changed unit/call context or report versions leave a visible conflict in the outbox rather than overwriting dispatch. Export/discard controls allow the operator to preserve and review a refused entry. Notes remain bound to their original incident. Historical GPS is audited and does not replace a newer live location.

Offline emergency requests are explicitly NOT SENT and the interface and voice instruct the officer to use radio or phone. Pending emergencies are prioritized on reconnect; ones over five minutes old require review. New remote calls, messages, Enformion results, NWS alerts, current traffic and uncached external imagery require connectivity. The app provides the last downloaded operational data during an outage and never fabricates current remote information.

Data lives under %LOCALAPPDATA%/Lakes Area Security/MDT, independently of application program files. The installer, updater and uninstaller retain that directory. Pending work belongs to its original CAD account and unit.

## Downloads and updates

Accounts offered the field/MDT duty see a Windows download in duty selection. The MDT screen and admin console also provide downloads. The server's public /api/mdt/windows-release endpoint exposes the newest published mdt-v release with its GitHub-computed SHA-256 digest and size; /api/mdt/windows-download redirects only to this repository's installer.

The native app checks at startup and hourly, and Help > Check for updates checks immediately. Install update downloads and verifies the installer inside the app, then installs the new version and reopens it. It never asks the user to download the installer again from a website. The installer uses separate version folders and keeps all sign-in, maps, drafts and outbox data.

The updater trusts only the fixed LAS HTTPS manifest endpoint and this repository's GitHub release asset path, validates version/size/SHA-256, and refuses untrusted download origins. An Authenticode signing certificate has not been configured; no trusted-publisher signature is claimed.

## Release workflow

.github/workflows/windows-mdt.yml builds the shared committed web interface, WPF application and Go offline map engine on Windows. It runs native protected-storage/restart/update-origin and map-engine IPC checks, builds the NSIS installer, and tests installation/update data preservation before publishing LAS-MDT-Setup-x64.exe to a versioned GitHub release.

Bump desktop/LAS.Mdt.Windows/LAS.Mdt.Windows.csproj Version for each published native update. Published versions are immutable. Web changes included in a native release must be rebuilt into internal/web/dist before building the installer. Go backend checks and shared frontend unit tests cover unit binding, duplicate replay, conflicts, cached local searches and offline drafts.
