package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

//go:embed Makefile.*.mk
var makefiles embed.FS

func main() {
	var name string
	flag.StringVar(&name, "name", "", "The Makefile to clone")
	flag.Parse()

	if name == "" {
		log.Fatalf("missing flag 'name'. Try running with go run github.com/alextanhongpin/core/cmd/make -name docker")
	}

	fileName := fmt.Sprintf("Makefile.%s.mk", name)
	data, err := makefiles.ReadFile(fileName)
	if err != nil {
		log.Fatalf("failed to read file: %s", err)
	}

	// Get the path relative to where the command is being run.
	path, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}

	// Save the file to the root directory of the caller.
	dest := filepath.Join(path, fileName)

	if err := os.WriteFile(dest, data, 0600); err != nil {
		log.Fatalf("failed to write file to %s: %v", dest, err)
	}
}
