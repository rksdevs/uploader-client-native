# Cross-Repo RFC: Guild Data Sync and Native Uploader Phase 1

Revision: `v1.0`  
Date: `2026-03-08`  
Owners: `web-app`, `uploader-client-native`, `wow-addon (new repo)`

## 1) Decision Summary

- Guild setup/sync v1 is **manual export/import only**.
- No native-uploader login is required for guild sync in v1.
- Guild data import is restricted to guild `OWNER`, `STAFF`, and platform `ADMIN`.
- Import behavior is **non-destructive and additive** by default.
- Native uploader Phase 1 ships now:
  - refreshed UI
  - dynamic server list
  - `View All Logs` button

## 2) Scope by Repo

### Web App (`log-parser-v2`)

- Add uploader server list API from the canonical `Server` table.
- Keep upload processing APIs unchanged.
- Next phase (not part of this implementation): Guild Import Wizard for manual addon exports.

### Native Uploader (`uploader-client-native`)

- Align UI aesthetics with current web app style direction.
- Load server list from web API with local fallback.
- Add `View All Logs` action that opens `${baseUrl}/logs`.

### WoW Addon (new repo)

- v1 capability target: export guild snapshot payload for manual upload in web app.
- No direct network communication expected in v1.

## 3) Manual Guild Export/Import Model

1. Addon exports guild snapshot payload (SavedVariables/JSON export flow).
2. Guild owner/staff uploads the payload in web app import wizard.
3. Web app runs dry-run preview and conflict report.
4. User confirms apply operation.

### Required Permission Model

- Allowed: guild `OWNER`, guild `STAFF`, platform `ADMIN`.
- Disallowed: guild member without elevated role, unauthenticated users.

## 4) Non-Destructive Import Guarantees

- No hard delete of existing guild players during import.
- No full roster replacement.
- Existing data is preserved unless explicitly opted into override behavior.
- Duplicate-safe insert semantics for links and import records.
- Every run produces an import report and audit record.

### Unknown/Unmatched Player Handling

- Characters not resolvable to `Player` are stored in a pending/unmatched bucket.
- They are not silently dropped.
- They are not auto-created in `Player` unless a later explicit resolve step is executed.

## 5) Future Bridge Contract (Deferred)

This RFC reserves a future automated bridge mode (`addon -> native -> web`) but does not enable it in v1.

Reserved contract goals:
- Versioned payload envelope
- Idempotent snapshot IDs
- Retry-safe acknowledgements
- Auditability and role-safe authorization

## 6) Native Uploader Phase 1 Acceptance Criteria

- Uploader server dropdown includes all servers from the web app `Server` table.
- Newly added servers appear without requiring ranked data presence.
- `View All Logs` opens the web logs listing in default browser.
- Existing upload flow continues to work:
  - preprocess
  - instance selection
  - enqueue
  - status polling and notification

## 7) Rollout and Compatibility

- Rollout mode: incremental milestones.
- Phase 1 is backward-compatible with existing upload endpoints.
- No migration required for existing uploader users besides updating desktop client binary.

## 8) Open Follow-Ups (Next Milestones)

- Implement web Guild Import Wizard + import tables.
- Define addon export schema with explicit versioning.
- Add import conflict resolution UX for unmatched players.
- Add operational runbook for guild import support cases.

