package cmd

import (
	"io"

	"github.com/luxas/workshopctl/pkg/apply"
	"github.com/luxas/workshopctl/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var dryrun = true

// NewApplyCommand returns the "apply" command
func NewApplyCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cfg := &config.Config{}
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Create a Kubernetes cluster and apply the desired manifests",
		Run:   RunApply(cfg),
	}

	addApplyFlags(cmd.Flags(), cfg)
	return cmd
}

func addApplyFlags(fs *pflag.FlagSet, cfg *config.Config) {
	addGenFlags(fs, cfg)
	fs.BoolVar(&dryrun, "dry-run", dryrun, "Whether to dry-run or not")
	fs.Uint16Var(&cfg.CPUs, "node-cpus", 2, "How much CPUs to use per-node")
	fs.Uint16Var(&cfg.RAM, "node-ram", 2, "How much RAM to use per-node")
	fs.Uint16Var(&cfg.NodeCount, "node-count", 1, "How many nodes per cluster")
	fs.StringVar(&cfg.VSCodePassword, "vscode-password", "kubernetesrocks", "What the password for Visual Studio Code should be")
	// TODO: This should be a custom flag
	fs.StringVar(&cfg.ServiceAccountStr, "service-account", "", "What serviceaccount/token to use. Can be a string or a file")
}

func RunApply(cfg *config.Config) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		err := func() error {
			return apply.Apply(cfg, dryrun)
		}()
		if err != nil {
			log.Fatal(err)
		}
	}
}
