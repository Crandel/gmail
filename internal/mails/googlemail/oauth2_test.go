package googlemail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

const (
	validKey   = "key"
	missingKey = "missing key"
)

// mockKeyring implements keyringHandler with the upsert contract: SetEntry
// replaces an existing value instead of failing.
//
//	type keyringHandler interface {
//	    GetEntry(key string) (string, error)
//	    SetEntry(key string, data string) error
//	    DeleteEntry(key string) error
//	}
type mockKeyring struct {
	storage map[string]string
}

func (m mockKeyring) GetEntry(key string) (string, error) {
	v, ok := m.storage[key]
	if !ok {
		return "", errors.New("No value for key: " + key)
	}
	return v, nil
}

func (m mockKeyring) SetEntry(key string, data string) error {
	m.storage[key] = data
	return nil
}

func (m mockKeyring) DeleteEntry(key string) error {
	delete(m.storage, key)
	return nil
}

func (m mockKeyring) tokenString(token *oauth2.Token) string {
	tokenByte, _ := json.Marshal(token)
	return string(tokenByte)
}

// neverAuthorize fails the test when browser authorization is started.
func neverAuthorize(t *testing.T) authorizeFunc {
	t.Helper()
	return func(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
		t.Error("browser authorization started, want stored token used")
		return nil, errors.New("browser authorization must not be used")
	}
}

func TestTokenFromKeyring(t *testing.T) {
	t.Parallel()
	validToken := &oauth2.Token{AccessToken: "your_token", Expiry: time.Now().AddDate(1, 0, 0)}
	keyringH := mockKeyring{
		storage: map[string]string{},
	}
	validTokenString := keyringH.tokenString(validToken)
	if err := keyringH.SetEntry(validKey, validTokenString); err != nil {
		t.Fatalf("unable to seed keyring: %v", err)
	}
	tests := []struct {
		name     string
		key      string
		token    *oauth2.Token
		keyringH keyringHandler
		wantErr  bool
	}{
		{
			name:     "Successful retrieval of token",
			key:      validKey,
			token:    validToken,
			keyringH: keyringH,
			wantErr:  false,
		},
		{
			name:     "Retrieve missing key",
			key:      missingKey,
			keyringH: keyringH,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tokenFromKeyring(tt.key, tt.keyringH)
			if tt.wantErr {
				if err == nil {
					t.Errorf("tokenFromKeyring() error = nil, want error for key %q", tt.key)
				}
				if got != nil {
					t.Errorf("tokenFromKeyring() token = %v, want nil on error", got)
				}
				return
			}
			if err != nil {
				t.Errorf("tokenFromKeyring() error = %v, want nil", err)
				return
			}
			if got.AccessToken != tt.token.AccessToken {
				t.Errorf("got token: '%s', expected: '%s'", got.AccessToken, tt.token.AccessToken)
			}
		})
	}
}

func Test_saveToken(t *testing.T) { //nolint:paralleltest // sub-tests share one keyring and need order.
	keyringH := mockKeyring{
		storage: map[string]string{},
	}
	type args struct {
		key     string
		token   *oauth2.Token
		keyring keyringHandler
	}
	tests := []struct {
		name   string
		args   args
		errMsg string
	}{
		{
			name: "Successful save of token",
			args: args{
				key: validKey,
				token: &oauth2.Token{
					AccessToken:  "access_token",
					RefreshToken: "refresh_token",
				},
				keyring: keyringH,
			},
			errMsg: "",
		},
		{
			name: "Save over an existing key replaces the value",
			args: args{
				key: validKey,
				token: &oauth2.Token{
					AccessToken:  "replacement_token",
					RefreshToken: "refresh_token_2",
				},
				keyring: keyringH,
			},
			errMsg: "",
		},
	}
	for _, tt := range tests { //nolint:paralleltest // sub-tests share one keyring and must run in order.
		t.Run(tt.name, func(t *testing.T) {
			err := saveToken(tt.args.key, tt.args.token, tt.args.keyring)
			if tt.errMsg == "" && err != nil {
				t.Fatalf("saveToken() error = %v, want nil", err)
			}
			if tt.errMsg != "" && (err == nil || err.Error() != tt.errMsg) {
				t.Fatalf("saveToken() error = %v, want %q", err, tt.errMsg)
			}
			token, err := tokenFromKeyring(tt.args.key, tt.args.keyring)
			if err != nil {
				t.Fatalf("tokenFromKeyring() error = %v", err)
			}
			if token.AccessToken != tt.args.token.AccessToken {
				t.Errorf("token is invalid. \nWant:\n%v,\ngot:\n%v", tt.args.token, token)
			}
		})
	}
}

