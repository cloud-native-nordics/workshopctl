package providers

import (
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider/digitalocean"
)

var CloudProviders = map[string]provider.CloudProviderFunc{
	"digitalocean": digitalocean.NewDigitalOceanCloudProvider,
}

var DNSProviders = map[string]provider.DNSProvider{
	"digitalocean": &digitalocean.DigitalOceanDNSProvider{},
}
