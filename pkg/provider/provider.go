package provider

import (
	"net"
	"net/url"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
)

type NodeSize struct {
	CPUs uint16
	RAM  uint16
}

type Cluster struct {
	Spec   ClusterSpec
	Status ClusterStatus
}

type ClusterSpec struct {
	NodeSize  NodeSize
	NodeCount uint16
	Name      string
	Version   string
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

type ProviderFunc func(*config.ServiceAccount, bool) Provider

type Provider interface {
	// CreateCluster creates a cluster. This call is _blocking_ until the cluster is properly provisioned
	CreateCluster(index config.ClusterNumber, c ClusterSpec) (*Cluster, error)
}
