package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra/doc"
	"github.com/luxas/workshopctl/cmd/workshopctl/cmd"
)

func main() {
	command := cmd.NewWorkshopCtlCommand(os.Stdin, os.Stdout, os.Stderr)
	if err := doc.GenMarkdownTree(command, "./docs/cli"); err != nil {
		log.Fatal(err)
	}
	sedCmd := `sed -e "/Auto generated/d" -i docs/cli/*.md`
	if output, err := exec.Command("/bin/bash", "-c", sedCmd).CombinedOutput(); err != nil {
		log.Fatal(string(output), err)
	}
}