// TestTwoAccountsShareClientID verifies each account keeps its own token entry.
func TestTwoAccountsShareClientID(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{ClientID: "shared-cid"}
	kr := mockKeyring{storage: map[string]string{}}
	tokenA := &oauth2.Token{AccessToken: "token-a", Expiry: time.Now().AddDate(1, 0, 0)}
	tokenB := &oauth2.Token{AccessToken: "token-b", Expiry: time.Now().AddDate(1, 0, 0)}
	keyA := tokenKey("shared-cid", "a@example.com")
	keyB := tokenKey("shared-cid", "b@example.com")

	if err := saveToken(keyA, tokenA, kr); err != nil {
		t.Fatalf("saveToken(a) error = %v", err)
	}
	if err := saveToken(keyB, tokenB, kr); err != nil {
		t.Fatalf("saveToken(b) error = %v", err)
	}
	if len(kr.storage) != 2 {
		t.Fatalf("keyring holds %d entries, want 2", len(kr.storage))
	}

	client, err := clientForAccount(context.Background(), cfg, "a@example.com", kr, neverAuthorize(t))
	if err != nil {
		t.Fatalf("clientForAccount() error = %v", err)
	}
	if client == nil {
		t.Fatal("clientForAccount() client = nil")
	}

	gotA, err := tokenFromKeyring(keyA, kr)
	if err != nil || gotA.AccessToken != "token-a" {
		t.Errorf("token a after authorizing account a: %v (err %v), want token-a", gotA, err)
	}
	gotB, err := tokenFromKeyring(keyB, kr)
	if err != nil || gotB.AccessToken != "token-b" {
		t.Errorf("token b was overwritten: %v (err %v), want token-b", gotB, err)
	}
}

// TestRefreshExpiredToken verifies an expired token is refreshed via the
// token endpoint and the refreshed token is upserted into the keyring.
func TestRefreshExpiredToken(t *testing.T) {
	t.Parallel()
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("unable to parse refresh request: %v", err)
		}
		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", r.FormValue("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"refreshed-token","token_type":"Bearer","expires_in":3599}`)
	}))
	defer tokenSrv.Close()

	cfg := &oauth2.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			TokenURL:  tokenSrv.URL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	kr := mockKeyring{storage: map[string]string{}}
	expired := &oauth2.Token{
		AccessToken:  "old-token",
		RefreshToken: "refresh-1",
		Expiry:       time.Now().Add(-time.Hour),
	}
	key := tokenKey("cid", "user@example.com")
	if err := saveToken(key, expired, kr); err != nil {
		t.Fatalf("unable to seed expired token: %v", err)
	}

	authorizeCalled := false
	authorize := func(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
		authorizeCalled = true
		return &oauth2.Token{AccessToken: "browser-token"}, nil
	}

	client, err := clientForAccount(context.Background(), cfg, "user@example.com", kr, authorize)
	if err != nil {
		t.Fatalf("clientForAccount() error = %v", err)
	}
	if client == nil {
		t.Fatal("clientForAccount() client = nil")
	}
	if authorizeCalled {
		t.Error("browser authorization started although refresh succeeded")
	}
	stored, err := tokenFromKeyring(key, kr)
	if err != nil {
		t.Fatalf("unable to read refreshed token: %v", err)
	}
	if stored.AccessToken != "refreshed-token" {
		t.Errorf("stored token = %q, want refreshed-token", stored.AccessToken)
	}
}

// TestRefreshFailsFallsBack verifies a failed refresh falls back to browser
// authorization and the new token replaces the stored entry.
func TestRefreshFailsFallsBack(t *testing.T) {
	t.Parallel()
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`)
	}))
	defer tokenSrv.Close()

	cfg := &oauth2.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			TokenURL:  tokenSrv.URL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	kr := mockKeyring{storage: map[string]string{}}
	expired := &oauth2.Token{
		AccessToken:  "old-token",
		RefreshToken: "revoked-refresh",
		Expiry:       time.Now().Add(-time.Hour),
	}
	key := tokenKey("cid", "user@example.com")
	if err := saveToken(key, expired, kr); err != nil {
		t.Fatalf("unable to seed expired token: %v", err)
	}

	authorizeCalled := false
	authorize := func(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
		authorizeCalled = true
		return &oauth2.Token{AccessToken: "browser-token", Expiry: time.Now().Add(time.Hour)}, nil
	}

	client, err := clientForAccount(context.Background(), cfg, "user@example.com", kr, authorize)
	if err != nil {
		t.Fatalf("clientForAccount() error = %v", err)
	}
	if client == nil {
		t.Fatal("clientForAccount() client = nil")
	}
	if !authorizeCalled {
		t.Error("browser authorization not started after failed refresh")
	}
	stored, err := tokenFromKeyring(key, kr)
	if err != nil {
		t.Fatalf("unable to read stored token: %v", err)
	}
	if stored.AccessToken != "browser-token" {
		t.Errorf("stored token = %q, want browser-token", stored.AccessToken)
	}
}

