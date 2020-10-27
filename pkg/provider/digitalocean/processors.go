package digitalocean

import (
	"io"

	"github.com/cloud-native-nordics/workshopctl/pkg/config/keyval"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/gen"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

type DigitalOceanDNSProvider struct{}

func (do *DigitalOceanDNSProvider) ChartProcessors() []gen.Processor {
	return []gen.Processor{&dnsProcessor{}}
}

func (do *DigitalOceanDNSProvider) ValuesProcessors() []gen.Processor {
	return nil
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

func (pr *dnsProcessor) Process(cd *gen.ChartData, p *keyval.Parameters, r io.Reader, w io.Writer) error {
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
