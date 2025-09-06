package notification

import (
	"log/slog"
	"sync"

	"github.com/gen2brain/beeep"
	"github.com/kumneger0/cligram/assets"
)

var (
	once sync.Once
)

func getAppLogo() *[]byte {
	var appLogo *[]byte
	once.Do(func() {
		logo, err := assets.Assets.ReadFile("logo.png")
		if err != nil {
			slog.Error(err.Error())
			return
		}
		appLogo = &logo
	})
	return appLogo
}

func Notify(title string, message string) {
	beeep.AppName = "Cligram"
	logo := getAppLogo

	err := beeep.Alert(title, message, logo)
	if err != nil {
		slog.Error(err.Error())
	}
}
