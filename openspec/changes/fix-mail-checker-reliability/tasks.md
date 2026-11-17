# Tasks

## 1. Config resolution and first-run creation

- [x] 1.1 Replace the package-level `configDir`/`filename` vars with a resolver that reads `XDG_CONFIG_HOME` via `internal/env`, falls back to `os.UserHomeDir()`, and joins with `filepath.Join`; verify with `go test ./internal/config` and a new test for set/unset `XDG_CONFIG_HOME` that no `~` directory is created.
- [x] 1.2 Change `readExistingConfig` to return an empty account list when the file is absent or decodes to `io.EOF`; verify `mail -add` succeeds against a temp config dir and that `TestReadExistingConfigMissingFile` passes.
- [x] 1.3 Compare accounts by normalized (lowercase) email instead of `client_id`; verify a new `TestDuplicateEmailRejected` passes.
- [x] 1.4 Fix the inverted, dead assertion in `TestAddNewUser` and make `MockReader` return `min(len(p), len(input))` plus `io.EOF`; verify `go test -run TestAddNewUser ./internal/config` passes a real equality check.
- [x] 1.5 Write the account config file with `0600`; verify `TestConfigFilePermissions` checks the mode.

## 2. Credential files and permissions

- [ ] 2.1 Remove empty-file creation from `GetConfig` and return an error when the credential file is missing; verify `TestGetConfigMissingFile` passes and no file is created.
- [ ] 2.2 Make `SetConfig` validate the JSON, refuse to overwrite, and write with `0600`; verify `TestSetConfigPermissions` and `TestSetConfigRefusesOverwrite` pass.
- [x] 2.3 Create the cache directory with `0700` and chmod existing directories to `0700`; verify `TestCacheDirPermissions` checks the mode.

## 3. Keyring storage

- [x] 3.1 Register file keyrings by absolute path and guard each with a `sync.Mutex`, writing to a temp file then renaming; verify two concurrent `SetEntry` calls under `go test -race ./internal/keyring` keep both entries.
- [x] 3.2 Make `SetEntry` an upsert that replaces existing values; verify `TestSetEntryUpsert` passes.
- [x] 3.3 Return a non-nil error from `GetEntry` when the key is absent; verify `TestGetEntryMissing` checks the error and empty value.
- [x] 3.4 Create the keyring file with `0600` and the cache directory with `0700`; verify `TestKeyringFilePermissions` checks both modes.
- [x] 3.5 Update `internal/config/config_test.go` and `internal/mails/googlemail/oauth2_test.go` mock keyrings to the upsert contract; verify `go test ./internal/...` passes.

## 4. Token lifecycle and migration

- [x] 4.1 Build the token key from client ID and email, and pass the email into the token lookup; verify a new test with two accounts sharing a client ID stores two entries.
- [x] 4.2 Read the token, refresh when expired, upsert the refreshed token, and fall back to browser authorization when refresh fails; verify `TestRefreshExpiredToken` and `TestRefreshFailsFallsBack` pass.
- [x] 4.3 On lookup, fall back to the legacy `gmailTokenKey:<client_id>` entry, re-key it, and delete the legacy entry; verify `TestLegacyTokenMigration` passes.
- [x] 4.4 Document the token-key migration and the re-authorization behavior after rollback in `README.md`; verify the text is present.

## 5. OAuth callback server

- [x] 5.1 Build a dedicated `http.ServeMux`, listen on `127.0.0.1:0`, derive the redirect URL from the bound address, generate a random `state`, and verify it in the callback; verify `TestStateValidation` passes and a manual two-account authorization run does not panic.
- [x] 5.2 Shut down the callback server with `defer` and bound the authorization wait with a context timeout; verify the manual run exits instead of hanging when the callback is never called.
- [x] 5.3 Document the loopback-redirect requirement for desktop OAuth clients in `README.md`; verify the text is present.

## 6. Unread status collection and output

- [x] 6.1 Replace `CheckOnline` with a version that guards `resp == nil`, closes the body, and uses a client with a timeout; verify `TestCheckOnlineError` passes against an `httptest` server and no panic occurs.
- [x] 6.2 Return immediately after a `NewService` error and funnel every failure into a `short:0` result; verify `TestFetchFailureReportsZero` passes.
- [x] 6.3 Rework `cmd/main.go` to use a buffered channel and `sync.WaitGroup`, sending exactly one result per account and closing after the wait; verify a run with a non-Gmail account exits promptly instead of hanging.
- [x] 6.4 Preserve the `short:count` concatenated output and final newline; verify `TestOutputFormat` produces `A:3B:5\n`.

## 7. Integration verification

- [ ] 7.1 Run `go build -o ./bin/mail ./cmd/main.go`, `go test -p 1 -count=1 -cover ./...`, `golangci-lint run`, and `gofmt -l .`; verify all commands exit successfully with no reported issues.

## Workflow follow-up

- Archive the change after the project's review requirements are satisfied.
- Verify the archived spec matches the implemented behavior.
