package client

import (
	"os"
	"path/filepath"

	"github.com/gotd/td/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

// SessionManager implements the SessionManager interface
type SessionManager struct {
	storage *FileSessionStorage
}

// FileSessionStorage implements the SessionStorage interface
type FileSessionStorage struct {
	path string
}

// NewSessionManager creates a new session manager
func NewSessionManager() (types.SessionManager, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sessionStoragePath := filepath.Join(userHomeDir, ".cligram")
	if err := ensureSessionDir(sessionStoragePath); err != nil {
		return nil, err
	}

	storage := &FileSessionStorage{
		path: filepath.Join(sessionStoragePath, "session.json"),
	}

	return &SessionManager{
		storage: storage,
	}, nil
}

// GetSessionStorage returns the session storage
func (sm *SessionManager) GetSessionStorage() types.SessionStorage {
	return sm.storage
}

// EnsureSessionDir ensures the session directory exists
func (sm *SessionManager) EnsureSessionDir() error {
	dir := filepath.Dir(sm.storage.path)
	return ensureSessionDir(dir)
}

// Load loads session data from file
func (fs *FileSessionStorage) Load() ([]byte, error) {
	return os.ReadFile(fs.path)
}

// Store saves session data to file
func (fs *FileSessionStorage) Store(data []byte) error {
	return os.WriteFile(fs.path, data, 0o600)
}

// GetTelegramFileSessionStorage returns a telegram.FileSessionStorage for compatibility
func (sm *SessionManager) GetTelegramFileSessionStorage() *telegram.FileSessionStorage {
	return &telegram.FileSessionStorage{
		Path: sm.storage.path,
	}
}

func ensureSessionDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o700)
		}
		return err
	}
	return nil
}
