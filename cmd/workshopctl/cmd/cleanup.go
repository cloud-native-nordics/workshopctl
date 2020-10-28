package cmd

import (
	"context"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider/providers"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CleanupFlags struct {
	*RootFlags
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

func addCleanupFlags(fs *pflag.FlagSet, cf *CleanupFlags) {}

func RunCleanup(cf *CleanupFlags) error {
	ctx := util.NewContext(cf.DryRun, cf.RootDir)

	cfg, err := loadConfig(ctx, cf.ConfigPath)
	if err != nil {
		return err
	}

	cloudP, err := providers.CloudProviders().NewCloudProvider(ctx, &cfg.CloudProvider)
	if err != nil {
		return err
	}

	dnsP, err := providers.DNSProviders().NewDNSProvider(ctx, cfg.DNSProvider, cfg.RootDomain)
	if err != nil {
		return err
	}

	return config.ForCluster(ctx, cfg.Clusters, cfg, func(clusterCtx context.Context, clusterInfo *config.ClusterInfo) error {
		// Delete the Kubernetes cluster
		if err := cloudP.DeleteCluster(clusterCtx, clusterInfo.Index); err != nil {
			return err
		}
		// Delete the KubeConfig file
		kubeconfigPath := util.JoinPaths(ctx, clusterInfo.Index.KubeConfigPath())
		if err := util.DeletePath(clusterCtx, kubeconfigPath); err != nil {
			return err
		}
		// Delete the DNS records
		return dnsP.CleanupRecords(clusterCtx, clusterInfo.Index)
	})
}
