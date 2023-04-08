package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

//go:embed Makefile.docker.mk
var docker []byte

func main() {

	var mk string
	flag.StringVar(&mk, "name", "", "The Makefile to clone")
	flag.Parse()

	if mk == "" {
		log.Fatalf("missing flag 'name'. Try running with go run github.com/alextanhongpin/go-core-microservice/cmd/make -name docker")
	}

	var data []byte
	switch mk {
	case "docker":
		data = docker
	default:
		log.Fatalf("%s not found", mk)
	}

	file := fmt.Sprintf("Makefile.%s.mk", mk)

	path, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}
	dest := filepath.Join(path, file)

	if err := os.WriteFile(dest, data, 0600); err != nil {
		log.Fatalf("failed to write file to %s: %v", dest, err)
	}
}
