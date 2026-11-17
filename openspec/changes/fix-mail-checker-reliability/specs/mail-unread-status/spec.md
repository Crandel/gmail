# Spec Delta

## Purpose

Reports the unread Gmail count for every configured account and never blocks or crashes on network, credential, or API failures.

## ADDED Requirements

### Requirement: Safe connectivity check
The system SHALL check Gmail reachability without dereferencing a nil HTTP response, SHALL close the response body, and SHALL bound the check with a timeout.

#### Scenario: Request fails
- **WHEN** the connectivity request returns an error
- **THEN** the account is treated as offline
- **AND** the process does not panic

#### Scenario: Request succeeds
- **WHEN** the connectivity request returns a 200 or 301 response
- **THEN** the account is treated as online
- **AND** the response body is closed

#### Scenario: Unreachable network
- **WHEN** the network does not respond
- **THEN** the check ends after its timeout
- **AND** the process continues

### Requirement: One result per account
The system SHALL produce exactly one status segment for every configured account, including accounts that fail.

#### Scenario: Account fails
- **WHEN** one account cannot load credentials, connect, or fetch counts
- **THEN** that account still produces exactly one status segment
- **AND** the output loop does not block

#### Scenario: Non-Gmail account
- **WHEN** the config contains an account with an unsupported mail type
- **THEN** that account still produces exactly one status segment

### Requirement: Failed fetch reports zero
The system SHALL report a count of zero for an account when credentials, connectivity, or the Gmail API call fails, and SHALL log the failure.

#### Scenario: Gmail API error
- **WHEN** the unread-label request fails
- **THEN** the status segment for that account has count `0`
- **AND** the error is logged

#### Scenario: Service creation fails
- **WHEN** the Gmail service cannot be created
- **THEN** the status segment for that account has count `0`
- **AND** no nil service is used

### Requirement: Status output format
The system SHALL emit one `short:count` segment per account, concatenated in config order, and SHALL write a final newline.

#### Scenario: Two accounts
- **WHEN** the config has accounts `A` and `B` and both read 3 and 5 unread
- **THEN** the output is `A:3B:5` followed by a newline
