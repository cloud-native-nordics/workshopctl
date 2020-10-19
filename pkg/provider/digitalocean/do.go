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

const Region = "fra1"

func NewDigitalOceanProvider(sa *config.ServiceAccount, dryRun bool) provider.Provider {
	p := &DigitalOceanProvider{
		sa:     sa,
		dryRun: dryRun,
	}
	oauthClient := oauth2.NewClient(context.Background(), sa)
	p.c = godo.NewClient(oauthClient)
	return p
}

type DigitalOceanProvider struct {
	sa     *config.ServiceAccount
	c      *godo.Client
	dryRun bool
}

func chooseSize(s provider.NodeSize) string {
	m := map[provider.NodeSize]string{
		{CPUs: 1, RAM: 2}:  "s-1vcpu-2gb",
		{CPUs: 1, RAM: 3}:  "s-1vcpu-3gb",
		{CPUs: 2, RAM: 2}:  "s-2vcpu-2gb",
		{CPUs: 2, RAM: 4}:  "s-2vcpu-4gb",
		{CPUs: 4, RAM: 8}:  "s-4vcpu-8gb",
		{CPUs: 6, RAM: 16}: "s-6vcpu-16gb",
	}
	if str, ok := m[s]; ok {
		return str
	}
	log.Warnf("didn't find a good size for you, fallback to s-1vcpu-2gb")
	return "s-1vcpu-2gb"
}

func (do *DigitalOceanProvider) CreateCluster(i config.ClusterNumber, c provider.ClusterSpec) (*provider.Cluster, error) {
	logger := i.NewLogger()

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
			Size:      chooseSize(c.NodeSize),
			Count:     int(c.NodeCount),
			AutoScale: false,
			Tags: []string{
				"workshopctl",
				nodePoolName,
			},
		},
	}

	req := &godo.KubernetesClusterCreateRequest{
		Name:        c.Name,
		RegionSlug:  Region,
		VersionSlug: c.Version, // TODO: Resolve c.Version correctly
		Tags: []string{
			"workshopctl",
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
	clusters, _, err := do.c.Kubernetes.List(context.Background(), &godo.ListOptions{})
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
		doCluster, _, err := do.c.Kubernetes.Create(context.Background(), req)
		if err != nil {
			return nil, err
		}
		cluster.Status.ID = doCluster.ID
	}

	err = util.Poll(nil, logger, func() (bool, error) {
		kcluster, _, err := do.c.Kubernetes.Get(context.Background(), cluster.Status.ID)
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
	cc, _, err := do.c.Kubernetes.GetKubeConfig(context.Background(), cluster.Status.ID)
	if err != nil {
		return nil, err
	}
	cluster.Status.KubeconfigBytes = cc.KubeconfigYAML

	return cluster, nil
}
