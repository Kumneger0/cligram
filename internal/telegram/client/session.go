package client

import (
	"os"
	"path/filepath"

	"github.com/gotd/td/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type SessionManager struct {
	storage *FileSessionStorage
}

type FileSessionStorage struct {
	path string
}

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

func (sm *SessionManager) GetSessionStorage() types.SessionStorage {
	return sm.storage
}

func (sm *SessionManager) EnsureSessionDir() error {
	dir := filepath.Dir(sm.storage.path)
	return ensureSessionDir(dir)
}

func (fs *FileSessionStorage) Load() ([]byte, error) {
	return os.ReadFile(fs.path)
}

func (fs *FileSessionStorage) Store(data []byte) error {
	dir := filepath.Dir(fs.path)
	tmp, err := os.CreateTemp(dir, "session.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), fs.path)
}

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
