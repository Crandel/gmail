# Mail checker for unread messages

### For i3status-rust

I use i3/Sway wm and want to have a notification for new emails in i3 status line.

## Requirements
You need to create [OAuth client ID credentials](https://developers.google.com/workspace/guides/create-credentials#desktop-app) and

In order to create `$XDG_CONFIG_HOME/mail/config.json` file run these commands:

```bash
$ go build -o ./bin/mail ./cmd/main.go
$ bin/mail -add
```

Answering few questions this command will create sample config file with this content:

```json
[
    {
        "client_id":     "<client_id>",
        "mail_type":     "gmail",
        "email":         "<email address>",
        "short":         "A",
    }
]
```

Just edit this file.
You could use several gmail accounts to have a personal and work notifications.
Before adding new account you need to have `<client id>.json` file, with your [credentials](https://developers.google.com/identity/protocols/oauth2/native-app) and specify the path to this file during account creation.
Unread count is available as `dbus` message.

# Install
You need to have an installed go.

Just clone the repository and run

```bash
go build -o ./bin/mail ./cmd/main.go
```

Or use go-task for this

```bash
task build
```

You get binary file mail. You can put it to /usr/local/bin and run

You can use multiple accounts, just make sure `$XDG_CONFIG_HOME/mail/config.json` has valid json format

## OAuth redirect requirement

The authorization callback listens on an ephemeral loopback port
(`http://127.0.0.1:<random port>/callback`). Because of this you must create a
**Desktop application** OAuth client, which accepts loopback redirects on any
port. A **Web application** client with a fixed redirect URI will not work:
Google rejects the ephemeral redirect URI and authorization fails.

## Token storage and migration

OAuth tokens are stored in the keyring under a per-account key
(`gmailTokenKey:<client_id>:<email>`), so several accounts can share one OAuth
client without overwriting each other's tokens.

On the first token lookup an entry stored under the previous
`gmailTokenKey<client_id>` key is copied to the new per-account key and the
legacy entry is deleted. Existing authorizations keep working after the upgrade.

If you roll back to an older release, it only reads the legacy key format.
Already re-keyed entries are ignored by the old binary, so you will be asked to
authorize in the browser again.
