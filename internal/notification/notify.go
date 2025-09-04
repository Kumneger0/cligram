package notification

import (
	"fmt"

	"github.com/gen2brain/beeep"
	"github.com/kumneger0/cligram/assets"
)

func Notify(title string, message string) {
	beeep.AppName = "Cligram"
	logo, err := assets.Assets.ReadFile("logo.png")

	if err != nil {
		fmt.Println("err")
	}

	err = beeep.Alert(title, message, logo)
	if err != nil {
		panic(err)
	}
}