// TestLegacyTokenMigration verifies a client-ID-only entry is re-keyed to the
// per-account key and the legacy entry is deleted.
func TestLegacyTokenMigration(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{ClientID: "cid"}
	kr := mockKeyring{storage: map[string]string{}}
	legacy := &oauth2.Token{AccessToken: "legacy-token", Expiry: time.Now().AddDate(1, 0, 0)}
	legacyKey := legacyTokenKey("cid")
	if err := saveToken(legacyKey, legacy, kr); err != nil {
		t.Fatalf("unable to seed legacy token: %v", err)
	}

	client, err := clientForAccount(context.Background(), cfg, "user@example.com", kr, neverAuthorize(t))
	if err != nil {
		t.Fatalf("clientForAccount() error = %v", err)
	}
	if client == nil {
		t.Fatal("clientForAccount() client = nil")
	}

	newKey := tokenKey("cid", "user@example.com")
	stored, err := tokenFromKeyring(newKey, kr)
	if err != nil || stored.AccessToken != "legacy-token" {
		t.Errorf("token under new key = %v (err %v), want legacy-token", stored, err)
	}
	if _, err := kr.GetEntry(legacyKey); err == nil {
		t.Error("legacy entry still present after migration, want deleted")
	}
}

// TestGetTokenFromWebExitsOnContextTimeout verifies the wait for a callback
// that never arrives ends with an error instead of hanging forever.
func TestGetTokenFromWebExitsOnContextTimeout(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{
		ClientID: "cid",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://127.0.0.1:1/auth",
			TokenURL: "http://127.0.0.1:1/token",
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := getTokenFromWeb(ctx, cfg)
	if err == nil {
		t.Fatal("getTokenFromWeb() error = nil, want error when the callback never arrives")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("getTokenFromWeb() took %v, want it to exit promptly", elapsed)
	}
}

// TestStateValidation verifies the callback rejects a foreign state parameter
// and delivers the code only for a matching state.
func TestStateValidation(t *testing.T) {
	t.Parallel()
	codeChan := make(chan string, 1)
	addr, server, err := startCallbackServer("expected-state", codeChan)
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer func() { _ = server.Shutdown(context.Background()) }()

	badResp, err := http.Get("http://" + addr + "/callback?state=attacker&code=stolen")
	if err != nil {
		t.Fatalf("mismatched-state request error = %v", err)
	}
	if badResp.Body != nil {
		_ = badResp.Body.Close()
	}
	if badResp.StatusCode != http.StatusBadRequest {
		t.Errorf("mismatched-state status = %d, want %d", badResp.StatusCode, http.StatusBadRequest)
	}
	select {
	case code := <-codeChan:
		t.Errorf("callback delivered code %q despite state mismatch", code)
	default:
	}

	goodResp, err := http.Get("http://" + addr + "/callback?state=expected-state&code=good-code")
	if err != nil {
		t.Fatalf("matching-state request error = %v", err)
	}
	if goodResp.Body != nil {
		_ = goodResp.Body.Close()
	}
	if goodResp.StatusCode != http.StatusOK {
		t.Errorf("matching-state status = %d, want %d", goodResp.StatusCode, http.StatusOK)
	}

	select {
	case code := <-codeChan:
		if code != "good-code" {
			t.Errorf("delivered code = %q, want good-code", code)
		}
	case <-time.After(5 * time.Second):
		t.Error("no code delivered for a matching state")
	}
}
