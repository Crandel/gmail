package googlemail

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	libGmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/Crandel/gmail/internal/accounts"
)

const (
	url           = "https://mail.google.com"
	onlineTimeout = 5 * time.Second
)

// onlineClient bounds the connectivity check so an unreachable network
// returns offline instead of blocking forever.
var onlineClient = &http.Client{Timeout: onlineTimeout}

// CheckOnline will check google servers availability.
func CheckOnline() bool {
	return checkOnline(onlineClient, url)
}

// checkOnline never dereferences a nil response and closes the response body,
// so a failing request reports offline instead of panicking.
func checkOnline(client *http.Client, checkURL string) bool {
	resp, err := client.Get(checkURL)
	if err != nil {
		slog.Debug("online check failed", slog.Any("error", err))
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMovedPermanently
}

// FetchCount returns the unread count for the account. Any failure in
// credentials, client creation, or the API call returns 0 and is logged, so
// every account still produces exactly one status segment.
func FetchCount(ctx context.Context, acc accounts.Account, systemKeyring bool) int64 {
	config, err := GetConfig(acc.ClientID)
	if err != nil {
		slog.Error("can't load config from credentials", slog.Any("error", err))
		return 0
	}
	client, err := GetClient(ctx, config, acc.Email, systemKeyring)
	if err != nil {
		slog.Error("can't load client for client id: "+acc.ClientID, slog.Any("error", err))
		return 0
	}

	srv, err := libGmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		slog.Error("Unable to retrieve Gmail client", slog.Any("error", err))
		return 0
	}

	user := "me"
	ll, err := srv.Users.Labels.Get(user, "UNREAD").Do()
	if err != nil {
		slog.Error("label get", slog.Any("error", err))
		return 0
	}
	return ll.MessagesUnread
}
