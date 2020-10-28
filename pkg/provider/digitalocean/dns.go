package digitalocean

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/config/keyval"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/gen"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	"github.com/digitalocean/godo"
	"github.com/sirupsen/logrus"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewDigitalOceanDNSProvider(ctx context.Context, p *config.Provider, rootDomain string) (provider.DNSProvider, error) {
	return &DigitalOceanDNSProvider{
		doCommon:   initCommon(ctx, p),
		rootDomain: rootDomain,
	}, nil
}

type DigitalOceanDNSProvider struct {
	doCommon
	rootDomain string
}

func (do *DigitalOceanDNSProvider) ChartProcessors() []gen.Processor {
	return []gen.Processor{&dnsProcessor{}}
}

func (do *DigitalOceanDNSProvider) ValuesProcessors() []gen.Processor {
	return nil
}

func (do *DigitalOceanDNSProvider) CleanupRecords(ctx context.Context, m provider.ClusterMeta) error {
	logger := util.Logger(ctx)

	subdomain := m.Index.Subdomain()
	logger.Debugf("Asking for records for domain %s and sub-domain %s", do.rootDomain, subdomain)
	// List all records for domain
	records, _, err := do.c.Domains.Records(ctx, do.rootDomain, &godo.ListOptions{})
	if err != nil {
		return err
	}

	for _, record := range records {
		logger.Debugf("Observed record: %s", record)
		// Skip records that aren't associated with the given subdomain
		// TODO: Maybe be even more restrictive/specific about what to delete
		// e.g. look at heritage=external-dns fields, or only delete A/TXT records.
		if !strings.HasSuffix(record.Name, subdomain) {
			logger.Debugf("Skipped record: %s", record)
			continue
		}
		// Delete records that are related to this subdomain
		if err := do.deleteRecord(ctx, &record, logger); err != nil {
			return err
		}
	}
	return nil
}

func (do *DigitalOceanDNSProvider) deleteRecord(ctx context.Context, record *godo.DomainRecord, logger *logrus.Entry) error {
	recordStr := do.recordStr(record)
	if do.dryRun {
		logger.Infof("Would delete record: %s", recordStr)
		return nil
	}
	logger.Infof("Deleting record: %s", recordStr)
	_, err := do.c.Domains.DeleteRecord(ctx, do.rootDomain, record.ID)
	return err
}

func (do *DigitalOceanDNSProvider) recordStr(record *godo.DomainRecord) string {
	return fmt.Sprintf("%s %s.%s: %s", record.Type, record.Name, do.rootDomain, record.Data)
}

var (
	externalDNSEnvValue = kyaml.MustParse(`
- name: DO_TOKEN
  valueFrom:
    secretKeyRef:
      name: workshopctl
      key: DNS_PROVIDER_SERVICEACCOUNT
`)

	traefikDNSEnvValue = kyaml.MustParse(`
- name: DO_AUTH_TOKEN
  valueFrom:
    secretKeyRef:
      name: workshopctl
      key: DNS_PROVIDER_SERVICEACCOUNT
`)
)

type dnsProcessor struct{}

func (pr *dnsProcessor) Process(ctx context.Context, cd *gen.ChartData, p *keyval.Parameters, r io.Reader, w io.Writer) error {
	return util.KYAMLFilter(r, w, util.KYAMLFilterFunc(
		func(node *kyaml.RNode) (*kyaml.RNode, error) {
			return node, util.KYAMLResourceMetaMatcher(node, util.KYAMLResourceMetaMatch{
				Kind:      "Deployment",
				Name:      "traefik",
				Namespace: constants.WorkshopctlNamespace,
				Func: func() error {
					return node.PipeE(
						kyaml.LookupCreate(kyaml.SequenceNode, "spec", "template", "spec", "containers", "[name=traefik]", "env"),
						kyaml.Append(traefikDNSEnvValue.YNode().Content...))
				},
			}, util.KYAMLResourceMetaMatch{
				Kind:      "Deployment",
				Name:      "external-dns",
				Namespace: constants.WorkshopctlNamespace,
				Func: func() error {
					return node.PipeE(
						kyaml.LookupCreate(kyaml.SequenceNode, "spec", "template", "spec", "containers", "[name=external-dns]", "env"),
						kyaml.Append(externalDNSEnvValue.YNode().Content...))
				},
			})
		},
	))
}
