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

	// NotifyFn and AlertFn allow overriding for tests or custom dispatchers.
	NotifyFn = defaultNotify
	AlertFn  = defaultAlert
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
	path := filepath.Join(os.TempDir(), "cligram-icon.png")
	once.Do(func() {
		logoPNG := getAppLogo()

		if logoPNG == nil {
			slog.Error("logo.png not found")
			return
		}

		if err := os.WriteFile(path, *logoPNG, 0o644); err != nil {
			slog.Error(err.Error())
			return
		}
	})
	return path
}

func emitDesktop(notifyFunc func(title, message string, appIcon any) error, title, message string) {
	beeep.AppName = "Cligram"
	logo := getAppIconPath()

	if err := notifyFunc(title, message, logo); err != nil {
		slog.Error("failed to emit desktop notification", "error", err)
	}
}

func defaultNotify(title string, message string) {
	emitDesktop(beeep.Notify, title, message)
}

func defaultAlert(title string, message string) {
	emitDesktop(beeep.Alert, title, message)
}

func Notify(title string, message string) {
	NotifyFn(title, message)
}

func Alert(title string, message string) {
	AlertFn(title, message)
}
