package main

import (
	"os"

	"github.com/joleques/northstar-ai/src/application"
)

func main() {
	app := application.NewApp(os.Stdout)
	code := app.Run(os.Args[1:])
	os.Exit(code)
}
