package constants

import "fmt"

const (
	// Top-level directories, i.e. ./
	ChartsDir   = "charts"
	ClustersDir = "clusters"
	CacheDir    = ".cache"

	// Under ./{ChartsDir}/<chart>/
	// Helm-specific
	TemplatesDir = "templates"
	ChartYAML    = "Chart.yaml"
	// workshopctl "extensions"
	NamespaceFile      = "namespace"
	ExternalChartFile  = "external-chart"
	ValuesOverrideYAML = "values-override.yaml"
	// jq "extensions"
	PipeJS   = "pipe.js"
	ValuesJS = "values.js"

	// Under ./{ClustersDir}/<cluster>/
	KubeconfigFile = ".kubeconfig"

	// The default namespace in k8s is called "default"
	DefaultNamespace     = "default"
	WorkshopctlNamespace = "workshopctl"

	WorkshopctlSecret = "workshopctl"
)

func ClusterName(namePrefix string, index fmt.Stringer) string {
	return fmt.Sprintf("workshopctl-%s-%s", namePrefix, index)
}

// These files will be copied from ./charts/<chart>/<file> to ./.cache/<chart>/<file>
var KnownChartFiles = []string{
	// Helm "classic" files
	TemplatesDir,
	ChartYAML,
	// TODO: Include the "classic", non-templated, base values.yaml here too.

	// workshopctl-specific files
	NamespaceFile,
	ExternalChartFile,
	ValuesOverrideYAML,
	// jq-specific files
	PipeJS,
	ValuesJS,
}
