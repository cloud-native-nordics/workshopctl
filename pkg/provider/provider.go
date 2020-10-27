package provider

import (
	"net"
	"net/url"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/gen"
)

type Cluster struct {
	Spec   ClusterSpec
	Status ClusterStatus
}

type ClusterSpec struct {
	Name       string
	Version    string
	NodeGroups []config.NodeGroup
}

type ClusterStatus struct {
	ID              string
	Index           config.ClusterNumber
	ProvisionStart  *time.Time
	ProvisionDone   *time.Time
	EndpointURL     *url.URL
	EndpointIP      net.IP
	KubeconfigBytes []byte
}

// TODO: Make a proper factory of this instead in sub-package providers/
type CloudProviderFunc func(*config.Provider, bool) CloudProvider

type CloudProvider interface {
	// CreateCluster creates a cluster. This call is _blocking_ until the cluster is properly provisioned
	CreateCluster(index config.ClusterNumber, c ClusterSpec) (*Cluster, error)
}

type DNSProvider interface {
	ChartProcessors() []gen.Processor
	ValuesProcessors() []gen.Processor
}
