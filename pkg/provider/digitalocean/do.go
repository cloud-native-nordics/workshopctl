package digitalocean

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/digitalocean/godo"
	"github.com/luxas/workshopctl/pkg/provider"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func NewDigitalOceanProvider(sa *provider.ServiceAccount, dryRun bool) provider.Provider {
	p := &DigitalOceanProvider{
		sa:     sa,
		dryRun: dryRun,
	}
	oauthClient := oauth2.NewClient(context.Background(), sa)
	p.c = godo.NewClient(oauthClient)
	return p
}

type DigitalOceanProvider struct {
	sa     *provider.ServiceAccount
	c      *godo.Client
	dryRun bool
}

func chooseSize(s provider.NodeSize) string {
	m := map[provider.NodeSize]string{
		provider.NodeSize{CPUs: 1, RAM: 2}:  "s-1vcpu-2gb",
		provider.NodeSize{CPUs: 1, RAM: 3}:  "s-1vcpu-3gb",
		provider.NodeSize{CPUs: 2, RAM: 2}:  "s-2vcpu-2gb",
		provider.NodeSize{CPUs: 2, RAM: 4}:  "s-2vcpu-4gb",
		provider.NodeSize{CPUs: 4, RAM: 8}:  "s-4vcpu-8gb",
		provider.NodeSize{CPUs: 6, RAM: 16}: "s-6vcpu-16gb",
	}
	if str, ok := m[s]; ok {
		return str
	}
	log.Warnf("didn't find a good size for you, fallback to s-1vcpu-2gb")
	return "s-1vcpu-2gb"
}

func (do *DigitalOceanProvider) CreateCluster(index uint16, c provider.ClusterSpec) (*provider.Cluster, error) {
	start := time.Now().UTC()
	cluster := &provider.Cluster{
		Spec: c,
		Status: provider.ClusterStatus{
			ProvisionStart: &start,
		},
	}

	nodePoolName := fmt.Sprintf("workshopctl-nodepool-%d", index)
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
		RegionSlug:  "fra1",
		VersionSlug: c.Version, // TODO: Resolve c.Version correctly
		Tags: []string{
			"workshopctl",
		},
		NodePools:   nodePool,
		AutoUpgrade: false,
	}

	if do.dryRun || log.GetLevel() == log.DebugLevel {
		b, _ := json.Marshal(req)
		if do.dryRun {
			log.Infof("Would send this request to DO: %s", string(b))
			return nil, nil
		}
		log.Debugf("Would send this request to DO: %s", string(b))
	}

	doCluster, _, err := do.c.Kubernetes.Create(context.Background(), req)
	if err != nil {
		return nil, err
	}
	cluster.Status.ID = doCluster.ID
	u, err := url.Parse(doCluster.Endpoint)
	if err != nil {
		return nil, err
	}
	cluster.Status.EndpointURL = u
	cluster.Status.EndpointIP = net.ParseIP(doCluster.IPv4)
	if log.GetLevel() == log.DebugLevel {
		b, _ := json.Marshal(cluster.Status)
		log.Debugf("Got a response from DO: %s", string(b))
	}

	for {
		time.Sleep(10 * time.Second)
		kcluster, _, err := do.c.Kubernetes.Get(context.Background(), cluster.Status.ID)
		if err != nil {
			log.Errorf("getting a kubernetes cluster failed: %v", err)
			continue
		}
		if kcluster.Status.State == godo.KubernetesClusterStatusRunning {
			log.Infof("Awesome! We're done! message: %q", kcluster.Status.Message)
			break
		}
		switch kcluster.Status.State {
		case godo.KubernetesClusterStatusProvisioning:
			log.Infof("cluster still provisioning! message: %q", kcluster.Status.Message)
			continue
		default:
			log.Warnf("unknown state %q! message: %q", kcluster.Status.State, kcluster.Status.Message)
		}
	}

	return cluster, nil
}
