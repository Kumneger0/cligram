package notification

import (
	"log/slog"

	"github.com/gen2brain/beeep"
	"github.com/kumneger0/cligram/assets"
)

func Notify(title string, message string) {
	beeep.AppName = "Cligram"
	logo, err := assets.Assets.ReadFile("logo.png")

	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}

	err = beeep.Alert(title, message, logo)
	if err != nil {
		panic(err)
	}
}
