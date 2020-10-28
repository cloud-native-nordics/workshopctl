package digitalocean

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	"github.com/digitalocean/godo"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var clusterNotFound = fmt.Errorf("couldn't find cluster by name")

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
		{CPU: 2, RAM: 4, Dedicated: true}:   "c-2",          // $40
		{CPU: 4, RAM: 8, Dedicated: true}:   "c-4",          // $80
	}
	if str, ok := m[c]; ok {
		return str
	}
	log.Warnf("didn't find a good size for you, fallback to s-2vcpu-4gb")
	return "s-2vcpu-4gb"
}

func (do *DigitalOceanCloudProvider) CreateCluster(ctx context.Context, c provider.ClusterSpec) (*provider.Cluster, error) {
	logger := util.Logger(ctx)

	start := time.Now().UTC()
	i := c.Index
	cluster := &provider.Cluster{
		Spec: c,
		Status: provider.ClusterStatus{
			ProvisionStart: &start,
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
		Name:        cluster.Spec.Name(),
		RegionSlug:  do.region,
		VersionSlug: cluster.Spec.Version, // TODO: Resolve c.Version correctly
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
			// TODO: Revamp this dry-run logic and unify it with DebugObject
			return cluster, nil
		}
		log.Debugf("Would send this request to DO: %s", string(b))
	}
	// TODO: Rate limiting
	doCluster, err := do.getClusterByName(ctx, cluster.Spec.Name())
	if err == nil {
		// If the cluster was found, just note it's ID
		cluster.Status.ID = doCluster.ID
		logger.Infof("Found existing cluster with name %q and ID %q", cluster.Spec.Name(), cluster.Status.ID)

	} else if errors.Is(err, clusterNotFound) {
		// If the cluster wasn't found, create it
		logger.Infof("Creating new cluster with name %s", cluster.Spec.Name())
		doCluster, _, err = do.c.Kubernetes.Create(ctx, req)
		if err != nil {
			return nil, err
		}
		cluster.Status.ID = doCluster.ID
	} else { // unexpected err != nil
		return nil, err
	}

	err = util.Poll(ctx, nil, func() (bool, error) {
		kcluster, _, err := do.c.Kubernetes.Get(ctx, cluster.Status.ID)
		if err != nil {
			return false, fmt.Errorf("getting a kubernetes cluster failed: %v", err)
		}
		util.DebugObject(ctx, "Got Kubernetes cluster response from DO", kcluster)

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
	name := constants.ClusterName(index)

	cluster, err := do.getClusterByName(ctx, name)
	if err != nil {
		return err
	}

	util.DebugObject(ctx, "Found wanted cluster", cluster)

	// List all relevant LBs
	lbs, err := do.listLBsForCluster(ctx, cluster)
	if err != nil {
		return err
	}
	util.DebugObject(ctx, "LBs", lbs)

	for _, lb := range lbs {
		if err := do.deleteLB(ctx, lb); err != nil {
			return err
		}
	}

	return do.deleteCluster(ctx, cluster)
}

func (do *DigitalOceanCloudProvider) getClusterByName(ctx context.Context, name string) (*godo.KubernetesCluster, error) {
	logger := util.Logger(ctx)

	logger.Debug("Listing Kubernetes clusters...")
	clusters, _, err := do.c.Kubernetes.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, cluster := range clusters {
		// Filter by name
		if cluster.Name != name {
			logger.Debugf("Cluster name %s isn't desired %s", cluster.Name, name)
			continue
		}
		// Found it
		return cluster, nil
	}

	return nil, fmt.Errorf("%w: %s", clusterNotFound, name)
}

func (do *DigitalOceanCloudProvider) deleteCluster(ctx context.Context, c *godo.KubernetesCluster) error {
	logger := util.Logger(ctx)

	if util.IsDryRun(ctx) {
		logger.Infof("Would delete Kubernetes cluster %s", c.Name)
		return nil
	}
	logger.Infof("Deleting Kubernetes cluster %s", c.Name)
	_, err := do.c.Kubernetes.Delete(ctx, c.ID)
	return err
}

func (do *DigitalOceanCloudProvider) listLBsForCluster(ctx context.Context, cluster *godo.KubernetesCluster) ([]godo.LoadBalancer, error) {
	logger := util.Logger(ctx)

	droplets := map[string]struct{}{}
	for _, nodePool := range cluster.NodePools {
		for _, node := range nodePool.Nodes {
			logger.Debugf("Found droplet with ID %s in node pool %s", node.DropletID, nodePool.Name)
			droplets[node.DropletID] = struct{}{}
		}
	}

	lbs, _, err := do.c.LoadBalancers.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterLBs := []godo.LoadBalancer{}
	for _, lb := range lbs {
		// Is there any droplet in our current cluster that is served by this LB?
		found := false
		for _, lbDropletID := range lb.DropletIDs {
			// lbDropletID is an int but the same droplet IDs above are strings, hence cast this to a string
			lbDropletIDStr := strconv.Itoa(lbDropletID)
			if _, ok := droplets[lbDropletIDStr]; ok {
				logger.Debugf("LB %s is served by droplet with ID %s", lb.Name, lbDropletIDStr)
				found = true
				break
			}
		}
		// If this LB didn't match any of our known droplets, it's isn't ours so proceed
		if !found {
			logger.Debugf("LB %s isn't served by any of the droplets in this cluster", lb.Name)
			continue
		}
		// Append to the list
		clusterLBs = append(clusterLBs, lb)
	}
	return clusterLBs, nil
}

func (do *DigitalOceanCloudProvider) deleteLB(ctx context.Context, lb godo.LoadBalancer) error {
	logger := util.Logger(ctx)

	if util.IsDryRun(ctx) {
		logger.Infof("Would delete Kubernetes load balancer %s", lb.Name)
		return nil
	}
	logger.Infof("Deleting Kubernetes load balancer %s", lb.Name)
	_, err := do.c.LoadBalancers.Delete(ctx, lb.ID)
	return err
}
