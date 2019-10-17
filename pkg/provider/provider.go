package provider

import (
	"io/ioutil"
	"net"
	"net/url"
	"time"

	"github.com/luxas/workshopctl/pkg/util"
	"golang.org/x/oauth2"
)

func NewServiceAccount(pathOrToken string) *ServiceAccount {
	if util.FileExists(pathOrToken) {
		return &ServiceAccount{
			path: pathOrToken,
		}
	}
	return &ServiceAccount{
		token: pathOrToken,
	}
}

type ServiceAccount struct {
	token, path string
}

func (sa *ServiceAccount) Token() (*oauth2.Token, error) {
	t, err := sa.Get()
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken: t,
	}, nil
}

func (sa *ServiceAccount) Get() (string, error) {
	if sa.token != "" {
		return sa.token, nil
	}
	b, err := ioutil.ReadFile(sa.path)
	return string(b), err
}

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
	ID             string
	ProvisionStart *time.Time
	ProvisionDone  *time.Time
	EndpointURL    *url.URL
	EndpointIP     net.IP
}

type Provider interface {
	// CreateCluster creates a cluster. This call is _blocking_ until the cluster is properly provisioned
	CreateCluster(index uint16, c ClusterSpec) (*Cluster, error)
}
