package main

import (
	"brightpod/cmd"
	"os"
)

func main() {

	cmd := cmd.NewRootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}

	os.Exit(1)

}
