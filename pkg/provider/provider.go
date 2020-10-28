package provider

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/gen"
)

type Cluster struct {
	ClusterMeta
	Spec   ClusterSpec
	Status ClusterStatus
}

type ClusterMeta struct {
	NamePrefix string
	Index      config.ClusterNumber
}

func (m ClusterMeta) Name() string {
	return constants.ClusterName(m.NamePrefix, m.Index)
}

type ClusterSpec struct {
	Version    string
	NodeGroups []config.NodeGroup
}

type ClusterStatus struct {
	ID              string
	ProvisionStart  *time.Time
	ProvisionDone   *time.Time
	EndpointURL     *url.URL
	EndpointIP      net.IP
	KubeconfigBytes []byte
}

func (s ClusterStatus) ProvisionTime() time.Duration {
	if s.ProvisionStart == nil || s.ProvisionDone == nil {
		return 0
	}
	return s.ProvisionDone.Sub(*s.ProvisionStart)
}

type CloudProviderFactory interface {
	NewCloudProvider(ctx context.Context, p *config.Provider) (CloudProvider, error)
}

type CloudProvider interface {
	// CreateCluster creates a cluster. This call is _blocking_ until the cluster is properly provisioned
	CreateCluster(ctx context.Context, m ClusterMeta, c ClusterSpec) (*Cluster, error)
	// DeleteCluster deletes a cluster and its associated load balancers
	DeleteCluster(ctx context.Context, m ClusterMeta) error
}

type DNSProviderFactory interface {
	NewDNSProvider(ctx context.Context, p *config.Provider, rootDomain string) (DNSProvider, error)
}

type DNSProvider interface {
	ChartProcessors() []gen.Processor
	ValuesProcessors() []gen.Processor
	// CleanupRecords deletes records associated with a cluster
	CleanupRecords(ctx context.Context, m ClusterMeta) error
}
