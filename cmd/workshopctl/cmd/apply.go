package cmd

import (
	"github.com/cloud-native-nordics/workshopctl/pkg/apply"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var dryrun = true

// NewApplyCommand returns the "apply" command
func NewApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Create a Kubernetes cluster and apply the desired manifests",
		Run:   RunApply,
	}

	addApplyFlags(cmd.Flags())
	return cmd
}

func addApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&dryrun, "dry-run", dryrun, "Whether to dry-run or not")
}

func RunApply(cmd *cobra.Command, args []string) {
	err := func() error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return apply.Apply(cfg, dryrun)
	}()
	if err != nil {
		log.Fatal(err)
	}
}
