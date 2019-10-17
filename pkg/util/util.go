package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/otiai10/copy"
	log "github.com/sirupsen/logrus"
)

func PathExists(path string) (bool, os.FileInfo) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}

	return true, info
}

func FileExists(filename string) bool {
	exists, info := PathExists(filename)
	if !exists {
		return false
	}

	return !info.IsDir()
}

// Copy copies both files and directories
func Copy(src string, dst string) error {
	log.Debugf("Copying %q to %q", src, dst)
	return copy.Copy(src, dst)
}

func ExecuteCommand(command string, args ...string) (string, error) {
	log.Debugf(`Executing "%s %s"`, command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf(`Returned error: %v and output: %q`, err, out)
		return "", fmt.Errorf("command %q exited with %q: %v", cmd.Args, out, err)
	}

	return string(bytes.TrimSpace(out)), nil
}
