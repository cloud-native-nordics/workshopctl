package config

import (
	"bytes"
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
	// The prefix to use for all identifying names/tags/etc.
	// This allows an user to have multiple workshop environments at once in the same provider
	Name string `json:"name"`

	// CloudProvider specifies what cloud provider to use and how to authenticate with it.
	CloudProvider Provider `json:"cloudProvider"`
	// DNSProvider specifies what dns provider to use and how to authenticate with it.
	DNSProvider Provider `json:"dnsProvider"`

	RootDomain string `json:"rootDomain"`
	// How many clusters should be created?
	Clusters uint16 `json:"clusters"`
	// Where to store the manifests for collaboration?
	Git Git `json:"git"`

	// If this is specified you can use "sealed secrets"
	// TODO: Implement this with the help of Mozilla SOPS
	// GPGKeyID string `json:"gpgKeyID"`

	// Whom to contact by Let's Encrypt
	LetsEncryptEmail string `json:"letsEncryptEmail"`

	Tutorials Tutorials `json:"tutorials"`

	ClusterLogin ClusterLogin `json:"clusterLogin"`

	NodeGroups []NodeGroup `json:"nodeGroups"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name must not be empty")
	}
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
		return fmt.Errorf("Let's Encrypt email must not be empty")
	}
	if c.Git.Repo == "" {
		return fmt.Errorf("must specify backing git repo")
	}
	if c.Git.ServiceAccountPath == "" {
		return fmt.Errorf("must specify git provider token")
	}
	return nil
}

func (c *Config) Complete(ctx context.Context) error {
	// First validate the struct
	if err := c.Validate(); err != nil {
		return err
	}
	if c.CloudProvider.Name == "" {
		c.CloudProvider.Name = "digitalocean"
	}
	if c.DNSProvider.Name == "" {
		c.DNSProvider.Name = "digitalocean"
	}
	if c.Clusters == 0 {
		c.Clusters = 1
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
		saPath := util.JoinPaths(ctx, c.CloudProvider.ServiceAccountPath)
		if err := readFileInto(saPath, &c.CloudProvider.ServiceAccountContent); err != nil {
			return err
		}
	}
	if c.DNSProvider.ServiceAccountPath != "" {
		saPath := util.JoinPaths(ctx, c.DNSProvider.ServiceAccountPath)
		if err := readFileInto(saPath, &c.DNSProvider.ServiceAccountContent); err != nil {
			return err
		}
	}
	if c.Git.ServiceAccountPath != "" {
		saPath := util.JoinPaths(ctx, c.Git.ServiceAccountPath)
		if err := readFileInto(saPath, &c.Git.ServiceAccountContent); err != nil {
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
	// Parse the git URL
	// TODO: This should live in go-git-providers
	u, err := giturls.Parse(c.Git.Repo)
	if err != nil {
		return err
	}
	paths := strings.Split(u.Path, "/")
	c.Git.RepoStruct = gitprovider.UserRepositoryRef{
		UserRef: gitprovider.UserRef{
			Domain:    u.Host,
			UserLogin: paths[0],
		},
		RepositoryName: strings.TrimSuffix(paths[1], ".git"),
	}
	return nil
}

type ServiceAccount struct {
	// ServiceAccountPath specifies the file path to the service account
	ServiceAccountPath string `json:"serviceAccountPath"`
	// The contents of ServiceAccountPath, read at runtime and never marshalled.
	ServiceAccountContent string `json:"-"`
}

// If the ServiceAccount is an oauth2 token, this helper method might be useful for
// the implementing provider
func (sa ServiceAccount) TokenSource() oauth2.TokenSource {
	return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: sa.ServiceAccountContent})
}

type Provider struct {
	// Name of the provider. For now, only "digitalocean" is supported.
	Name string `json:"name"`
	// The ServiceAccount struct is embedded and inlined into the provider
	ServiceAccount `json:",inline"`
	// Provider-specific data
	ProviderSpecific map[string]string `json:"providerSpecific,omitempty"`
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

type Git struct {
	// Repo specifies where the "infra" git repo should be
	Repo       string                        `json:"repo"`
	RepoStruct gitprovider.UserRepositoryRef `json:"-"`

	// The ServiceAccount struct is embedded and inlined into this struct
	ServiceAccount `json:",inline"`
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

func NewClusterInfo(ctx context.Context, cfg *Config, i ClusterNumber) *ClusterInfo {
	pass := cfg.ClusterLogin.CommonPassword
	if cfg.ClusterLogin.UniquePasswords {
		var err error
		pass, err = util.RandomSHA(4) // TODO: constant
		if err != nil {
			panic(err)
		}
		// Warn about possible misconfigurations
		if len(cfg.ClusterLogin.CommonPassword) != 0 {
			util.Logger(ctx).Warnf("You have specified both .ClusterLogin.UniquePasswords and .ClusterLogin.CommonPassword. UniquePasswords has higher priority and hence CommonPassword is ignored.")
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

	// mutex shared by cluster threads when they need to coordinate
	// TODO: This is limited to only one lock operation, consider supporting more in the future
	mux := &sync.Mutex{}
	for i := ClusterNumber(1); i <= ClusterNumber(n); i++ {
		go func(j ClusterNumber) {
			clusterCtx := util.WithClusterNumber(ctx, uint16(j))
			clusterCtx = util.WithMutex(clusterCtx, mux)
			logger := util.Logger(clusterCtx)
			logger.Tracef("ForCluster goroutine starting...")
			clusterInfo := NewClusterInfo(clusterCtx, cfg, j)
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
	*target = string(bytes.TrimSpace(b))
	return nil
}
