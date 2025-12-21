package notification

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/gen2brain/beeep"
	"github.com/kumneger0/cligram/assets"
)

var (
	once sync.Once
)

func getAppLogo() *[]byte {
	var appLogo *[]byte
	logo, err := assets.Assets.ReadFile("logo.png")
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	appLogo = &logo
	return appLogo
}

func getAppIconPath() string {
	var iconPath string
	once.Do(func() {
		path := filepath.Join(os.TempDir(), "cligram-icon.png")
		logoPNG := getAppLogo()

		if logoPNG == nil {
			slog.Error("logo.png not found")
			return
		}

		if err := os.WriteFile(path, *logoPNG, 0o644); err != nil {
			slog.Error(err.Error())
			return
		}
		iconPath = path
	})
	return iconPath
}

func Notify(title string, message string) {
	beeep.AppName = "Cligram"
	logo := getAppIconPath()

	err := beeep.Notify(title, message, logo)
	if err != nil {
		slog.Error(err.Error())
	}
}
