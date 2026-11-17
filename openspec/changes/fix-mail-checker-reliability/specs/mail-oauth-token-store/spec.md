# Spec Delta

## Purpose

Persists OAuth tokens with per-account keys, refreshes expired tokens, and serializes storage access so concurrent accounts do not lose credentials.

## ADDED Requirements

### Requirement: Token backend selection
The system SHALL use the system keyring when the system-keyring option is enabled, and a file-based keyring otherwise.

#### Scenario: System keyring requested
- **WHEN** the system-keyring option is enabled
- **THEN** tokens are read from and written to the OS keyring

#### Scenario: File keyring by default
- **WHEN** the system-keyring option is disabled
- **THEN** tokens are read from and written to a keyring file in the mail cache directory

### Requirement: Per-account token key
The system SHALL build the token key from the account identity, not from the client ID alone.

#### Scenario: Two accounts share a client ID
- **WHEN** two configured accounts use the same client ID
- **THEN** each account has a separate token entry
- **AND** authorizing one account does not overwrite the other's token

### Requirement: Missing entry error
The system SHALL return a non-nil error when a token key is not present in the keyring.

#### Scenario: Key absent
- **WHEN** the caller requests a key that has no entry
- **THEN** the keyring reader returns an error
- **AND** the returned token is nil or empty

### Requirement: Token upsert
The system SHALL replace an existing entry when saving a token for the same key.

#### Scenario: Save over an existing token
- **WHEN** a token is saved for a key that already has an entry
- **THEN** the new token replaces the old value
- **AND** the save returns no error

### Requirement: Token refresh
The system SHALL refresh an expired token with its refresh token and persist the refreshed token. If refresh fails, the system SHALL fall back to browser authorization.

#### Scenario: Expired token with a valid refresh token
- **WHEN** the stored token is expired and refresh succeeds
- **THEN** the refreshed token is saved to the keyring
- **AND** the caller receives an authorized client

#### Scenario: Refresh fails
- **WHEN** the stored token is expired and refresh fails
- **THEN** the system starts browser authorization
- **AND** the new token replaces the stored entry

### Requirement: Serialized keyring access
The system SHALL serialize read-modify-write operations on the keyring file.

#### Scenario: Concurrent saves
- **WHEN** two accounts save tokens at the same time
- **THEN** both entries are present in the keyring after both saves finish

### Requirement: Token storage permissions
The system SHALL create the token file with owner-only permissions and the enclosing cache directory with owner-only permissions.

#### Scenario: New token file
- **WHEN** the keyring file is created
- **THEN** the file mode is `0600`

#### Scenario: New cache directory
- **WHEN** the mail cache directory is created
- **THEN** the directory mode is `0700`
