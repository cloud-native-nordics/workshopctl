package constants

const (
	// Top-level directories, i.e. ./
	ChartsDir   = "charts"
	ClustersDir = "clusters"
	CacheDir    = ".cache"

	// Under ./{ChartsDir}/<chart>/
	// Helm-specific
	TemplatesDir = "templates"
	ValuesYAML   = "values.yaml"
	ChartYAML    = "Chart.yaml"
	// workshopctl "extensions"
	NamespaceFile     = "namespace"
	ExternalChartFile = "external-chart"
	// jq "extensions"
	PipeJS   = "pipe.js"
	ValuesJS = "values.js"

	// The default namespace in k8s is called "default"
	DefaultNamespace     = "default"
	WorkshopctlNamespace = "workshopctl"

	WorkshopctlSecret = "workshopctl"
)

var KnownChartFiles = []string{
	// Helm "classic" files
	TemplatesDir,
	ChartYAML,
	ValuesYAML,

	// workshopctl-specific files
	NamespaceFile,
	ExternalChartFile,
	// jq-specific files
	PipeJS,
	ValuesJS,
}
