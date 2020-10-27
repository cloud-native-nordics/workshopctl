package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const EnvCluster = "WORKSHOPCTL_CLUSTER"

type KubectlFlags struct {
	*RootFlags

	Cluster uint16
}

// NewKubectlCommand returns the "kubectl" command
func NewKubectlCommand(rf *RootFlags) *cobra.Command {
	kf := &KubectlFlags{
		RootFlags: rf,
	}
	cmd := &cobra.Command{
		Use:   "kubectl [kubectl commands]",
		Short: "An alias for the kubectl command, pointing the KUBECONFIG to the right place",
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunKubectl(kf, args); err != nil {
				log.Fatal(err)
			}
		},
	}

	addKubectlFlags(cmd.Flags(), kf)
	return cmd
}

func addKubectlFlags(fs *pflag.FlagSet, kf *KubectlFlags) {
	fs.Uint16VarP(&kf.Cluster, "cluster", "c", kf.Cluster, fmt.Sprintf("What cluster number you want to connect to. Env var %s can also be used.", EnvCluster))
}

func RunKubectl(kf *KubectlFlags, args []string) error {
	ctx := util.NewContext(kf.DryRun)

	if kf.Cluster == 0 {
		clusterEnv := os.Getenv(EnvCluster)
		if clusterEnv == "" {
			return fmt.Errorf("Flag --cluster is mandatory")
		}
		cluster, err := strconv.Atoi(clusterEnv)
		if err != nil {
			return err
		}
		kf.Cluster = uint16(cluster)
	}
	cn := config.ClusterNumber(kf.Cluster)
	kubeconfigPath := filepath.Join(kf.RootDir, constants.ClustersDir, cn.String(), constants.KubeconfigFile)
	kubeconfigEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath)
	_, _, err := util.Command(ctx, "kubectl", args...).WithEnv(kubeconfigEnv).Run()
	return err
}
