package googlemail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/Crandel/gmail/internal/file"
	"github.com/Crandel/gmail/internal/keyring"
)

const (
	tokKey   = "gmailTokenKey"
	endpoint = "callback"

	authWaitTimeout = 5 * time.Minute
	authStateBytes  = 16
)

type keyringHandler interface {
	GetEntry(key string) (string, error)
	SetEntry(key string, data string) error
	DeleteEntry(key string) error
}

// authorizeFunc runs the interactive browser authorization and returns a fresh token.
type authorizeFunc func(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error)

// tokenKey builds the per-account keyring key. One OAuth client can serve
// several mailboxes, so the client ID alone would make accounts overwrite
// each other's tokens.
func tokenKey(clientID, email string) string {
	return fmt.Sprintf("%s:%s:%s", tokKey, clientID, email)
}

// legacyTokenKey is the pre-migration key format that only used the client ID.
func legacyTokenKey(clientID string) string {
	return tokKey + clientID
}

// GetClient returns an authorized HTTP client for email, authorizing in the
// browser when no usable stored token exists.
func GetClient(ctx context.Context, config *oauth2.Config, email string, systemKeyring bool) (*http.Client, error) {
	var keyringH keyringHandler
	if systemKeyring {
		keyringH = keyring.NewKeyring(tokKey)
	} else {
		keyringDir, err := file.GetCacheDir()
		if err != nil {
			return nil, err
		}
		keyringH, err = keyring.NewFileKeyring(keyringDir, tokKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create keyring handler: %w", err)
		}
	}
	return clientForAccount(ctx, config, email, keyringH, getTokenFromWeb)
}

// clientForAccount reads the stored token, refreshes it when expired, and
// falls back to browser authorization when no token exists or refresh fails.
// Every resulting token is upserted under the per-account key.
func clientForAccount(
	ctx context.Context,
	config *oauth2.Config,
	email string,
	keyringH keyringHandler,
	authorize authorizeFunc,
) (*http.Client, error) {
	key := tokenKey(config.ClientID, email)
	token, err := lookupToken(key, legacyTokenKey(config.ClientID), keyringH)
	if err != nil {
		slog.Debug("no usable stored token, starting authorization", slog.Any("error", err))
		token, err = authorize(ctx, config)
		if err != nil {
			slog.Debug("can't get token from web", slog.Any("error", err))
			return nil, err
		}
		if err := saveToken(key, token, keyringH); err != nil {
			slog.Debug("can't save token to keyring", slog.Any("error", err))
			return nil, err
		}
		return config.Client(ctx, token), nil
	}

	if !token.Valid() {
		newToken, rerr := config.TokenSource(ctx, token).Token()
		if rerr != nil {
			slog.Debug("can't refresh token from web, starting authorization", slog.Any("error", rerr))
			token, err = authorize(ctx, config)
			if err != nil {
				slog.Debug("can't get token from web", slog.Any("error", err))
				return nil, err
			}
		} else {
			token = newToken
		}
		if err := saveToken(key, token, keyringH); err != nil {
			slog.Debug("can't save refreshed token to keyring", slog.Any("error", err))
			return nil, err
		}
	}
	return config.Client(ctx, token), nil
}

// lookupToken reads the token under key and migrates a legacy client-ID-only
// entry to the per-account key, deleting the legacy entry on success.
func lookupToken(key, legacyKey string, keyringH keyringHandler) (*oauth2.Token, error) {
	token, err := tokenFromKeyring(key, keyringH)
	if err == nil {
		return token, nil
	}
	legacy, legacyErr := tokenFromKeyring(legacyKey, keyringH)
	if legacyErr != nil {
		return nil, err
	}
	if saveErr := saveToken(key, legacy, keyringH); saveErr != nil {
		slog.Error("unable to migrate legacy token entry", slog.Any("error", saveErr))
		return nil, err
	}
	if delErr := keyringH.DeleteEntry(legacyKey); delErr != nil {
		slog.Debug("unable to delete legacy token entry", slog.Any("error", delErr))
	}
	slog.Debug("migrated legacy token entry to the per-account key")
	return legacy, nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	state, err := randomState()
	if err != nil {
		return nil, err
	}
	codeChan := make(chan string, 1)
	addr, server, err := startCallbackServer(state, codeChan)
	if err != nil {
		return nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, authWaitTimeout)
	defer cancel()
	defer func() {
		if err := server.Shutdown(context.Background()); err != nil {
			slog.Debug("HTTP server shutdown", slog.Any("error", err))
		}
	}()

	config.RedirectURL = fmt.Sprintf("http://%s/%s", addr, endpoint)
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	select {
	case authCode = <-codeChan:
	case <-waitCtx.Done():
		return nil, fmt.Errorf("timed out waiting for the authorization callback: %w", waitCtx.Err())
	}

	tok, err := config.Exchange(waitCtx, authCode)
	if err != nil {
		slog.Debug("Unable to retrieve token from web", slog.Any("error", err))
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// Retrieves a token from the keyring. Expired tokens are returned as-is so
// the caller can refresh them instead of re-authorizing.
func tokenFromKeyring(key string, keyringH keyringHandler) (*oauth2.Token, error) {
	userString, err := keyringH.GetEntry(key)
	if err != nil {
		return nil, fmt.Errorf("unable to read token from keyring: %w", err)
	}
	token := &oauth2.Token{}
	if err := json.Unmarshal([]byte(userString), token); err != nil {
		return nil, fmt.Errorf("unable to unmarshal token: %w", err)
	}
	return token, nil
}

// Saves a token to a file path, replacing any entry stored under the same key.
func saveToken(key string, token *oauth2.Token, keyring keyringHandler) error {
	slog.Debug("Saving credentials  to keyring with expiry date: " + token.Expiry.String())
	tokenByte, err := json.Marshal(token)
	if err != nil {
		slog.Debug("Error during marshaling token", slog.Any("error", err))
		return err
	}
	return keyring.SetEntry(key, string(tokenByte))
}

// randomState returns an opaque value used to tie the callback to this run.
func randomState() (string, error) {
	buf := make([]byte, authStateBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("unable to generate authorization state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// startCallbackServer serves the OAuth callback from a dedicated mux on an
// ephemeral loopback port, so several accounts can authorize in parallel.
// Requests whose state parameter does not match are rejected.
func startCallbackServer(state string, codeChan chan<- string) (string, *http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/"+endpoint, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		select {
		case codeChan <- query.Get("code"):
		default:
			slog.Debug("ignoring duplicate authorization callback")
		}
		fmt.Fprint(w, "Authorization successful! You can close this window now.") //nolint: errcheck
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("unable to listen for the authorization callback: %w", err)
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server Serve", slog.Any("error", err))
		}
	}()
	return listener.Addr().String(), server, nil
}
