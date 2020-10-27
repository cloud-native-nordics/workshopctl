package digitalocean

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	"github.com/digitalocean/godo"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	DefaultRegion  = "fra1"
	RegionKey      = "region"
	WorkshopctlTag = "workshopctl"
)

type doCommon struct {
	p      *config.Provider
	c      *godo.Client
	dryRun bool
}

func initCommon(ctx context.Context, p *config.Provider) doCommon {
	oauthClient := oauth2.NewClient(ctx, p.TokenSource())
	return doCommon{
		p:      p,
		c:      godo.NewClient(oauthClient),
		dryRun: util.IsDryRun(ctx),
	}
}

func NewDigitalOceanCloudProvider(ctx context.Context, p *config.Provider) (provider.CloudProvider, error) {
	doProvider := &DigitalOceanCloudProvider{
		doCommon: initCommon(ctx, p),
		region:   DefaultRegion,
	}

	if r, ok := p.ProviderSpecific[RegionKey]; ok {
		doProvider.region = r
	}
	return doProvider, nil
}

type DigitalOceanCloudProvider struct {
	doCommon

	region string
}

func chooseSize(c config.NodeClaim) string {
	m := map[config.NodeClaim]string{
		{CPU: 2, RAM: 2, Dedicated: false}:  "s-2vcpu-2gb",  // $15
		{CPU: 2, RAM: 4, Dedicated: false}:  "s-2vcpu-4gb",  // $20
		{CPU: 4, RAM: 8, Dedicated: false}:  "s-4vcpu-8gb",  // $40
		{CPU: 8, RAM: 16, Dedicated: false}: "s-8vcpu-16gb", // $80
		{CPU: 2, RAM: 4, Dedicated: true}:   "c-2-4gib",     // $40
		{CPU: 4, RAM: 8, Dedicated: true}:   "c-4-8gib",     // $80
	}
	if str, ok := m[c]; ok {
		return str
	}
	log.Warnf("didn't find a good size for you, fallback to s-2vcpu-4gb")
	return "s-2vcpu-4gb"
}

func (do *DigitalOceanCloudProvider) CreateCluster(ctx context.Context, i config.ClusterNumber, c provider.ClusterSpec) (*provider.Cluster, error) {
	logger := util.Logger(ctx)

	start := time.Now().UTC()
	cluster := &provider.Cluster{
		Spec: c,
		Status: provider.ClusterStatus{
			ProvisionStart: &start,
			Index:          i,
		},
	}

	nodePoolName := fmt.Sprintf("workshopctl-nodepool-%s", i)
	nodePool := []*godo.KubernetesNodePoolCreateRequest{
		{
			Name:      nodePoolName,
			Size:      chooseSize(c.NodeGroups[0].NodeClaim), // TODO
			Count:     int(c.NodeGroups[0].Instances),
			AutoScale: false,
			Tags: []string{
				WorkshopctlTag,
				nodePoolName,
			},
		},
	}

	req := &godo.KubernetesClusterCreateRequest{
		Name:        c.Name,
		RegionSlug:  do.region,
		VersionSlug: c.Version, // TODO: Resolve c.Version correctly
		Tags: []string{
			WorkshopctlTag,
		},
		NodePools:   nodePool,
		AutoUpgrade: false,
	}

	if do.dryRun || log.IsLevelEnabled(log.DebugLevel) {
		b, _ := json.Marshal(req)
		if do.dryRun {
			log.Infof("Would send this request to DO: %s", string(b))
			return cluster, nil
		}
		log.Debugf("Would send this request to DO: %s", string(b))
	}
	// TODO: Rate limiting
	clusters, _, err := do.c.Kubernetes.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, c := range clusters {
		if c.Name == cluster.Spec.Name {
			logger.Infof("Found existing cluster with name %q and ID %q", c.Name, c.ID)
			cluster.Status.ID = c.ID
			break
		}
	}

	if len(cluster.Status.ID) == 0 {
		logger.Infof("Creating new cluster with name %s", cluster.Spec.Name)
		doCluster, _, err := do.c.Kubernetes.Create(ctx, req)
		if err != nil {
			return nil, err
		}
		cluster.Status.ID = doCluster.ID
	}

	err = util.Poll(ctx, nil, func() (bool, error) {
		kcluster, _, err := do.c.Kubernetes.Get(ctx, cluster.Status.ID)
		if err != nil {
			return false, fmt.Errorf("getting a kubernetes cluster failed: %v", err)
		}
		util.DebugObject("Got Kubernetes cluster response from DO", kcluster)

		if kcluster.Status.State == godo.KubernetesClusterStatusRunning {
			logger.Infof("Awesome, the cluster is Ready! Endpoints: %s %s", kcluster.Endpoint, kcluster.IPv4)

			u, err := url.Parse(kcluster.Endpoint)
			if err != nil {
				return true, err // fatal; exit
			}
			cluster.Status.EndpointURL = u
			cluster.Status.EndpointIP = net.ParseIP(kcluster.IPv4)
			now := time.Now().UTC()
			cluster.Status.ProvisionDone = &now

			return true, nil
		}
		if kcluster.Status.State == godo.KubernetesClusterStatusProvisioning {
			return false, fmt.Errorf("Cluster is still provisioning")
		}

		return false, fmt.Errorf("Unknown state %q! Message: %q", kcluster.Status.State, kcluster.Status.Message)
	})
	if err != nil {
		return nil, err
	}

	log.Infof("Downloading KubeConfig...")
	cc, _, err := do.c.Kubernetes.GetKubeConfig(ctx, cluster.Status.ID)
	if err != nil {
		return nil, err
	}
	cluster.Status.KubeconfigBytes = cc.KubeconfigYAML

	return cluster, nil
}

func (do *DigitalOceanCloudProvider) DeleteCluster(ctx context.Context, index config.ClusterNumber) error {
	return nil // TODO: implement this
}
