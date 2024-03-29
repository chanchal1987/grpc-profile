package main

import (
	"fmt"
	"os"

	"github.com/chanchal1987/grpc-profile/cmd/grpc-profile/cmd"
)

var (
	// Version number
	Version = "-"
	// Build string
	Build = "-"
)

func main() {
	if err := cmd.Execute(Version, Build); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
