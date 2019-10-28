package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/otiai10/copy"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
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
	outbytes, err := cmd.CombinedOutput()
	out := string(bytes.TrimSpace(outbytes))
	if err != nil {
		log.Debugf(`Returned error: %v and output: %q`, err, out)
		return out, fmt.Errorf("command %q exited with %q: %v", cmd.Args, out, err)
	}

	return out, nil
}

func Poll(d *time.Duration, logger *log.Entry, fn wait.ConditionFunc) error {
	duration := 10 * time.Second
	if d != nil {
		duration = *d
	}
	if logger == nil {
		logger = log.NewEntry(log.StandardLogger())
	}
	tryCount := 0
	return wait.PollImmediateInfinite(duration, func() (done bool, err error) {
		tryCount++
		errFn := logger.Debugf
		if tryCount%3 == 0 { // print info every third time
			errFn = logger.Infof
		}

		done, err = fn()
		if err != nil {
			errFn("an error occurred: %v. done: %t", err, done)
			// if we're not "done" yet, set the err to nil so that PollImmediateInfinite doesn't exit
			if !done {
				err = nil
			}
		}
		return
	})
}

func DebugObject(msg string, obj interface{}) {
	if log.GetLevel() == log.DebugLevel {
		b, _ := json.Marshal(obj)
		log.Debugf("%s: %s", msg, string(b))
	}
}
