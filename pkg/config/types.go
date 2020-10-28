package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/sirupsen/logrus"
	giturls "github.com/whilp/git-urls"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

type Config struct {
	RootDir string `json:"-"`

	// CloudProvider specifies what cloud provider to use and how to authenticate with it.
	CloudProvider Provider `json:"cloudProvider"`
	// DNSProvider specifies what dns provider to use and how to authenticate with it.
	// If nil, CloudProvider is used.
	DNSProvider *Provider `json:"dnsProvider"`

	RootDomain    string                        `json:"rootDomain"`
	Clusters      uint16                        `json:"clusters"`
	GitRepo       string                        `json:"gitRepo"`
	GitRepoStruct gitprovider.UserRepositoryRef `json:"-"`

	// If this is specified you can use "sealed secrets"
	GPGKeyID string `json:"gpgKeyID"`

	// Whom to contact by Let's Encrypt
	LetsEncryptEmail string `json:"letsEncryptEmail"`

	Tutorials Tutorials `json:"tutorials"`

	ClusterLogin ClusterLogin `json:"clusterLogin"`

	NodeGroups []NodeGroup `json:"nodeGroups"`
}

func (c *Config) Validate() error {
	if c.CloudProvider.ServiceAccountPath == "" {
		return fmt.Errorf("must specify cloud provider SA path")
	}
	if c.DNSProvider.ServiceAccountPath == "" {
		return fmt.Errorf("must specify DNS provider SA path")
	}
	if c.RootDomain == "" {
		return fmt.Errorf("root domain must not be empty")
	}
	if c.LetsEncryptEmail == "" {
		return fmt.Errorf("lets encrypt email must not be empty")
	}
	return nil
}

func (c *Config) Complete(ctx context.Context) error {
	if c.CloudProvider.Name == "" {
		c.CloudProvider.Name = "digitalocean"
	}
	if c.Clusters == 0 {
		c.Clusters = 1
	}
	if c.DNSProvider == nil {
		c.DNSProvider = &c.CloudProvider
	}
	if c.ClusterLogin.Username == "" {
		c.ClusterLogin.Username = "workshopctl"
	}
	if c.ClusterLogin.CommonPassword == "" {
		pass, err := util.RandomSHA(4)
		if err != nil {
			return err
		}
		// TODO: This maybe shouldn't "leak" to the config file when marshalling?
		c.ClusterLogin.CommonPassword = pass
	}
	if c.CloudProvider.ServiceAccountPath != "" {
		if err := readFileInto(c.CloudProvider.ServiceAccountPath, &c.CloudProvider.InternalToken); err != nil {
			return err
		}
	}
	if c.CloudProvider.ServiceAccountPath != "" {
		if err := readFileInto(c.DNSProvider.ServiceAccountPath, &c.DNSProvider.InternalToken); err != nil {
			return err
		}
	}
	if c.NodeGroups == nil {
		c.NodeGroups = []NodeGroup{
			{
				Instances: 1,
				NodeClaim: NodeClaim{
					CPU:       2,
					RAM:       4,
					Dedicated: false,
				},
			},
		}
	}
	if c.GitRepo == "" {
		origin, _, err := util.ShellCommand(ctx, `git -C %s remote -v | grep push | grep origin | awk '{print $2}'`, c.RootDir).Run()
		if err != nil {
			return err
		}
		c.GitRepo = origin
	}
	u, err := giturls.Parse(c.GitRepo)
	if err != nil {
		return err
	}
	paths := strings.Split(u.Path, "/")
	c.GitRepoStruct = gitprovider.UserRepositoryRef{
		UserRef: gitprovider.UserRef{
			Domain:    u.Host,
			UserLogin: paths[0],
		},
		RepositoryName: paths[1],
	}
	return nil
}

type Provider struct {
	// Name of the provider. For now, only "digitalocean" is supported.
	Name string `json:"name"`
	// ServiceAccountPath specifies the file path to the service account
	ServiceAccountPath string `json:"serviceAccountPath"`

	ProviderSpecific map[string]string `json:"providerSpecific,omitempty"`

	InternalToken string `json:"-"`
}

func (p *Provider) TokenSource() oauth2.TokenSource {
	return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: p.InternalToken})
}

type NodeGroup struct {
	Instances uint16    `json:"instances"`
	NodeClaim NodeClaim `json:"nodeClaim"`
}

type NodeClaim struct {
	CPU uint16 `json:"cpus"`
	RAM uint16 `json:"ram"`
	// Refers to if the CPU is shared with other tenants, or dedicated for this VM
	Dedicated bool `json:"dedicated"`
}

type ClusterLogin struct {
	// Username for basic auth logins. Defaults to workshopctl.
	Username string `json:"username"`
	// CommonPassword sets the same password for VS code and all basic auth
	// for all clusters. If unset, a random password will be generated.
	CommonPassword string `json:"commonPassword"`
	// UniquePasswords tells whether every cluster should have its own password.
	// By default false, which means all clusters share CommonPassword. If true,
	// CommonPassword will be ignored and all clusters' passwords will be generated.
	UniquePasswords bool `json:"uniquePasswords"`
}

type Tutorials struct {
	Repo string `json:"repo"`
	Dir  string `json:"dir"`
}

type ClusterInfo struct {
	*Config
	Index    ClusterNumber
	Password string
}

func NewClusterInfo(cfg *Config, i ClusterNumber) *ClusterInfo {
	pass := cfg.ClusterLogin.CommonPassword
	if cfg.ClusterLogin.UniquePasswords {
		var err error
		pass, err = util.RandomSHA(4) // TODO: constant
		if err != nil {
			panic(err)
		}
	}
	return &ClusterInfo{cfg, i, pass}
}

func (c *ClusterInfo) Domain() string {
	return c.Index.Domain(c.RootDomain)
}

func (c *ClusterInfo) BasicAuth() string {
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s:%s", c.ClusterLogin.Username, hash)
}

var _ fmt.Stringer = ClusterNumber(0)

type ClusterNumber uint16

func (n ClusterNumber) String() string {
	return fmt.Sprintf("%02d", n)
}

func (n ClusterNumber) Subdomain() string {
	return fmt.Sprintf("cluster-%s", n)
}

func (n ClusterNumber) Domain(rootDomain string) string {
	return fmt.Sprintf("%s.%s", n.Subdomain(), rootDomain)
}

func (n ClusterNumber) ClusterDir() string {
	return filepath.Join(constants.ClustersDir, n.String())
}

func (n ClusterNumber) KubeConfigPath() string {
	return filepath.Join(n.ClusterDir(), constants.KubeconfigFile)
}

func ForCluster(ctx context.Context, n uint16, cfg *Config, fn func(context.Context, *ClusterInfo) error) error {
	logrus.Debugf("Running function for all %d clusters", n)

	wg := &sync.WaitGroup{}
	wg.Add(int(n))
	foundErr := false
	for i := ClusterNumber(1); i <= ClusterNumber(n); i++ {
		go func(j ClusterNumber) {
			clusterCtx := util.WithClusterNumber(ctx, uint16(j))
			logger := util.Logger(clusterCtx)
			logger.Tracef("ForCluster goroutine starting...")
			clusterInfo := NewClusterInfo(cfg, j)
			if err := fn(clusterCtx, clusterInfo); err != nil {
				logger.Error(err)
				foundErr = true
			}
			logger.Tracef("ForCluster goroutine is done")
			wg.Done()
		}(i)
	}
	wg.Wait()
	if foundErr {
		return fmt.Errorf("an error occured previously")
	}
	return nil
}

func readFileInto(file string, target *string) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	*target = string(b)
	return nil
}
