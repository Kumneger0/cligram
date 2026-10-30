package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type ChatConfig struct {
	SendTypingState *bool   `json:"sendTypingState,omitempty"`
	ReadReceiptMode *string `json:"readReceiptMode,omitempty"`
}

type CliGramConfig struct {
	Chat        ChatConfig `json:"chat"`
	ReadStories bool       `json:"readStories"`
}

func defaultCliGramConfig() CliGramConfig {
	sendTyping := true
	readReceipt := "default"
	readStories := false

	return CliGramConfig{
		Chat: ChatConfig{
			SendTypingState: &sendTyping,
			ReadReceiptMode: &readReceipt,
		},
		ReadStories: readStories,
	}
}

var (
	configMu sync.RWMutex
	config   CliGramConfig
	loaded   bool
)

func GetConfig() CliGramConfig {
	configMu.RLock()
	if loaded {
		cfg := config
		configMu.RUnlock()
		return cfg
	}
	configMu.RUnlock()

	configMu.Lock()
	defer configMu.Unlock()
	if !loaded {
		config = readConfig()
		loaded = true
	}
	return config
}

func SaveConfig(cfg CliGramConfig) error {
	configMu.Lock()
	defer configMu.Unlock()

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		config = cfg
		loaded = true
		return err
	}

	configDir := filepath.Join(userHomeDir, ".cligram")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		config = cfg
		loaded = true
		return err
	}

	configPath := filepath.Join(configDir, "user.config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return err
	}

	config = cfg
	loaded = true
	return nil
}

func readConfig() CliGramConfig {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return defaultCliGramConfig()
	}
	configPath := filepath.Join(userHomeDir, ".cligram", "user.config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return defaultCliGramConfig()
	}

	file, err := os.Open(configPath)
	if err != nil {
		return defaultCliGramConfig()
	}
	defer file.Close()

	configFileContent, err := io.ReadAll(file)
	if err != nil {
		return defaultCliGramConfig()
	}

	cfg := defaultCliGramConfig()
	if err := json.Unmarshal(configFileContent, &cfg); err != nil {
		return defaultCliGramConfig()
	}

	return cfg
}
