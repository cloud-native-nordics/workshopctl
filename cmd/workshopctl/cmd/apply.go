package cmd

import (
	"context"

	"github.com/cloud-native-nordics/workshopctl/pkg/apply"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ApplyFlags struct {
	*RootFlags

	DryRun bool
}

// NewApplyCommand returns the "apply" command
func NewApplyCommand(rf *RootFlags) *cobra.Command {
	af := &ApplyFlags{
		RootFlags: rf,
		DryRun:    true,
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

func addApplyFlags(fs *pflag.FlagSet, af *ApplyFlags) {
	fs.BoolVar(&af.DryRun, "dry-run", af.DryRun, "Whether to dry-run or not")
}

func RunApply(af *ApplyFlags) error {
	ctx := context.Background()
	cfg, err := loadConfig(af.ConfigPath)
	if err != nil {
		return err
	}
	return apply.Apply(ctx, cfg, af.DryRun)
}
