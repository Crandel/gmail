# Mail checker for unread messages

### For i3status-rust

I use i3/Sway wm and want to have a notification for new emails in i3 status line.

Without **$XDG_CONFIG_HOME/mail_checker/config.json** it will fail.

In order to create `config.json` file with this structure.

```bash
$ bin/gmail -create
```

This command will create sample config file with this content

```json
[
    {
        "mail_type": "gmail",
        "account": "account_name",
        "short_alias":"A",
        "client_id": "<client_id>",
        "client_secret": "<client_secret>"
    }
]
```

Just edit this file.
You could use several gmail accounts to have a personal and work notifications.
Unread count is available as `dbus` message.

# Install
You need installed go.

Just clone the repository and run

```bash
go build -o ./bin/mail ./cmd/main.go
```

Or use go-task for this

```bash
task build
```

You get binary file mail. You can put it to /usr/local/bin and run

You can use multiple accounts, just make shure `$XDG_CONFIG_HOME/mail_checker/config.json` has valid json format
