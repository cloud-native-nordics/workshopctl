package cmd

import (
	"os"
	"path/filepath"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type InitFlags struct {
	*RootFlags
}

// NewInitCommand returns the "init" command
func NewInitCommand(rf *RootFlags) *cobra.Command {
	inf := &InitFlags{
		RootFlags: rf,
	}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Setup the user configuration interactively",
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunInit(inf); err != nil {
				log.Fatal(err)
			}
		},
	}

	addInitFlags(cmd.Flags(), inf)
	return cmd
}

func addInitFlags(fs *pflag.FlagSet, inf *InitFlags) {}

func RunInit(inf *InitFlags) error {
	if util.FileExists(inf.ConfigPath) {
		return nil
	}
	cfg := &config.Config{}
	if err := initConfig(cfg); err != nil {
		return err
	}
	return util.WriteYAMLFile(inf.ConfigPath, cfg)
}

func initConfig(cfg *config.Config) error {
	if !filepath.IsAbs(cfg.RootDir) { // TODO: This probably doesn't work yet
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
