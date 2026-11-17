package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/Crandel/gmail/internal/accounts"
	"github.com/Crandel/gmail/internal/config"
	"github.com/Crandel/gmail/internal/env"
	"github.com/Crandel/gmail/internal/logging"
	"github.com/Crandel/gmail/internal/mails/googlemail"
)

// result carries one per-account status segment together with the account's
// position so output stays in config order regardless of completion order.
type result struct {
	index   int
	segment string
}

// accountSegment renders one "short:count" status segment.
func accountSegment(acc accounts.Account, count int64) string {
	return fmt.Sprintf("%s:%d", acc.Short, count)
}

// formatOutput concatenates the segments in config order and appends the
// final newline.
func formatOutput(segments []string) string {
	return strings.Join(segments, "") + "\n"
}

// accountCount returns the unread count for one account, or 0 when the mail
// type is unsupported or the account is offline.
func accountCount(ctx context.Context, acc accounts.Account, systemKeyring bool) int64 {
	if acc.MailType != accounts.Gmail {
		slog.ErrorContext(ctx, "unsupported mail type",
			slog.String("account", acc.Short),
			slog.String("mail_type", string(acc.MailType)))
		return 0
	}
	if !googlemail.CheckOnline() {
		slog.ErrorContext(ctx, "can't reach google, reporting zero", slog.String("account", acc.Short))
		return 0
	}
	return googlemail.FetchCount(ctx, acc, systemKeyring)
}

func main() {
	ctx := context.Background()

	debug := env.GetEnv("DEBUG", "false")
	logLevel := slog.LevelInfo
	showSources := false
	if debug == "1" {
		showSources = true
		logLevel = slog.LevelDebug
	}
	logging.InitLogger(logLevel, showSources)

	addUserFlag := flag.Bool("add", false, "Add new user to specific mail client")
	systemKeyringFlag := flag.Bool("system_keyring", false, "Use system keyring instead of file")

	flag.Parse()

	var err error
	if *addUserFlag {
		err = config.AddToConfig()
		if err != nil {
			slog.ErrorContext(ctx, "failed to add user to config", slog.Any("error", err))
		}
		return
	}

	cfgFile, err := config.GetFile()
	if err != nil {
		slog.ErrorContext(ctx, "error during get file", slog.Any("error", err))
		return
	}
	listAccounts, err := config.GetAccounts(cfgFile)
	if err != nil {
		slog.ErrorContext(ctx, "unable to retrieve account list")
		return
	}

	channel := make(chan result, len(listAccounts))
	var wg sync.WaitGroup
	for i, acc := range listAccounts {
		wg.Add(1)
		go func(i int, acc accounts.Account) {
			defer wg.Done()
			channel <- result{index: i, segment: accountSegment(acc, accountCount(ctx, acc, *systemKeyringFlag))}
		}(i, acc)
	}
	wg.Wait()
	close(channel)

	segments := make([]string, len(listAccounts))
	for res := range channel {
		segments[res.index] = res.segment
	}
	fmt.Print(formatOutput(segments))
}
