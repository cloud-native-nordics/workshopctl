package cmd

import (
	"fmt"
	"io"

	"github.com/luxas/workshopctl/pkg/config"
	"github.com/luxas/workshopctl/pkg/provider"
	"github.com/luxas/workshopctl/pkg/provider/digitalocean"
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
	fs.StringVar(&cfg.ServiceAccount, "service-account", "", "What serviceaccount/token to use. Can be a string or a file")
}

func RunApply(cfg *config.Config) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		err := func() error {
			if cfg.Provider != "digitalocean" {
				return fmt.Errorf("no other providers but DO supported at the moment")
			}
			if len(cfg.ServiceAccount) == 0 {
				return fmt.Errorf("a serviceaccount is required")
			}
			sa := provider.NewServiceAccount(cfg.ServiceAccount)
			p := digitalocean.NewDigitalOceanProvider(sa, dryrun)
			i := uint16(1)
			cluster, err := p.CreateCluster(i, provider.ClusterSpec{
				Name: fmt.Sprintf("workshopctl-cluster-%d", i),
				NodeSize: provider.NodeSize{
					CPUs: cfg.CPUs,
					RAM:  cfg.RAM,
				},
				NodeCount: cfg.Clusters,
				Version:   "latest",
			})
			if err != nil {
				return err
			}
			fmt.Println(*cluster)
			return nil
		}()
		if err != nil {
			log.Fatal(err)
		}
	}
}
