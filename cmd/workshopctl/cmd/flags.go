package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/spf13/pflag"
)

const (
	EnvCluster     = "WORKSHOPCTL_CLUSTER"
	EnvClusterDesc = "What cluster number you want to connect to. Env var " + EnvCluster + " can also be used."
)

type ClusterFlag uint16

func (f *ClusterFlag) String() string {
	if f == nil {
		*f = ClusterFlag(0)
	}
	if *f == 0 {
		clusterEnv := os.Getenv(EnvCluster)
		if clusterEnv != "" {
			_ = f.Set(clusterEnv)
		}
	}
	return fmt.Sprintf("%d", *f)
}
func (f *ClusterFlag) Set(str string) error {
	clusterNum, err := strconv.Atoi(str)
	if err != nil {
		return err
	}
	*f = ClusterFlag(clusterNum)
	return nil
}
func (f ClusterFlag) Type() string { return "cluster-number" }

func (f ClusterFlag) Number() config.ClusterNumber {
	return config.ClusterNumber(uint16(f))
}

func AddClusterFlag(fs *pflag.FlagSet, cf *ClusterFlag) {
	fs.VarP(cf, "cluster", "c", EnvClusterDesc)
}
