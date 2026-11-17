# Spec Delta

## Purpose

Resolves the account configuration file, creates the first account on a clean machine, and stores client credentials without unsafe defaults.

## ADDED Requirements

### Requirement: Configuration file resolution
The system SHALL resolve the account config file to `$XDG_CONFIG_HOME/mail/config.json`. When `XDG_CONFIG_HOME` is unset, the system SHALL use the user's home config directory and SHALL NOT create a literal `~` path.

#### Scenario: XDG_CONFIG_HOME is set
- **WHEN** `XDG_CONFIG_HOME=/home/user/.config`
- **THEN** the config file path is `/home/user/.config/mail/config.json`

#### Scenario: XDG_CONFIG_HOME is unset
- **WHEN** `XDG_CONFIG_HOME` is unset and the home directory is `/home/user`
- **THEN** the config file path is `/home/user/.config/mail/config.json`
- **AND** no directory named `~` is created in the working directory

### Requirement: First-run configuration creation
The system SHALL treat a missing or empty config file as an empty account list when adding an account.

#### Scenario: Add the first account
- **WHEN** `mail/config.json` does not exist and the user runs `mail -add`
- **THEN** the new account is appended and written to the config file
- **AND** the command exits successfully

#### Scenario: Empty existing config file
- **WHEN** the config file exists but contains no JSON value
- **THEN** the system starts with an empty account list

### Requirement: Account validation
The system SHALL reject a mail type other than `gmail` and SHALL reject an account whose email already exists in the config.

#### Scenario: Unsupported mail type
- **WHEN** the user enters a mail type other than `gmail`
- **THEN** the system returns an error and does not write the account

#### Scenario: Duplicate account email
- **WHEN** the user adds an account with an email already in the config
- **THEN** the system returns an error and does not write a second entry

### Requirement: Client credential storage
The system SHALL validate the OAuth client secret JSON before storing it, SHALL refuse to overwrite an existing credential file, and SHALL NOT create an empty credential file.

#### Scenario: Invalid client secret file
- **WHEN** the supplied client secret file is not valid OAuth client JSON
- **THEN** the system returns an error and writes no credential file

#### Scenario: Credential file already exists
- **WHEN** a credential file for the client ID already exists
- **THEN** the system returns an error and does not overwrite the file

#### Scenario: Missing credential file during lookup
- **WHEN** `mail` runs and the credential file for the client ID is missing
- **THEN** the system returns an error
- **AND** it does not create an empty credential file

### Requirement: Secret file permissions
The system SHALL write the account config file and credential files with owner-only permissions.

#### Scenario: New credential file permissions
- **WHEN** a client secret is stored
- **THEN** the credential file mode is `0600`

#### Scenario: New account config permissions
- **WHEN** the account config file is written
- **THEN** the config file mode is `0600`
