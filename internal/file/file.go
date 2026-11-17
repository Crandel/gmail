package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Crandel/gmail/internal/env"
)

const (
	dir = "mail"

	dirMode = 0700
)

// GetCacheDir return path to mail cache dir. It resolves XDG_CACHE_HOME at call
// time and falls back to the user's home cache directory, so a literal "~"
// path is never created.
func GetCacheDir() (string, error) {
	base := env.GetEnv("XDG_CACHE_HOME", "")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".cache")
	}
	mailCacheDir := filepath.Join(base, dir)
	if err := CreateDirectory(mailCacheDir); err != nil {
		return "", err
	}
	return mailCacheDir, nil
}

// CreateDirectory will create directory if not exists. Existing directories
// are tightened to owner-only permissions because they hold OAuth secrets.
func CreateDirectory(directory string) error {
	if _, err := os.Stat(directory); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(directory, dirMode); err != nil {
			return err
		}
	}
	return os.Chmod(directory, dirMode)
}
