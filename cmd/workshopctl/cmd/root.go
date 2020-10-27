package cmd

import (
	"io"
	"os"

	"github.com/cloud-native-nordics/workshopctl/pkg/logs"
	logflag "github.com/cloud-native-nordics/workshopctl/pkg/logs/flag"
	versioncmd "github.com/cloud-native-nordics/workshopctl/pkg/version/cmd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	logLevelFlag   = logrus.InfoLevel
	configPathFlag = "workshopctl.yaml"
	rootDirFlag    = "."
)

// NewIgniteCommand returns the root command for ignite
func NewWorkshopCtlCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "workshopctl",
		Short: "workshopctl: easily run Kubernetes workshops",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set the desired logging level, now that the flags are parsed
			logs.Logger.SetLevel(logLevelFlag)
		},
	}

	addGlobalFlags(root.PersistentFlags())

	root.AddCommand(NewInitCommand(os.Stdin, os.Stdout, os.Stderr))
	root.AddCommand(NewGenCommand())
	root.AddCommand(NewApplyCommand())
	root.AddCommand(NewKubectlCommand(os.Stdin, os.Stdout, os.Stderr))
	root.AddCommand(versioncmd.NewCmdVersion(os.Stdout))
	return root
}

func addGlobalFlags(fs *pflag.FlagSet) {
	logflag.LogLevelFlagVar(fs, &logLevelFlag)
	fs.StringVar(&rootDirFlag, "root-dir", rootDirFlag, "Where the workshopctl directory is. Must be a Git repo.")
	fs.StringVar(&configPathFlag, "config-path", configPathFlag, "Where to find the config file")
}
