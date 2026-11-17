package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Crandel/gmail/internal/accounts"
	"github.com/Crandel/gmail/internal/file"
)

// MockBufioReader is a struct that implements the bufio.Reader interface for testing.
type MockBufioReader struct {
	input []string
}

func (mr *MockBufioReader) ReadString(delim byte) (string, error) {
	if len(mr.input) > 0 {
		s := mr.input[0]
		mr.input = mr.input[1:]
		return s + string(delim), nil
	}
	return "", errors.New("no more input")
}

func TestAddNewUser(t *testing.T) {
	t.Parallel()
	type testAccount struct {
		Short    string
		Type     string
		Email    string
		ClientID string
	}

	// Define your test data: alias, type, email and clientId in that order.
	testData := []struct {
		testAccount
		error
	}{
		{
			testAccount{"alias1", "gmail", "example@email.com", "clientId1"},
			nil,
		},
		{
			testAccount{"alias2", "notGmail", "anotherExample@email.com", "clientId2"},
			ErrMailType,
		},
	}

	for _, td := range testData {
		// Create a mock reader that returns the test data when ReadString is called.
		mockReader := &MockBufioReader{input: []string{td.Short, td.Type, td.Email, td.ClientID}}
		account, err := addNewUser(mockReader)
		if td.error != nil {
			if err == nil {
				t.Errorf("addNewUser returned nil error, want '%+v'", td.error)
				continue
			}
			if err.Error() != td.Error() {
				t.Errorf("addNewUser returned different error: got '%+v', want '%+v'", err, td.error)
			}
			continue
		}
		if err != nil {
			t.Errorf("addNewUser returned unexpected error: %+v", err)
			continue
		}
		want := accounts.Account{
			Short:    td.Short,
			MailType: accounts.MailType(td.Type),
			Email:    td.Email,
			ClientID: td.ClientID,
		}
		if !reflect.DeepEqual(account, want) {
			t.Errorf("addNewUser returned incorrect account: got %+v, want %+v", account, want)
		}
	}
}

// MockReader is a struct that implements the io.Reader interface for testing.
type MockReader struct {
	input string
}

func (mr *MockReader) Read(p []byte) (n int, err error) {
	n = copy(p, []byte(mr.input))
	mr.input = mr.input[n:]
	if len(mr.input) == 0 {
		err = io.EOF
	}
	return n, err
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name     string
		input    io.Reader
		expected accounts.ListAccounts
		err      error
	}

	var jsonStr = `[{"short":"alias1", "mail_type":"gmail", "email":"email@example.com", "client_id":"clientId1"}, {"short":"alias2", "mail_type":"gmail", "email":"another@example.com", "client_id":"clientId2"}]`
	var accountsList = []accounts.Account{
		{
			Short:    "alias1",
			MailType: accounts.Gmail,
			Email:    "email@example.com",
			ClientID: "clientId1",
		},
		{
			Short:    "alias2",
			MailType: accounts.Gmail,
			Email:    "another@example.com",
			ClientID: "clientId2",
		},
	}

	tests := []testCase{{
		name:     "GetAccounts test",
		input:    &MockReader{jsonStr},
		expected: accountsList,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotAccounts, err := GetAccounts(tc.input)
			if err != nil {
				t.Errorf("GetAccounts() error = %v, wantErr %v", err, tc.err)
				return
			}
			if !reflect.DeepEqual(gotAccounts, tc.expected) {
				t.Errorf("GetAccounts() got =%v, want =%v", gotAccounts, tc.expected)
			}
		})
	}
}

// TestConfigPathResolution covers tasks for the config path resolver: a set
// XDG_CONFIG_HOME wins, an unset one falls back to the user's home directory,
// and no literal "~" directory is ever created.
func TestConfigPathResolution(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("unable to read working directory: %v", err)
	}

	t.Run("XDG_CONFIG_HOME is set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/home/user/.config")
		got, err := configFile()
		if err != nil {
			t.Fatalf("configFile() error = %v", err)
		}
		want := filepath.Join("/home/user/.config", "mail", "config.json")
		if got != want {
			t.Errorf("configFile() = %q, want %q", got, want)
		}
	})

	t.Run("XDG_CONFIG_HOME is unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/user")
		got, err := configFile()
		if err != nil {
			t.Fatalf("configFile() error = %v", err)
		}
		want := filepath.Join("/home/user/.config", "mail", "config.json")
		if got != want {
			t.Errorf("configFile() = %q, want %q", got, want)
		}
	})

	t.Run("no literal ~ directory is created", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", home)

		cfgDir, err := configDir()
		if err != nil {
			t.Fatalf("configDir() error = %v", err)
		}
		if err := file.CreateDirectory(cfgDir); err != nil {
			t.Fatalf("CreateDirectory() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(cwd, "~")); err == nil {
			t.Error("a literal ~ directory was created in the working directory")
		}
		if _, err := os.Stat(filepath.Join(home, ".config", "mail")); err != nil {
			t.Errorf("config directory was not created under home: %v", err)
		}
	})
}

func TestReadExistingConfigMissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		accs, err := readExistingConfig(filepath.Join(dir, "absent.json"))
		if err != nil {
			t.Fatalf("readExistingConfig() error = %v, want nil", err)
		}
		if len(accs) != 0 {
			t.Errorf("readExistingConfig() = %v, want empty list", accs)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		t.Parallel()
		name := filepath.Join(dir, "empty.json")
		if err := os.WriteFile(name, []byte("  \n"), 0600); err != nil {
			t.Fatalf("unable to write empty file: %v", err)
		}
		accs, err := readExistingConfig(name)
		if err != nil {
			t.Fatalf("readExistingConfig() error = %v, want nil", err)
		}
		if len(accs) != 0 {
			t.Errorf("readExistingConfig() = %v, want empty list", accs)
		}
	})
}

func TestDuplicateEmailRejected(t *testing.T) {
	t.Parallel()
	list := accounts.ListAccounts{
		{Short: "A", MailType: accounts.Gmail, Email: "User@Example.com", ClientID: "clientId1"},
	}

	if !accountExists(list, "user@example.com") {
		t.Error("accountExists() = false, want true for a case-insensitive duplicate")
	}
	if !accountExists(list, "User@Example.com") {
		t.Error("accountExists() = false, want true for the identical email")
	}
	if accountExists(list, "other@example.com") {
		t.Error("accountExists() = true, want false for an unknown email")
	}
}

func TestConfigFilePermissions(t *testing.T) {
	t.Parallel()
	name := filepath.Join(t.TempDir(), "config.json")
	list := accounts.ListAccounts{
		{Short: "A", MailType: accounts.Gmail, Email: "a@example.com", ClientID: "clientId1"},
	}

	if err := writeConfig(name, list); err != nil {
		t.Fatalf("writeConfig() error = %v", err)
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatalf("unable to stat config file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file mode = %o, want 0600", perm)
	}
}
