package googlemail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/Crandel/gmail/internal/file"
	"github.com/Crandel/gmail/internal/keyring"
)

const (
	tokKey       = "gmailTokenKey"
	redirectHost = "http://localhost"
	redirectPort = ":8080"
	endpoint     = "callback"
)

type keyringHandler interface {
	GetEntry(key string) (string, error)
	SetEntry(key string, data string) error
}

// Retrieve a token, saves the token, then returns the generated client.
func GetClient(ctx context.Context, config *oauth2.Config, systemKeyring bool) (*http.Client, error) {
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
	key := tokKey + config.ClientID
	token, err := tokenFromKeyring(key, keyringH)
	slog.Debug("token from keyring", slog.Any("token", token))
	if err != nil {
		token, err = getTokenFromWeb(ctx, config)
		if err != nil || token == nil {
			slog.Debug("can't get token from web", slog.Any("error", err), slog.Any("token", token))
			return nil, err
		}
		err = saveToken(key, token, keyringH)
		if err != nil {
			slog.Debug("can't save token to keyring", slog.Any("error", err))
			return nil, err
		}
	}

	if !token.Valid() {
		newToken, err := config.TokenSource(ctx, token).Token()
		if err != nil {
			slog.Debug("can't refresh token from web", slog.Any("error", err))
			return nil, err
		}
		err = saveToken(key, newToken, keyringH)
		if err != nil {
			slog.Debug("can't save refreshed token to keyring", slog.Any("error", err))
			return nil, err
		}
		token = newToken
	}
	return config.Client(ctx, token), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = fmt.Sprintf("%s%s/%s", redirectHost, redirectPort, endpoint)
	codeChan := startServer(redirectPort, endpoint)
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	authCode := <-codeChan
	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		slog.Debug("Unable to retrieve token from web", slog.Any("error", err))
		return &oauth2.Token{}, err
	}
	return tok, nil
}

// Retrieves a token from a local file.
func tokenFromKeyring(key string, keyringH keyringHandler) (*oauth2.Token, error) {
	token := &oauth2.Token{}
	userString, err := keyringH.GetEntry(key)
	if err == nil {
		err = json.Unmarshal([]byte(userString), token)
		if err == nil && token.Valid() {
			return token, nil
		}
		return nil, errors.New("invalid token")
	}

	return token, err
}

// Saves a token to a file path.
func saveToken(key string, token *oauth2.Token, keyring keyringHandler) error {
	slog.Debug("Saving credentials  to keyring with expiry date: " + token.Expiry.String())
	tokenByte, err := json.Marshal(token)
	if err != nil {
		slog.Debug("Error during marshaling token", slog.Any("error", err))
		return err
	}
	return keyring.SetEntry(key, string(tokenByte))
}

func startServer(port, endpoint string) chan string {
	codeChan := make(chan string)
	server := &http.Server{Addr: port}

	http.HandleFunc("/"+endpoint, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		codeChan <- code
		fmt.Fprintf(w, "Authorization successful! You can close this window now.") //nolint: errcheck
		go func() {
			_ = server.Shutdown(context.Background())
		}()
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("HTTP server ListenAndServe", slog.Any("error", err))
			return
		}
	}()

	return codeChan
}
