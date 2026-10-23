package main

import (
	"log"

	"github.com/musiermoore/oksana-vpn-api/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
