package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDefaultCliGramConfig(t *testing.T) {
	cfg := defaultCliGramConfig()

	if cfg.Chat.ReadReceiptMode == nil || *cfg.Chat.ReadReceiptMode != "default" {
		t.Fatalf("expected ReadReceiptMode default, got %v", cfg.Chat.ReadReceiptMode)
	}
	if cfg.Chat.SendTypingState == nil || !*cfg.Chat.SendTypingState {
		t.Fatalf("expected SendTypingState true, got %v", cfg.Chat.SendTypingState)
	}
	if cfg.ReadStories {
		t.Fatalf("expected ReadStories false, got %v", cfg.ReadStories)
	}
}

func TestSaveAndGetConfig(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Reset global state for isolation
	configMu.Lock()
	loaded = false
	configMu.Unlock()

	// Initial GetConfig should return defaults
	cfg := GetConfig()
	if *cfg.Chat.ReadReceiptMode != "default" {
		t.Fatalf("expected default read receipt mode, got %s", *cfg.Chat.ReadReceiptMode)
	}

	// Modify and save
	neverMode := "never"
	noTyping := false
	cfg.Chat.ReadReceiptMode = &neverMode
	cfg.Chat.SendTypingState = &noTyping
	cfg.ReadStories = true

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was written
	configPath := filepath.Join(tempHome, ".cligram", "user.config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config file at %s", configPath)
	}

	// Verify in-memory GetConfig returns updated values
	updated := GetConfig()
	if *updated.Chat.ReadReceiptMode != "never" {
		t.Errorf("expected read receipt mode 'never', got %s", *updated.Chat.ReadReceiptMode)
	}
	if *updated.Chat.SendTypingState != false {
		t.Errorf("expected send typing false, got %v", *updated.Chat.SendTypingState)
	}
	if !updated.ReadStories {
		t.Errorf("expected read stories true, got %v", updated.ReadStories)
	}

	// Reset in-memory loaded state and verify readConfig reads from file
	configMu.Lock()
	loaded = false
	configMu.Unlock()

	reloaded := GetConfig()
	if *reloaded.Chat.ReadReceiptMode != "never" || !reloaded.ReadStories {
		t.Errorf("reloaded config mismatch: %+v", reloaded)
	}
}

func TestConfigCorruptedFileFallback(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configDir := filepath.Join(tempHome, ".cligram")
	_ = os.MkdirAll(configDir, 0700)
	_ = os.WriteFile(filepath.Join(configDir, "user.config.json"), []byte("{not valid json"), 0600)

	configMu.Lock()
	loaded = false
	configMu.Unlock()

	cfg := GetConfig()
	if *cfg.Chat.ReadReceiptMode != "default" {
		t.Errorf("expected default fallback for corrupted config, got %s", *cfg.Chat.ReadReceiptMode)
	}
}

func TestConcurrentConfigAccess(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configMu.Lock()
	loaded = false
	configMu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			_ = GetConfig()
		}(i)
		go func(idx int) {
			defer wg.Done()
			mode := "instant"
			c := GetConfig()
			c.Chat.ReadReceiptMode = &mode
			_ = SaveConfig(c)
		}(i)
	}
	wg.Wait()
}
