package telegram

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteAccount(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create test account directory
	accountDir := filepath.Join(tempHome, ".cligram", "test-account")
	if err := os.MkdirAll(accountDir, 0700); err != nil {
		t.Fatalf("failed to create test account dir: %v", err)
	}
	sessionFile := filepath.Join(accountDir, "session.json")
	if err := os.WriteFile(sessionFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create session file: %v", err)
	}

	// 1. Test empty account name
	if err := DeleteAccount(""); err == nil {
		t.Errorf("expected error for empty account name, got nil")
	}

	// 2. Test directory traversal
	if err := DeleteAccount("../evil"); err == nil {
		t.Errorf("expected error for path traversal account name, got nil")
	}
	if err := DeleteAccount("../../etc"); err == nil {
		t.Errorf("expected error for path traversal account name, got nil")
	}

	// 3. Test deleting valid account
	if err := DeleteAccount("test-account"); err != nil {
		t.Errorf("unexpected error deleting valid account: %v", err)
	}
	if _, err := os.Stat(accountDir); !os.IsNotExist(err) {
		t.Errorf("expected account directory to be removed, but it still exists")
	}
}
