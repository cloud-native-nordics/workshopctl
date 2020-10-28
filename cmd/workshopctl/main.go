package main

import (
	"os"

	"github.com/cloud-native-nordics/workshopctl/cmd/workshopctl/cmd"
)

func main() {
	if err := Run(); err != nil {
		os.Exit(1)
	}
}

// Run runs the main cobra command of this application
func Run() error {
	return cmd.NewWorkshopCtlCommand().Execute()
}
