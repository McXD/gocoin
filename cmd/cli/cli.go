package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"
)

func ping(cCtx *cli.Context) error {
	_, err := http.Get("http://localhost:8080/ping")

	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	} else {
		fmt.Println("Ping succeeded!")
		return nil
	}
}

func main() {
	app := &cli.App{
		Action: ping,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
