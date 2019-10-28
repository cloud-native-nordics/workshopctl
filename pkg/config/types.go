package config

import (
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/luxas/workshopctl/pkg/util"
	"github.com/sirupsen/logrus"
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
	// Cache the token from the file
	sa.token = string(b)
	return sa.token, err
}

type Config struct {
	RootDomain string `json:"rootDomain"`
	Clusters   uint16 `json:"clusters"`
	GitRepo    string `json:"gitRepo"`
	RootDir    string `json:"-"`

	VSCodePassword    string `json:"vsCodePassword"`
	Provider          string `json:"provider"`
	ServiceAccountStr string `json:"serviceAccount"`
	CPUs              uint16 `json:"cpus"`
	RAM               uint16 `json:"ram"`
	NodeCount         uint16 `json:"nodeCount"`

	ServiceAccount *ServiceAccount `json:"-"`
}

type ClusterInfo struct {
	*Config
	Index  ClusterNumber
	Logger *logrus.Entry
}

func NewClusterInfo(cfg *Config, i ClusterNumber) *ClusterInfo {
	return &ClusterInfo{cfg, i, i.NewLogger()}
}

func (c *ClusterInfo) KubeConfigPath() string {
	return fmt.Sprintf("clusters/%s/.kubeconfig", c.Index)
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

func (n ClusterNumber) NewLogger() *logrus.Entry {
	return logrus.WithField("cluster", n)
}

func ForCluster(n uint16, cfg *Config, fn func(*ClusterInfo) error) error {
	logrus.Debugf("Running function for all %d clusters", n)

	wg := &sync.WaitGroup{}
	wg.Add(int(n))
	foundErr := false
	for i := ClusterNumber(1); i <= ClusterNumber(n); i++ {
		go func(j ClusterNumber) {
			logrus.Tracef("ForCluster goroutine with index %s starting...", j)
			clusterInfo := NewClusterInfo(cfg, j)
			if err := fn(clusterInfo); err != nil {
				clusterInfo.Logger.Error(err)
				foundErr = true
			}
			logrus.Tracef("ForCluster goroutine with index %s is done", j)
			wg.Done()
		}(i)
	}
	wg.Wait()
	if foundErr {
		return fmt.Errorf("an error occured previously")
	}
	return nil
}
