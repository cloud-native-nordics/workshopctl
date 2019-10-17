package main

import (
	"os"

	"github.com/luxas/workshopctl/cmd/workshopctl/cmd"
)

func main() {
	if err := Run(); err != nil {
		os.Exit(1)
	}
}

// Run runs the main cobra command of this application
func Run() error {
	c := cmd.NewWorkshopCtlCommand(os.Stdin, os.Stdout, os.Stderr)
	return c.Execute()
}
