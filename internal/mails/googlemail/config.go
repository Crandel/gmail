package googlemail

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	libGmail "google.golang.org/api/gmail/v1"

	"github.com/Crandel/gmail/internal/file"
)

const credentialsFileMode = 0600

// SetConfig will write client credentials to config file. The JSON is
// validated first, and an existing credential file is never overwritten.
func SetConfig(clientID, filename string) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read file %s: %w", filename, err)
	}

	_, err = google.ConfigFromJSON(b, libGmail.GmailMetadataScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	cacheDir, err := file.GetCacheDir()
	if err != nil {
		return fmt.Errorf("unable to get cache dir: %w", err)
	}
	credentialsFile := filepath.Join(cacheDir, clientID+".json")
	if _, err := os.Stat(credentialsFile); err == nil {
		return fmt.Errorf("credential file %s already exists", credentialsFile)
	}
	return os.WriteFile(credentialsFile, b, credentialsFileMode)
}

// GetConfig will return oauth2 config for clientID. A missing credential file
// is an error; no file is created as a side effect.
func GetConfig(clientID string) (*oauth2.Config, error) {
	cacheDir, err := file.GetCacheDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get cache dir: %w", err)
	}
	credentialsFile := filepath.Join(cacheDir, clientID+".json")

	slog.Debug("credentialsFile: " + credentialsFile)
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %w", err)
	}

	config, err := google.ConfigFromJSON(b, libGmail.GmailMetadataScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}
	slog.Debug("config", slog.Any("conf", config))

	return config, nil
}
