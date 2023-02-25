package main

import (
	"log"
	"os"

	"github.com/itkq/linta/internal/linta"
)

func main() {
	app := linta.CreateApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
