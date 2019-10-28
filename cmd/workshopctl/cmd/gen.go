package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/luxas/workshopctl/pkg/config"
	"github.com/luxas/workshopctl/pkg/gen"
	"github.com/sirupsen/logrus"
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
	fs.StringVarP(&cfg.Domain, "domain", "d", "workshopctl.kubernetesfinland.com", "What domain to use")
	fs.StringVarP(&cfg.GitRepo, "git-repo", "r", "https://github.com/luxas/workshopctl", "What git repo to use")
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

			return config.ForCluster(cfg.Clusters, func(i config.ClusterNumber, logger *logrus.Entry) error {
				log.Infof("Cluster %s...", i)
				for _, chart := range charts {
					logger.Infof("  Generating chart %q...", chart.Name)
					if err := gen.GenerateChart(chart, i, cfg); err != nil {
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
