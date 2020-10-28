package main

import (
	"log"
	"os/exec"

	"github.com/cloud-native-nordics/workshopctl/cmd/workshopctl/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	command := cmd.NewWorkshopCtlCommand()
	if err := doc.GenMarkdownTree(command, "./docs/cli"); err != nil {
		log.Fatal(err)
	}
	sedCmd := `sed -e "/Auto generated/d" -i docs/cli/*.md`
	if output, err := exec.Command("/bin/bash", "-c", sedCmd).CombinedOutput(); err != nil {
		log.Fatal(string(output), err)
	}
}
