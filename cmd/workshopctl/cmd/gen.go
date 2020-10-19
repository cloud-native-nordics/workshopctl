package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/gen"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewGenCommand returns the "gen" command
func NewGenCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cfg := &config.Config{}
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate a set of manifests based on the configuration",
		Run:   RunGen(cfg),
	}

	addGenFlags(cmd.Flags(), cfg)
	return cmd
}

func addGenFlags(fs *pflag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Provider, "provider", "digitalocean", "What provider to use")
	fs.Uint16VarP(&cfg.Clusters, "clusters", "c", 1, "How many clusters to create")
	fs.StringVarP(&cfg.RootDomain, "root-domain", "d", "workshopctl.kubernetesfinland.com", "What domain to use")
	fs.StringVarP(&cfg.GitRepo, "git-repo", "r", "https://github.com/cloud-native-nordics/workshopctl", "What git repo to use")
	fs.StringVar(&cfg.RootDir, "root-dir", ".", "Where the workshopctl directory is")
}

func RunGen(cfg *config.Config) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		err := func() error {
			// Resolve relative paths to absolute ones
			if !filepath.IsAbs(cfg.RootDir) {
				pwd, _ := os.Getwd()
				cfg.RootDir = filepath.Join(pwd, cfg.RootDir)
			}
			manifestDir := filepath.Join(cfg.RootDir, "manifests")
			chartInfos, err := ioutil.ReadDir(manifestDir)
			if err != nil {
				return err
			}
			charts := make([]*gen.ChartData, 0, len(chartInfos))
			for _, chartInfo := range chartInfos {
				if !chartInfo.IsDir() {
					continue
				}
				chart, err := gen.SetupChartCache(cfg.RootDir, chartInfo.Name())
				if err != nil {
					return err
				}
				charts = append(charts, chart)
			}

			return config.ForCluster(cfg.Clusters, cfg, func(clusterInfo *config.ClusterInfo) error {
				for _, chart := range charts {
					clusterInfo.Logger.Infof("Generating chart %q...", chart.Name)
					if err := gen.GenerateChart(chart, clusterInfo); err != nil {
						return err
					}
				}
				return nil
			})
		}()
		if err != nil {
			log.Fatal(err)
		}
	}
}
