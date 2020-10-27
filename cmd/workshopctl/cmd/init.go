package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewInitCommand returns the "init" command
func NewInitCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Setup the user configuration interactively",
		RunE:  RunInit,
	}

	addInitFlags(cmd.Flags())
	return cmd
}

func addInitFlags(fs *pflag.FlagSet) {}

func RunInit(cmd *cobra.Command, args []string) error {
	if util.FileExists(configPathFlag) {
		return nil
	}
	cfg := &config.Config{}
	if err := initConfig(cfg); err != nil {
		return err
	}
	return util.WriteYAMLFile(configPathFlag, cfg)
}

func initConfig(cfg *config.Config) error {
	if !filepath.IsAbs(cfg.RootDir) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		cfg.RootDir = filepath.Join(wd, cfg.RootDir)
	}
	if err := cfg.Complete(); err != nil {
		return err
	}
	return nil
}
