package config

type Config struct {
	Domain   string `json:"domain"`
	Clusters uint16 `json:"clusters"`
	GitRepo  string `json:"gitRepo"`
	RootDir  string `json:"-"`

	Provider       string `json:"provider"`
	ServiceAccount string `json:"serviceAccount"`
	CPUs           uint16 `json:"cpus"`
	RAM            uint16 `json:"ram"`
	NodeCount      uint16 `json:"nodeCount"`
}
