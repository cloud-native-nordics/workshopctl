package cmd

import (
	"os"

	"github.com/cloud-native-nordics/workshopctl/pkg/logs"
	logflag "github.com/cloud-native-nordics/workshopctl/pkg/logs/flag"
	versioncmd "github.com/cloud-native-nordics/workshopctl/pkg/version/cmd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type RootFlags struct {
	LogLevel   logrus.Level
	ConfigPath string
	RootDir    string
	DryRun     bool
}

// NewWorkshopCtlCommand returns the root command for workshopctl
func NewWorkshopCtlCommand() *cobra.Command {
	rf := &RootFlags{
		LogLevel:   logrus.InfoLevel,
		ConfigPath: "workshopctl.yaml",
		RootDir:    ".",
		DryRun:     true,
	}
	root := &cobra.Command{
		Use:   "workshopctl",
		Short: "workshopctl: easily run Kubernetes workshops",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set the desired logging level, now that the flags are parsed
			logs.Logger.SetLevel(rf.LogLevel)
		},
	}

	addGlobalFlags(root.PersistentFlags(), rf)

	root.AddCommand(NewInitCommand(rf))
	root.AddCommand(NewGenCommand(rf))
	root.AddCommand(NewApplyCommand(rf))
	root.AddCommand(NewKubectlCommand(rf))
	root.AddCommand(NewCleanupCommand(rf))
	root.AddCommand(versioncmd.NewCmdVersion(os.Stdout))
	return root
}

func addGlobalFlags(fs *pflag.FlagSet, rf *RootFlags) {
	logflag.LogLevelFlagVar(fs, &rf.LogLevel)
	fs.StringVar(&rf.RootDir, "root-dir", rf.RootDir, "Where the workshopctl directory is. Must be a Git repo.")
	fs.StringVar(&rf.ConfigPath, "config-path", rf.ConfigPath, "Where to find the config file")
	fs.BoolVar(&rf.DryRun, "dry-run", rf.DryRun, "Whether to apply the selected operation, or just print what would happen (to dry-run)")
}
