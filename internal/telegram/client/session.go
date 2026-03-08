package client

import (
	"os"
	"path/filepath"

	"github.com/gotd/td/telegram"
)

func newFileSessionStorage() (*telegram.FileSessionStorage, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sessionDir := filepath.Join(userHomeDir, ".cligram")
	if err := ensureDir(sessionDir); err != nil {
		return nil, err
	}

	return &telegram.FileSessionStorage{
		Path: filepath.Join(sessionDir, "session.json"),
	}, nil
}

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o700)
	} else if err != nil {
		return err
	}
	return nil
}
