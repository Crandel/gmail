package keyring

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func newTestFileKeyring(t *testing.T) Handler {
	t.Helper()
	h, err := NewFileKeyring(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("NewFileKeyring() error = %v", err)
	}
	return h
}

func TestSetEntryUpsert(t *testing.T) {
	t.Parallel()
	h := newTestFileKeyring(t)

	if err := h.SetEntry("key", "first"); err != nil {
		t.Fatalf("SetEntry() first write error = %v", err)
	}
	if err := h.SetEntry("key", "second"); err != nil {
		t.Fatalf("SetEntry() upsert error = %v, want nil", err)
	}
	got, err := h.GetEntry("key")
	if err != nil {
		t.Fatalf("GetEntry() error = %v", err)
	}
	if got != "second" {
		t.Errorf("GetEntry() = %q, want %q", got, "second")
	}
}

func TestGetEntryMissing(t *testing.T) {
	t.Parallel()
	h := newTestFileKeyring(t)

	got, err := h.GetEntry("absent")
	if err == nil {
		t.Fatal("GetEntry() error = nil, want non-nil error for missing key")
	}
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("GetEntry() error = %v, want wrapped ErrKeyNotFound", err)
	}
	if got != "" {
		t.Errorf("GetEntry() value = %q, want empty", got)
	}
}

func TestDeleteEntry(t *testing.T) {
	t.Parallel()
	h := newTestFileKeyring(t)

	if err := h.SetEntry("key", "value"); err != nil {
		t.Fatalf("SetEntry() error = %v", err)
	}
	if err := h.DeleteEntry("key"); err != nil {
		t.Fatalf("DeleteEntry() error = %v", err)
	}
	if _, err := h.GetEntry("key"); !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("GetEntry() after delete error = %v, want ErrKeyNotFound", err)
	}
	if err := h.DeleteEntry("key"); err != nil {
		t.Errorf("DeleteEntry() of absent key error = %v, want nil", err)
	}
}

// TestConcurrentSetEntry needs -race to be meaningful.
func TestConcurrentSetEntry(t *testing.T) { //nolint:paralleltest // exercises concurrency itself.
	dir := t.TempDir()
	h1, err := NewFileKeyring(dir, "test")
	if err != nil {
		t.Fatalf("NewFileKeyring() error = %v", err)
	}
	h2, err := NewFileKeyring(dir, "test")
	if err != nil {
		t.Fatalf("NewFileKeyring() error = %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := h1.SetEntry("a", "1"); err != nil {
			t.Errorf("SetEntry(a) error = %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := h2.SetEntry("b", "2"); err != nil {
			t.Errorf("SetEntry(b) error = %v", err)
		}
	}()
	wg.Wait()

	for key, want := range map[string]string{"a": "1", "b": "2"} {
		got, err := h1.GetEntry(key)
		if err != nil {
			t.Errorf("GetEntry(%s) error = %v, want both entries kept", key, err)
			continue
		}
		if got != want {
			t.Errorf("GetEntry(%s) = %q, want %q", key, got, want)
		}
	}
}

func TestKeyringFilePermissions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatalf("unable to chmod cache dir: %v", err)
	}

	h, err := NewFileKeyring(dir, "perm")
	if err != nil {
		t.Fatalf("NewFileKeyring() error = %v", err)
	}
	if err := h.SetEntry("key", "value"); err != nil {
		t.Fatalf("SetEntry() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "perm_keyring.json"))
	if err != nil {
		t.Fatalf("keyring file was not written: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("keyring file mode = %o, want 0600", perm)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("unable to stat cache dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("cache dir mode = %o, want 0700", perm)
	}
}
