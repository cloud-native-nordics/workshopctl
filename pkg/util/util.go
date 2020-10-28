package util

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
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

func Poll(ctx context.Context, d *time.Duration, fn wait.ConditionFunc) error {
	logger := Logger(ctx)

	logger.Traceln("Poll function started")
	defer logger.Traceln("Poll function quit")

	duration := 15 * time.Second
	if d != nil {
		duration = *d
	}
	// Set a deadline at 10 mins
	ctxWithDeadline, cancel := context.WithTimeout(ctx, 10*time.Minute)
	// releases resources if operation completes before timeout elapses
	defer cancel()

	tryCount := 0
	return wait.PollImmediateUntil(duration, func() (bool, error) {
		tryCount++
		errFn := logger.Debugf
		if tryCount%3 == 0 { // print info every third time
			errFn = logger.Infof
		}

		done, err := fn()
		logger.Tracef("Poll function (round %d) returned %t %v", tryCount, done, err)
		if err != nil {
			// if we're not "done" yet, set the err to nil so that PollImmediateInfinite doesn't exit
			if !done {
				errFn("Polling continues due to: %v", err)
				err = nil
			}
		}
		if IsDryRun(ctx) {
			logger.Info("This is a dry-run, hence one loop run is enough. Under normal circumstances, this loop would continue until the condition is met.")
			return true, nil
		}
		return done, err
	}, ctxWithDeadline.Done())
}

func DebugObject(ctx context.Context, msg string, obj interface{}) {
	// If debug logging isn't enabled, just exit
	if !log.IsLevelEnabled(log.DebugLevel) {
		return
	}

	logger := Logger(ctx)
	b, err := json.Marshal(obj)
	if err != nil {
		logger.Errorf("DebugObject failed with %v", err)
		return
	}
	logger.Debugf("%s: %s", msg, string(b))
}

// RandomSHA returns a hex-encoded string from {byteLen} random bytes.
func RandomSHA(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func ReadYAMLFile(file string, obj interface{}) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict(b, obj)
}

func WriteYAMLFile(ctx context.Context, file string, obj interface{}) error {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	return WriteFile(ctx, file, b)
}

func WriteFile(ctx context.Context, file string, b []byte) error {
	logger := Logger(ctx)
	if IsDryRun(ctx) {
		logger.Infof("Would write the following contents to file %q: %s", file, string(b))
		return nil
	}
	logger.Debugf("Writing the following contents to file %q: %s", file, string(b))
	return ioutil.WriteFile(file, b, 0644)
}

func DeletePath(ctx context.Context, fileOrFolder string) error {
	logger := Logger(ctx)
	if IsDryRun(ctx) {
		logger.Infof("Would delete path %q", fileOrFolder)
		return nil
	}
	logger.Debugf("Deleting path %q", fileOrFolder)
	return os.RemoveAll(fileOrFolder)
}

type KYAMLFilterFunc func(*kyaml.RNode) (*kyaml.RNode, error)

func KYAMLFilter(r io.Reader, w io.Writer, filters ...KYAMLFilterFunc) error {
	setAnnotationFn := kio.FilterFunc(func(operand []*kyaml.RNode) ([]*kyaml.RNode, error) {
		var err error
		for i := range operand {
			resource := operand[i]
			for _, filter := range filters {
				resource, err = filter(resource)
				if err != nil {
					return nil, err
				}
			}
		}
		return operand, nil
	})

	return kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: r}},
		Filters: []kio.Filter{setAnnotationFn},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: w}},
	}.Execute()
}

type KYAMLResourceMetaMatch struct {
	Kind      string
	Name      string
	Namespace string
	Func      func() error
}

func KYAMLResourceMetaMatcher(node *kyaml.RNode, matchStatements ...KYAMLResourceMetaMatch) error {
	meta, err := node.GetMeta()
	if err != nil {
		return err
	}
	for _, statement := range matchStatements {
		// If statement.Kind is set, it must match meta.Kind in order not be skipped
		if len(statement.Kind) != 0 && statement.Kind != meta.Kind {
			continue
		}
		// If statement.Name is set, it must match meta.Name in order not be skipped
		if len(statement.Name) != 0 && statement.Name != meta.Name {
			continue
		}
		// If statement.Namespace is set, it must match meta.Namespace in order not be skipped
		if len(statement.Namespace) != 0 && statement.Namespace != meta.Namespace {
			continue
		}
		// All case statements matched, let's run the function
		if err := statement.Func(); err != nil {
			return err
		}
	}
	return nil
}

func Command(ctx context.Context, command string, args ...string) *ExecUtil {
	return &ExecUtil{
		cmd:    exec.CommandContext(ctx, command, args...),
		outBuf: new(bytes.Buffer),
		ctx:    ctx,
		logger: Logger(ctx),
	}
}

func ShellCommand(ctx context.Context, format string, args ...interface{}) *ExecUtil {
	return Command(ctx, "/bin/sh", "-c", fmt.Sprintf(format, args...))
}

type ExecUtil struct {
	cmd    *exec.Cmd
	outBuf *bytes.Buffer
	ctx    context.Context
	logger *logrus.Entry
}

func (e *ExecUtil) Cmd() *exec.Cmd {
	return e.cmd
}

func (e *ExecUtil) WithStdio(stdin io.Reader, stdout, stderr io.Writer) *ExecUtil {
	if stdin != nil {
		e.logger.Debug("Set command stdin")
		e.cmd.Stdin = stdin
	}
	if stdout != nil {
		e.logger.Debug("Set command stdout")
		e.cmd.Stdout = stdout
	}
	if stderr != nil {
		e.logger.Debug("Set command stderr")
		e.cmd.Stderr = stderr
	}
	return e
}

func (e *ExecUtil) WithPwd(pwd string) *ExecUtil {
	e.logger.Debugf("Set command pwd: %q", pwd)
	e.cmd.Dir = pwd
	return e
}

func (e *ExecUtil) WithEnv(envVars ...string) *ExecUtil {
	e.logger.Debugf("Set command env vars: %v", envVars)
	e.cmd.Env = append(e.cmd.Env, envVars...)
	return e
}

func (e *ExecUtil) Run() (output string, exitCode int, cmdErr error) {
	cmdArgs := strings.Join(e.cmd.Args, " ")

	// Don't do this if we're dry-running
	if IsDryRun(e.ctx) {
		e.logger.Infof("Would execute command %q", cmdArgs)
		return "", 0, nil
	}

	// Always capture stdout output to e.outBuf
	if e.cmd.Stdout != nil {
		e.cmd.Stdout = io.MultiWriter(e.cmd.Stdout, e.outBuf)
	} else {
		e.cmd.Stdout = e.outBuf
	}
	// Always capture stderr output to e.outBuf
	if e.cmd.Stderr != nil {
		e.cmd.Stderr = io.MultiWriter(e.cmd.Stderr, e.outBuf)
	} else {
		e.cmd.Stderr = e.outBuf
	}
	// Run command
	e.logger.Debugf("Running command %q", cmdArgs)
	err := e.cmd.Run()

	// Capture combined output
	output = string(bytes.TrimSpace(e.outBuf.Bytes()))
	if len(output) != 0 {
		e.logger.Debugf("Command %q produced output: %s", cmdArgs, output)
	}

	// Handle the error
	if err != nil {
		exitCodeStr := "'unknown'"
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
			exitCodeStr = fmt.Sprintf("%d", exitCode)
		}

		cmdErr = fmt.Errorf("external command %q exited with code %s, error: %w and output: %s", cmdArgs, exitCodeStr, err, output)
		e.logger.Debugf("Command error: %v", cmdErr)
	}
	return
}
