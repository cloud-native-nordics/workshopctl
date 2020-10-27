package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	// TODO: Add a context here for debugging and dry-running
	_, err := util.ExecForeground("/bin/sh", "-c",
		fmt.Sprintf(`KUBECONFIG=%s kubectl %s`, kubeconfigPath, strings.Join(args, " ")),
	)
	return err
}
