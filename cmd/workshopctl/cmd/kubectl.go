package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type KubectlFlags struct {
	*RootFlags

	Cluster ClusterFlag
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
	AddClusterFlag(fs, &kf.Cluster)
}

func RunKubectl(kf *KubectlFlags, args []string) error {
	if kf.Cluster == 0 {
		return fmt.Errorf("--cluster is required")
	}

	ctx := util.NewContext(kf.DryRun)

	cn := kf.Cluster.Number()
	kubeconfigPath := filepath.Join(kf.RootDir, constants.ClustersDir, cn.String(), constants.KubeconfigFile)
	kubeconfigEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath)
	_, _, err := util.Command(ctx, "kubectl", args...).WithEnv(kubeconfigEnv).Run()
	return err
}
