package util

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/otiai10/copy"
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

func Poll(d *time.Duration, logger *log.Entry, fn wait.ConditionFunc, dryRun bool) error {
	logger.Traceln("Poll function started")
	defer logger.Traceln("Poll function quit")

	duration := 15 * time.Second
	if d != nil {
		duration = *d
	}
	if logger == nil {
		logger = log.NewEntry(log.StandardLogger())
	}
	tryCount := 0
	return wait.PollImmediate(duration, 10*time.Minute, func() (bool, error) {
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
		if dryRun {
			logger.Info("This is a dry-run, hence one loop run is enough. Under normal circumstances, this loop would continue until the condition is met.")
			return true, nil
		}
		return done, err
	})
}

func DebugObject(msg string, obj interface{}) {
	if log.IsLevelEnabled(log.DebugLevel) {
		b, err := json.Marshal(obj)
		if err != nil {
			log.Errorf("DebugObject failed with %v", err)
			return
		}
		log.Debugf("%s: %s", msg, string(b))
	}
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

func WriteYAMLFile(file string, obj interface{}) error {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, b, 0644)
}

func ApplyTemplate(tmpl string, data interface{}) (string, error) {
	buf := &bytes.Buffer{}
	if err := template.Must(template.New("tmpl").Delims("{{", "}}").Parse(tmpl)).Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExecPipe executes the given command, but reads from r and writes to w
func ExecPipe(r io.Reader, w io.Writer, pwd string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = r
	cmd.Stdout = w
	cmd.Stderr = os.Stdout
	if len(pwd) != 0 {
		cmd.Dir = pwd
	}
	log.Debugf("Running command: %q with pwd: %q", cmd.Args, cmd.Dir)

	return cmd.Run() // TODO: Output debugging
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
