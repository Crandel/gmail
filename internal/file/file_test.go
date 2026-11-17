package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDirPermissions(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)

	got, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}
	want := filepath.Join(root, "mail")
	if got != want {
		t.Fatalf("GetCacheDir() = %q, want %q", got, want)
	}

	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("unable to stat cache dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("cache dir mode = %o, want 0700", perm)
	}
}

func TestCreateDirectoryTightensExisting(t *testing.T) {
	t.Parallel()
	existing := filepath.Join(t.TempDir(), "loose")
	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatalf("unable to create directory: %v", err)
	}
	if err := CreateDirectory(existing); err != nil {
		t.Fatalf("CreateDirectory() error = %v", err)
	}

	info, err := os.Stat(existing)
	if err != nil {
		t.Fatalf("unable to stat directory: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("existing directory mode = %o, want 0700", perm)
	}
}
