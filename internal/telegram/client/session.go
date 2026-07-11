package client

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotd/td/telegram"
)

func newFileSessionStorage(account string) (*telegram.FileSessionStorage, error) {
	if account == "" {
		return nil, errors.New("account cannot be empty")
	}
	if account == "." || account == ".." ||
		strings.Contains(account, string(filepath.Separator)) {
		return nil, errors.New("invalid account name")
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sessionDir := filepath.Join(userHomeDir, ".cligram", account)
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
