package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const fileName = "keyring.json"

// ErrKeyNotFound is returned when the requested key has no entry in the keyring.
var ErrKeyNotFound = errors.New("key not found in keyring")

type content map[string]string

type fileKeyring struct {
	mu   sync.Mutex
	file string
}

var (
	registryMu sync.Mutex
	registry   = map[string]*fileKeyring{}
)

// NewFileKeyring returns a Handler bound to dir/name_keyring.json. Handlers are
// shared per absolute file path so goroutines for different accounts serialize
// on the same mutex instead of racing on read-modify-write.
func NewFileKeyring(dir, name string) (Handler, error) {
	filename := filepath.Join(dir, name+"_"+fileName)
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve keyring path: %w", err)
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	fk, ok := registry[abs]
	if !ok {
		fk = &fileKeyring{file: abs}
		registry[abs] = fk
	}
	return fk, nil
}

// getMap decodes the keyring file. A missing or empty file yields an empty map.
// Callers must hold fk.mu.
func (fk *fileKeyring) getMap() (content, error) {
	c := make(content)
	origFile, err := os.Open(fk.file)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		slog.Error("error during opening file", slog.Any("error", err))
		return nil, fmt.Errorf("error during opening file: %w", err)
	}
	defer func() {
		if cerr := origFile.Close(); cerr != nil {
			slog.Error("error during closing file", slog.Any("error", cerr))
		}
	}()

	if err := json.NewDecoder(origFile).Decode(&c); err != nil {
		if !errors.Is(err, io.EOF) {
			slog.Error("error during Unmarshal", slog.Any("error", err))
			return nil, fmt.Errorf("error during Unmarshal: %w", err)
		}
	}
	return c, nil
}

// GetEntry returns the value stored for key, or an ErrKeyNotFound-wrapped
// error and an empty value when the key is absent.
func (fk *fileKeyring) GetEntry(key string) (string, error) {
	fk.mu.Lock()
	defer fk.mu.Unlock()

	con, err := fk.getMap()
	if err != nil {
		return "", err
	}
	entry, ok := con[key]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return entry, nil
}

// SetEntry stores data under key, replacing any existing entry.
func (fk *fileKeyring) SetEntry(key string, data string) error {
	fk.mu.Lock()
	defer fk.mu.Unlock()

	con, err := fk.getMap()
	if err != nil {
		return err
	}
	con[key] = data
	slog.Debug("con", slog.Any("map", con))

	newJSON, err := json.MarshalIndent(con, "", "  ")
	if err != nil {
		return fmt.Errorf("error during marshaling: %w", err)
	}
	return fk.write(newJSON)
}

// DeleteEntry removes key from the keyring. Deleting an absent key succeeds.
func (fk *fileKeyring) DeleteEntry(key string) error {
	fk.mu.Lock()
	defer fk.mu.Unlock()

	con, err := fk.getMap()
	if err != nil {
		return err
	}
	if _, ok := con[key]; !ok {
		return nil
	}
	delete(con, key)

	newJSON, err := json.MarshalIndent(con, "", "  ")
	if err != nil {
		return fmt.Errorf("error during marshaling: %w", err)
	}
	return fk.write(newJSON)
}

// write replaces the keyring file atomically. The temporary file keeps the
// owner-only mode 0600 that os.CreateTemp applies, so the renamed file never
// gets loose permissions.
func (fk *fileKeyring) write(newJSON []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(fk.file), ".keyring-*.tmp")
	if err != nil {
		slog.Error("error during creating temp file", slog.Any("error", err))
		return fmt.Errorf("error during creating temp file: %w", err)
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()

	if _, err := tmp.Write(newJSON); err != nil {
		_ = tmp.Close()
		slog.Error("error during writing string", slog.Any("error", err))
		return fmt.Errorf("error during writing string: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("error during closing temp file: %w", err)
	}
	if err := os.Rename(name, fk.file); err != nil {
		return fmt.Errorf("error during replacing keyring file: %w", err)
	}
	return nil
}
