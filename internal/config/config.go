package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Crandel/gmail/internal/accounts"
	"github.com/Crandel/gmail/internal/env"
	"github.com/Crandel/gmail/internal/file"
	"github.com/Crandel/gmail/internal/mails/googlemail"
)

const (
	dir      = "mail"
	fileName = "config.json"

	fileMode = 0600
)

// ErrMailType is an config error type.
var ErrMailType = errors.New("mail type should be only gmail")

type inputReader interface {
	ReadString(delim byte) (string, error)
}

// configDir resolves the directory holding the account config from
// XDG_CONFIG_HOME, falling back to the user's home config directory when the
// variable is unset or empty.
func configDir() (string, error) {
	base := env.GetEnv("XDG_CONFIG_HOME", "")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, dir), nil
}

// configFile resolves the full path of the account config file.
func configFile() (string, error) {
	cfgDir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, fileName), nil
}

// GetFile return io.Reader for config file.
func GetFile() (io.Reader, error) {
	name, err := configFile()
	if err != nil {
		return nil, err
	}
	slog.Debug("GetFile filename " + name)

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// GetAccounts return account list from config file.
func GetAccounts(reader io.Reader) (accounts.ListAccounts, error) {
	listAccounts := accounts.ListAccounts{}
	err := json.NewDecoder(reader).Decode(&listAccounts)
	if err != nil {
		slog.Debug("error during Unmarshal", slog.Any("error", err))
		return listAccounts, err
	}
	return listAccounts, nil
}

// readExistingConfig loads the account list. A missing file or a file that
// holds no JSON value is a valid empty list so the first account can be added
// on a clean machine.
func readExistingConfig(name string) (accounts.ListAccounts, error) {
	origFile, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("config file not found, starting with empty account list")
			return accounts.ListAccounts{}, nil
		}
		slog.Error("error during opening file", slog.Any("error", err))
		return nil, fmt.Errorf("error during opening file: %w", err)
	}
	defer func() {
		if cerr := origFile.Close(); cerr != nil {
			slog.Error("error during closing file", slog.Any("error", cerr))
		}
	}()

	accs, err := GetAccounts(origFile)
	if err != nil {
		if errors.Is(err, io.EOF) {
			slog.Debug("config file holds no value, starting with empty account list")
			return accounts.ListAccounts{}, nil
		}
		slog.Error("error during Unmarshal", slog.Any("error", err))
		return nil, fmt.Errorf("error during Unmarshal: %w", err)
	}
	return accs, nil
}

// accountExists reports whether the email is already configured. Mailbox
// addresses are not case sensitive, so both sides are normalized.
func accountExists(list accounts.ListAccounts, email string) bool {
	want := strings.ToLower(email)
	for i := range list {
		if strings.ToLower(list[i].Email) == want {
			return true
		}
	}
	return false
}

// writeConfig stores the account list as indented JSON with owner-only
// permissions.
func writeConfig(name string, list accounts.ListAccounts) error {
	newJSON, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		slog.Error("error during marshaling", slog.Any("error", err))
		return fmt.Errorf("error during marshaling: %w", err)
	}
	if err := os.WriteFile(name, newJSON, fileMode); err != nil {
		slog.Error("error during writing string", slog.Any("error", err))
		return fmt.Errorf("error during writing string: %w", err)
	}
	return nil
}

// AddToConfig adds new user to existed config file.
func AddToConfig() error {
	cfgDir, err := configDir()
	if err != nil {
		return err
	}
	if err := file.CreateDirectory(cfgDir); err != nil {
		return err
	}

	name, err := configFile()
	if err != nil {
		return err
	}
	reader := bufio.NewReader(os.Stdin)
	listAccounts, err := readExistingConfig(name)
	if err != nil {
		return err
	}
	newAccount, err := addNewUser(reader)
	if err != nil {
		slog.Error("An error occurred while reading input. Please try again", slog.Any("error", err))
		return err
	}
	if accountExists(listAccounts, newAccount.Email) {
		return fmt.Errorf("account with '%s' already exists", newAccount.Email)
	}
	if newAccount.MailType == accounts.Gmail {
		fmt.Println("Please specify OAuth 2.0 Client config file")
		clientConfigPath, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		clientConfigPath = strings.TrimSpace(clientConfigPath)
		err = googlemail.SetConfig(newAccount.ClientID, clientConfigPath)
		if err != nil {
			return err
		}
	}

	listAccounts = append(listAccounts, newAccount)
	slog.Debug("Accounts list", slog.Any("list", listAccounts))

	return writeConfig(name, listAccounts)
}

func addNewUser(reader inputReader) (accounts.Account, error) {
	fmt.Println("Please add short alias for this account")
	mailAlias, err := reader.ReadString('\n')
	if err != nil {
		return accounts.Account{}, err
	}
	mailAlias = strings.TrimSpace(mailAlias)

	fmt.Println("Please add mail type. Available types are: gmail")
	mailType, err := reader.ReadString('\n')
	if err != nil {
		return accounts.Account{}, err
	}
	mailType = strings.TrimSpace(mailType)
	mailT := accounts.MailType(mailType)
	if mailT != accounts.Gmail {
		return accounts.Account{}, ErrMailType
	}

	fmt.Println("Please add email address")
	email, err := reader.ReadString('\n')
	if err != nil {
		return accounts.Account{}, err
	}
	email = strings.TrimSpace(email)

	fmt.Println("Please add oauth2 clientId")
	clientId, err := reader.ReadString('\n')
	if err != nil {
		return accounts.Account{}, err
	}
	clientId = strings.TrimSpace(clientId)

	return accounts.Account{
		Short:    mailAlias,
		MailType: mailT,
		Email:    email,
		ClientID: clientId,
	}, nil
}
