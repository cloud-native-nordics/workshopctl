package keyval

import (
	"encoding/json"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
)

var externalDNSMap = map[string]string{
	"digitalocean": "digitalocean",
	"gke":          "google",
	"scaleway":     "scaleway",
	"aws":          "aws",
	"cloudflare":   "cloudflare",
}

var traefikDNSMap = map[string]string{
	"digitalocean": "digitalocean",
	"gke":          "gcloud",
	"scaleway":     "scaleway",
	"aws":          "route53",
	"cloudflare":   "cloudflare",
}

func FromClusterInfo(cfg *config.ClusterInfo) *Parameters {
	return &Parameters{
		WorkshopctlParameters: WorkshopctlParameters{
			CloudProvider:               cfg.CloudProvider.Name,
			CloudProviderServiceAccount: cfg.CloudProvider.InternalToken,
			CloudProviderSpecific:       cfg.CloudProvider.ProviderSpecific,

			ExternalDNSProvider:       externalDNSMap[cfg.DNSProvider.Name],
			TraefikDNSProvider:        traefikDNSMap[cfg.DNSProvider.Name],
			DNSProviderServiceAccount: cfg.DNSProvider.InternalToken,
			DNSProviderSpecific:       cfg.DNSProvider.ProviderSpecific,

			RootDomain:    cfg.RootDomain,
			ClusterDomain: cfg.Domain(),

			TutorialsRepo: cfg.Tutorials.GitRepo,
			TutorialsDir:  cfg.Tutorials.Dir,

			LetsEncryptEmail: cfg.LetsEncryptEmail,

			ClusterPassword:  cfg.Password,
			ClusterBasicAuth: cfg.BasicAuth(),
		},
	}
}

type Parameters struct {
	WorkshopctlParameters `json:"workshopctl"`
}

type WorkshopctlParameters struct {
	CloudProvider               string            `json:"CLOUD_PROVIDER"`
	CloudProviderServiceAccount string            `json:"CLOUD_PROVIDER_SERVICEACCOUNT"`
	CloudProviderSpecific       map[string]string `json:"-"`

	ExternalDNSProvider       string            `json:"EXTERNAL_DNS_PROVIDER"`
	TraefikDNSProvider        string            `json:"TRAEFIK_DNS_PROVIDER"`
	DNSProviderServiceAccount string            `json:"DNS_PROVIDER_SERVICEACCOUNT"`
	DNSProviderSpecific       map[string]string `json:"-"`

	RootDomain    string `json:"ROOT_DOMAIN"`
	ClusterDomain string `json:"CLUSTER_DOMAIN"`

	TutorialsRepo string `json:"TUTORIALS_REPO"`
	TutorialsDir  string `json:"TUTORIALS_DIR"`

	LetsEncryptEmail string `json:"LETSENCRYPT_EMAIL"`

	ClusterPassword  string `json:"CLUSTER_PASSWORD"`
	ClusterBasicAuth string `json:"CLUSTER_BASIC_AUTH_BCRYPT"`
}

func (p *Parameters) ToMap() map[string]string {
	b, _ := json.Marshal(p.WorkshopctlParameters)
	m := map[string]string{}
	_ = json.Unmarshal(b, &m)
	for k, v := range p.CloudProviderSpecific {
		m[k] = v
	}
	// TODO: handle conflicts?
	for k, v := range p.DNSProviderSpecific {
		m[k] = v
	}
	return m
}
