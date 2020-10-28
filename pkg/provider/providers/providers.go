package providers

import (
	"context"
	"fmt"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider/digitalocean"
)

func CloudProviders() provider.CloudProviderFactory {
	return providers
}

func DNSProviders() provider.DNSProviderFactory {
	return providers
}

var providers = providersImpl{}

type cloudFunc func(ctx context.Context, p *config.Provider) (provider.CloudProvider, error)

type dnsFunc func(ctx context.Context, p *config.Provider, rootDomain string) (provider.DNSProvider, error)

var cloudProviders = map[string]cloudFunc{
	"digitalocean": digitalocean.NewDigitalOceanCloudProvider,
}

var dnsProviders = map[string]dnsFunc{
	"digitalocean": digitalocean.NewDigitalOceanDNSProvider,
}

type providersImpl struct{}

func (providersImpl) NewCloudProvider(ctx context.Context, p *config.Provider) (provider.CloudProvider, error) {
	fn, ok := cloudProviders[p.Name]
	if !ok {
		return nil, fmt.Errorf("cloud provider %s not supported", p.Name)
	}
	return fn(ctx, p)
}

func (providersImpl) NewDNSProvider(ctx context.Context, p *config.Provider, rootDomain string) (provider.DNSProvider, error) {
	fn, ok := dnsProviders[p.Name]
	if !ok {
		return nil, fmt.Errorf("DNS provider %s not supported", p.Name)
	}
	return fn(ctx, p, rootDomain)
}
