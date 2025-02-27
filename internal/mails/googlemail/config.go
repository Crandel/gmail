package googlemail

import (
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	libGmail "google.golang.org/api/gmail/v1"

	"github.com/Crandel/gmail/internal/file"
)

// SetConfig will write client credentials to config file.
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
	credentialsFile := fmt.Sprintf("%s/%s.json", cacheDir, clientID)
	_, err = os.Stat(credentialsFile)
	if err == nil {
		return fmt.Errorf("credential file %s already exists", credentialsFile)
	}
	return os.WriteFile(credentialsFile, b, 0644)
}

// GetConfig will return oauth2 config for clientID.
func GetConfig(clientID string) (*oauth2.Config, error) {
	cacheDir, err := file.GetCacheDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get cache dir: %w", err)
	}
	credentialsFile := fmt.Sprintf("%s/%s.json", cacheDir, clientID)

	slog.Debug("credentialsFile: " + credentialsFile)
	_, err = os.Stat(credentialsFile)
	if err != nil {
		_, err = os.Create(credentialsFile)
		if err != nil {
			return nil, fmt.Errorf("unable to create credential file: %w", err)
		}
	}
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
