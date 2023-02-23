package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config, err := loadBuiltinConfig()
	if err != nil {
		return err
	}

	path := os.Args[1]
	if path == "" {
		return fmt.Errorf("path is required")
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	workflow, err := parse(f)
	if err != nil {
		return err
	}

	linter := newLinter(config, workflow)
	if err := linter.Lint(); err != nil {
		return err
	}

	succeeded := linter.Conclude()
	if !succeeded {
		os.Exit(1)
	}

	return nil
}
