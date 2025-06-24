package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

type CliGramConfig struct {
	Chat struct {
		SendTypingState *bool   `json:"sendTypingState,omitempty"`
		ReadReceiptMode *string `json:"readReceiptMode,omitempty"`
	} `json:"chat"`
	Privacy struct {
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

	return CliGramConfig{
		Chat: struct {
			SendTypingState *bool   `json:"sendTypingState,omitempty"`
			ReadReceiptMode *string `json:"readReceiptMode,omitempty"`
		}{
			SendTypingState: &sendTyping,
			ReadReceiptMode: &readReceipt,
		},
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

func GetConfig() CliGramConfig {
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

	json.Unmarshal(configFileContent, &config)

	return config
}
