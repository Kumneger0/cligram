package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type CliGramConfig struct {
	Chat struct {
		SendTypingState *bool   `json:"sendTypingState,omitempty"`
		ReadReceiptMode *string `json:"readReceiptMode,omitempty"`
	} `json:"chat"`
	ReadStories bool `json:"readStories"`
	Privacy     struct {
		LastSeenVisibility *string `json:"lastSeenVisibility,omitempty"`
	} `json:"privacy"`
	Notifications struct {
		Enabled            *bool `json:"enabled,omitempty"`
		ShowMessagePreview *bool `json:"showMessagePreview,omitempty"`
	} `json:"notifications"`
}

func defaultCliGramConfig() CliGramConfig {
	sendTyping := true
	readReceipt := "default"
	enabled := true
	showPreview := true
	readStories := false

	return CliGramConfig{
		Chat: struct {
			SendTypingState *bool   `json:"sendTypingState,omitempty"`
			ReadReceiptMode *string `json:"readReceiptMode,omitempty"`
		}{
			SendTypingState: &sendTyping,
			ReadReceiptMode: &readReceipt,
		},
		ReadStories: readStories,
		Privacy: struct {
			LastSeenVisibility *string `json:"lastSeenVisibility,omitempty"`
		}{
			LastSeenVisibility: nil,
		},
		Notifications: struct {
			Enabled            *bool `json:"enabled,omitempty"`
			ShowMessagePreview *bool `json:"showMessagePreview,omitempty"`
		}{
			Enabled:            &enabled,
			ShowMessagePreview: &showPreview,
		},
	}
}

// read the config only once, mostly it won't change that often so we can cache it
var config CliGramConfig
var configOnce sync.Once

func GetConfig() CliGramConfig {
	configOnce.Do(func() {
		config = readConfig()
	})
	return config
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

	var config CliGramConfig

	configFileContent, err := io.ReadAll(file)
	if err != nil {
		return defaultCliGramConfig()
	}

	if err := json.Unmarshal(configFileContent, &config); err != nil {
		return defaultCliGramConfig()
	}

	return config
}
