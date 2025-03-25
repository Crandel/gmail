package config

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/Crandel/gmail/internal/accounts"
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
			if err.Error() != td.Error() {
				t.Errorf("addNewUser returned different error: got '%+v', want '%+v'", err, td.error)
			}
			continue
		}
		if reflect.DeepEqual(account, td.testAccount) {
			t.Errorf("addNewUser returned incorrect account: got %+v, want Short=%s, Type=%s, Email=%s, ClientId =%s",
				account,
				td.Short, td.Type, td.Email, td.ClientID)
		}
	}
}

// MockReader is a struct that implements the io.Reader interface for testing.
type MockReader struct {
	input string
}

func (mr MockReader) Read(p []byte) (n int, err error) {
	copy(p, []byte(mr.input))
	return len(mr.input), nil
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
		input:    MockReader{jsonStr},
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
