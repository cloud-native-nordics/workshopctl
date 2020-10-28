package cmd

import (
	"github.com/cloud-native-nordics/workshopctl/pkg/apply"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ApplyFlags struct {
	*RootFlags
}

// NewApplyCommand returns the "apply" command
func NewApplyCommand(rf *RootFlags) *cobra.Command {
	af := &ApplyFlags{
		RootFlags: rf,
	}
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Create a Kubernetes cluster and apply the desired manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunApply(af); err != nil {
				log.Fatal(err)
			}
		},
	}

	addApplyFlags(cmd.Flags(), af)
	return cmd
}

func addApplyFlags(fs *pflag.FlagSet, af *ApplyFlags) {}

func RunApply(af *ApplyFlags) error {
	ctx := util.NewContext(af.DryRun, af.RootDir)
	cfg, err := loadConfig(ctx, af.ConfigPath)
	if err != nil {
		return err
	}
	return apply.Apply(ctx, cfg)
}
