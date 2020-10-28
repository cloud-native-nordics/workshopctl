package cmd

import (
	"context"
	"fmt"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider/providers"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CleanupFlags struct {
	*RootFlags

	Cluster ClusterFlag
}

// NewCleanupCommand returns the "cleanup" command
func NewCleanupCommand(rf *RootFlags) *cobra.Command {
	cf := &CleanupFlags{
		RootFlags: rf,
	}
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete the k8s-managed cluster",
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunCleanup(cf); err != nil {
				log.Fatal(err)
			}
		},
	}

	addCleanupFlags(cmd.Flags(), cf)
	return cmd
}

func addCleanupFlags(fs *pflag.FlagSet, cf *CleanupFlags) {
	AddClusterFlag(fs, &cf.Cluster)
}

func RunCleanup(cf *CleanupFlags) error {
	if cf.Cluster == 0 {
		return fmt.Errorf("--cluster is required")
	}

	ctx := util.NewContext(true)

	cfg, err := loadConfig(ctx, cf.ConfigPath)
	if err != nil {
		return err
	}

	cloudP, err := providers.CloudProviders().NewCloudProvider(ctx, &cfg.CloudProvider)
	if err != nil {
		return err
	}
	if err := cloudP.DeleteCluster(ctx, config.ClusterNumber(1)); err != nil {
		return err
	}

	dnsP, err := providers.DNSProviders().NewDNSProvider(context.Background(), cfg.DNSProvider, cfg.RootDomain)
	if err != nil {
		return err
	}
	return dnsP.CleanupRecords(context.Background(), config.ClusterNumber(2))
}
