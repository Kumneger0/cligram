package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kumneger0/cligram/internal/assets"
)

func GetJSExcutable() (*string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user cache directory: %w", err)
	}

	appDir := filepath.Join(cacheDir, "cligram")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create app cache directory: %w", err)
	}

	backendPath := filepath.Join(appDir, "cligram-js-backend")

	embeddedHash := sha256.Sum256(assets.JSBackendBinary)
	embeddedHashStr := hex.EncodeToString(embeddedHash[:])

	fileOnDisk, err := os.Open(backendPath)
	if err == nil {
		hasher := sha256.New()
		if _, err := io.Copy(hasher, fileOnDisk); err == nil {
			diskHashStr := hex.EncodeToString(hasher.Sum(nil))
			if diskHashStr == embeddedHashStr {
				fileOnDisk.Close()
				return &backendPath, nil
			}
		}
		fileOnDisk.Close()
	}

	if err := os.WriteFile(backendPath, assets.JSBackendBinary, 0755); err != nil {
		return nil, fmt.Errorf("could not write embedded backend binary: %w", err)
	}
	return &backendPath, nil
}
