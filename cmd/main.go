package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Crandel/gmail/internal/accounts"
	"github.com/Crandel/gmail/internal/config"
	"github.com/Crandel/gmail/internal/env"
	"github.com/Crandel/gmail/internal/logging"
	"github.com/Crandel/gmail/internal/mails/googlemail"
)

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

	file, err := config.GetFile()
	if err != nil {
		slog.ErrorContext(ctx, "error during get file", slog.Any("error", err))
		return
	}
	listAccounts, err := config.GetAccounts(file)
	if err != nil {
		slog.ErrorContext(ctx, "unable to retrieve account list")
		return
	}

	channel := make(chan string)
	defer close(channel)

	for _, acc := range listAccounts {
		if acc.MailType == accounts.Gmail {
			if googlemail.CheckOnline() {
				go googlemail.GetGMailCount(ctx, channel, acc, *systemKeyringFlag)
			}
		}
	}

	accLen := len(listAccounts)
	counts := make([]string, accLen)
	for i := range accLen {
		counts[i] = <-channel
	}
	fmt.Println(strings.Join(counts, ""))
}
