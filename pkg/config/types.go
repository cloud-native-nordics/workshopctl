package config

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type Config struct {
	RootDomain string `json:"rootDomain"`
	Clusters   uint16 `json:"clusters"`
	GitRepo    string `json:"gitRepo"`
	RootDir    string `json:"-"`

	Provider       string `json:"provider"`
	ServiceAccount string `json:"serviceAccount"`
	CPUs           uint16 `json:"cpus"`
	RAM            uint16 `json:"ram"`
	NodeCount      uint16 `json:"nodeCount"`
}

type ClusterInfo struct {
	*Config
	Index  ClusterNumber
	Logger *logrus.Entry
}

func NewClusterInfo(cfg *Config, i ClusterNumber) *ClusterInfo {
	return &ClusterInfo{cfg, i, logrus.WithField("cluster", i)}
}

func (c *ClusterInfo) KubeConfigPath() string {
	return fmt.Sprintf("clusters/%s/kubeconfig.private.yaml", c.Index)
}

func (c *ClusterInfo) ClusterDir() string {
	return fmt.Sprintf("clusters/%s", c.Index)
}

func (c *ClusterInfo) Domain() string {
	return fmt.Sprintf("cluster-%s.%s", c.Index, c.RootDomain)
}

var _ fmt.Stringer = ClusterNumber(0)

type ClusterNumber uint16

func (n ClusterNumber) String() string {
	return fmt.Sprintf("%02d", n)
}

func ForCluster(n uint16, cfg *Config, fn func(*ClusterInfo) error) error {
	wg := &sync.WaitGroup{}
	wg.Add(int(n))
	foundErr := false
	for i := ClusterNumber(1); i <= ClusterNumber(n); i++ {
		go func(j ClusterNumber) {
			defer wg.Done()
			clusterInfo := NewClusterInfo(cfg, j)
			if err := fn(clusterInfo); err != nil {
				clusterInfo.Logger.Error(err)
				foundErr = true
			}
		}(i)
	}
	wg.Wait()
	if foundErr {
		return fmt.Errorf("an error occured previously")
	}
	return nil
}
