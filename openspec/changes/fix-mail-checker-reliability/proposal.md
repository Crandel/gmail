# Proposal

## Why

The checker can hang forever, crash on a nil HTTP response, fail on a fresh install, and store OAuth secrets with world-readable permissions. These defects make the tool unusable or unsafe on first run and after token expiry.

## What Changes

- Send exactly one status result per configured account, so the output loop can never block. **BREAKING**: output now includes a result for accounts that fail, instead of omitting or hanging.
- Check network reachability safely: guard the nil response, close the body, and apply a timeout.
- Treat a missing or empty config file as an empty account list, so `mail -add` works on a clean machine.
- Make token storage an upsert keyed by account email, so an expired or refreshed token replaces the old entry.
- Serialize keyring file access to prevent concurrent read-modify-write loss.
- Write token and credential files with `0600` and cache directories with `0700`.
- Serve the OAuth callback from a local mux and an ephemeral port, so two accounts can authorize at the same time.
- Return early when `gmail.NewService` fails, instead of dereferencing a nil service.
- Return an error, not a nil error, when a keyring entry is missing.

## Capabilities

### New Capabilities

- `mail-account-config`: configuration file resolution, first-run creation, account entry, and credential file permissions.
- `mail-oauth-token-store`: token persistence, keyring backend selection, refresh, and per-account keying.
- `mail-unread-status`: network availability check, per-account unread fetch, and status output.

### Modified Capabilities

<!-- None: openspec/specs is empty, so every capability is new. -->

## Impact

- `cmd/main.go`: result collection and channel handling.
- `internal/config/config.go`: config path defaults, first-run read, account uniqueness.
- `internal/env/env.go`: environment access stays the only allowed entry point.
- `internal/file/file.go`: cache directory and permissions.
- `internal/keyring/file.go`, `internal/keyring/keyring.go`: locking, upsert, error semantics, file modes.
- `internal/mails/googlemail/config.go`: credential file creation and permissions.
- `internal/mails/googlemail/gmail.go`: online check and early returns.
- `internal/mails/googlemail/oauth2.go`: token refresh, callback server, token key.
- Tests: `internal/config/config_test.go`, `internal/mails/googlemail/oauth2_test.go`.
- No public API change. The CLI flags and output shape stay the same.
