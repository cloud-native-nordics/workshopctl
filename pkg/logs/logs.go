package logs

import (
	golog "log"
	"os"

	log "github.com/sirupsen/logrus"
)

// Wrap the logrus logger together with the exit code
// so we can control what log.Fatal returns
type logger struct {
	*log.Logger
	ExitCode int
}

func newLogger() *logger {
	l := &logger{
		Logger:   log.StandardLogger(), // Use the standard logrus logger
		ExitCode: 1,
	}

	l.ExitFunc = func(_ int) {
		os.Exit(l.ExitCode)
	}

	return l
}

// Expose the logger
var Logger *logger

// Automatically initialize the logging system for Ignite
func init() {
	// Initialize the logger
	Logger = newLogger()

	// Disable timestamp logging, but still output the seconds elapsed
	Logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    false,
	})

	// Disable the stdlib's automatic add of the timestamp in beginning of the log message,
	// as we stream the logs from stdlib log to this logrus instance.
	golog.SetFlags(0)
	golog.SetOutput(Logger.Writer())
}
