package googlemail

import (
	"os"
	"path/filepath"
	"testing"
)

const validClientSecret = `{"web":{"client_id":"cid1","client_secret":"secret",` +
	`"redirect_uris":["http://localhost"],"auth_uri":"https://accounts.google.com/o/oauth2/auth",` +
	`"token_uri":"https://oauth2.googleapis.com/token"}}`

func TestGetConfigMissingFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)

	if _, err := GetConfig("absent-client"); err == nil {
		t.Fatal("GetConfig() error = nil, want error for missing credential file")
	}

	cacheDir, err := os.ReadDir(filepath.Join(root, "mail"))
	if err != nil {
		t.Fatalf("unable to read cache dir: %v", err)
	}
	for _, e := range cacheDir {
		if e.Name() == "absent-client.json" {
			t.Error("GetConfig() created an empty credential file, want none")
		}
	}
}

func TestSetConfigPermissions(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)

	secretPath := filepath.Join(t.TempDir(), "client_secret.json")
	if err := os.WriteFile(secretPath, []byte(validClientSecret), 0600); err != nil {
		t.Fatalf("unable to write client secret: %v", err)
	}

	if err := SetConfig("cid1", secretPath); err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}

	credentialsFile := filepath.Join(root, "mail", "cid1.json")
	info, err := os.Stat(credentialsFile)
	if err != nil {
		t.Fatalf("credential file was not written: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("credential file mode = %o, want 0600", perm)
	}
}

func TestSetConfigRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)

	secretPath := filepath.Join(t.TempDir(), "client_secret.json")
	if err := os.WriteFile(secretPath, []byte(validClientSecret), 0600); err != nil {
		t.Fatalf("unable to write client secret: %v", err)
	}

	cacheDir := filepath.Join(root, "mail")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatalf("unable to create cache dir: %v", err)
	}
	credentialsFile := filepath.Join(cacheDir, "cid1.json")
	original := []byte("existing content")
	if err := os.WriteFile(credentialsFile, original, 0600); err != nil {
		t.Fatalf("unable to seed credential file: %v", err)
	}

	if err := SetConfig("cid1", secretPath); err == nil {
		t.Fatal("SetConfig() error = nil, want error when credential file exists")
	}

	got, err := os.ReadFile(credentialsFile)
	if err != nil {
		t.Fatalf("unable to read credential file: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("credential file was overwritten: got %q, want %q", got, original)
	}
}

func TestSetConfigInvalidJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)

	secretPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(secretPath, []byte(`{"not":"credentials"}`), 0600); err != nil {
		t.Fatalf("unable to write invalid secret: %v", err)
	}

	if err := SetConfig("bad", secretPath); err == nil {
		t.Fatal("SetConfig() error = nil, want error for invalid client JSON")
	}
	if _, err := os.Stat(filepath.Join(root, "mail", "bad.json")); err == nil {
		t.Error("SetConfig() wrote a credential file for invalid JSON, want none")
	}
}
