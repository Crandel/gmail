# Design

## Context

See `proposal.md` for motivation. The current implementation makes one goroutine per Gmail account but reads a fixed number of results from an unbuffered channel, so any early return or skipped account blocks the process. The keyring file is read and rewritten without a lock, the token key uses only the client ID, and the OAuth callback uses the global default mux and a fixed port. The config path is computed at package init with a literal `~`. Credentials are written with `0644`/`0666`.

## Goals / Non-Goals

**Goals:**

- Guarantee one result per account and a bounded run.
- Make every failure path observable and terminating.
- Make token storage concurrency-safe within one process.
- Store secrets with owner-only permissions.
- Keep the CLI flags and the `short:count` output shape compatible.

**Non-Goals:**

- Cross-process locking of the keyring file.
- A new output format (for example JSON or dbus).
- Retry logic for transient API failures.
- Changes to the OAuth scopes or the Gmail API surface.

## Decisions

### Result collection: buffered channel plus WaitGroup

Allocate a channel with capacity equal to the account count. Each goroutine computes a `short:count` string through a function that has a single exit path, then sends exactly once. After starting every account goroutine, use `sync.WaitGroup` and close the channel after the wait completes. The main loop then ranges over the channel.

- Rationale: no receive can block, no early return can drop a result, and closing after `Wait` removes the "send on closed channel" risk.
- Alternative considered: keep an unbuffered channel and always read N values. Rejected because a missing send still hangs the process.
- Alternative considered: `golang.org/x/sync/errgroup` and a result slice. Rejected because failures are not meant to abort the run, and it adds a dependency.

### Empty config is a valid empty list

`readExistingConfig` returns an empty `accounts.ListAccounts` when the file is absent (`os.IsNotExist`) or contains only whitespace (`io.EOF` from the decoder). Other decode errors are still returned.

- Rationale: matches the documented first-run flow and avoids creating a broken file.
- Alternative: always create a `[]` file first. Rejected because it adds a write before validation.

### Config path resolution is a function

Replace the package-level `configDir`/`filename` vars with a resolver that reads `XDG_CONFIG_HOME` through `internal/env`, falls back to `os.UserHomeDir()`, and joins with `filepath.Join`. The cache directory uses the same approach.

- Rationale: removes init-time environment reads, fixes the literal `~` path, and makes the package testable with per-test environment values.
- Alternative: keep globals and expand `~`. Rejected because package init still freezes the value.

### Account identity is the email

Duplicate detection compares normalized (lowercase) email, not `client_id`. The token key combines the client ID and email, for example `gmailTokenKey:<client_id>:<email>`.

- Rationale: one OAuth client can serve several accounts; the existing client-ID key would make them overwrite each other.
- Alternative: key by email only. Rejected because the same mailbox can be configured with a different client ID.

### Token lifecycle: read, refresh, then authorize

The flow becomes: read the entry; if missing, authorize; if expired, refresh and upsert; if refresh fails, authorize. Remove the unreachable `!token.Valid()` block. `SetEntry` becomes an upsert and no longer errors on an existing key.

- Rationale: refresh must replace the stored token; the old "already exists" error made refresh impossible.
- Alternative: delete then set. Rejected because it leaves a window with no token and complicates atomicity.

### Keyring locking: process-level mutex

Each file keyring handler is registered by absolute file path. A package-level registry returns a shared handler, and the handler holds a `sync.Mutex` around the read-modify-write sequence. Writes go to a temporary file in the same directory and are renamed into place.

- Rationale: goroutines for different accounts share one file, so per-instance locks are insufficient.
- Alternative: OS file locking (`flock`). Rejected because it is platform-specific and the tool is a short-lived CLI.
- Trade-off: two processes can still race; documented as a non-goal.

### Permissions

Create directories with `0700`, files with `0600`, and chmod existing directories/files to these modes when they are already present. Credential files are validated, then written with `0600`. `GetConfig` no longer creates an empty file; a missing file is an error.

- Rationale: OAuth tokens grant mailbox access, so they must not be readable by other users.
- Alternative: `0600` only on creation. Rejected because existing loose files stay loose.

### OAuth callback server

Build a dedicated `http.ServeMux`, set it as the server handler, listen on `127.0.0.1:0`, derive the redirect URL from the bound address, generate a random `state`, and verify it in the callback. The listener starts before the auth URL is printed so the port is known. The server shuts down in a `defer`.

- Rationale: removes global mux registration panics and fixed-port collisions, and allows parallel authorization.
- Alternative: keep the fixed port and serialize authorization. Rejected because two accounts can need authorization in one run.

### Reordering of GetClient

`GetGMailCount` returns immediately when `libGmail.NewService` fails and emits a `short:0` segment. All early returns funnel into a deferred segment send.

## Risks / Trade-offs

- **Token key migration** → On lookup, fall back to the legacy `gmailTokenKey:<client_id>` entry, re-key it under the new format, and delete the legacy entry. Users keep their existing authorization.
- **Ephemeral redirect port** → Desktop OAuth clients accept loopback redirects on any port. If a user created a Web application client with a fixed redirect URI, authorization fails; document this in the README.
- **Upsert changes keyring semantics** → Any caller that relied on the "already exists" error changes behavior. Only `saveToken` relies on it, so the blast radius is small.
- **Process-level lock only** → Two concurrent `mail` runs can still lose an update. Acceptable for a short-lived status command.
- **Output still concatenates segments** → Config order and the `short:count` format are preserved; consumers must not rely on separators.
- **Failure looks like zero unread** → A failed fetch emits `0`, which is indistinguishable from an empty inbox. The error log is the only signal; this keeps the status line parseable.

## Migration Plan

1. Deploy the new binary; existing `config.json` and keyring files remain readable.
2. On first token lookup, migrate legacy client-ID keys to the new email-based keys.
3. Rollback: previous binaries still read the legacy keys, but re-keyed entries are ignored; users re-authorize.
