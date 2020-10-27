package cmd

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/gen"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider/providers"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var genSkipLocalCharts = false

// NewGenCommand returns the "gen" command
func NewGenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate a set of manifests based on the configuration",
		Run:   RunGen,
	}

	addGenFlags(cmd.Flags())
	return cmd
}

func addGenFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&genSkipLocalCharts, "skip-local-charts", genSkipLocalCharts, "Don't consider the local directory's charts/ directory")
}

func loadConfig() (*config.Config, error) {
	cfg := &config.Config{}
	if err := util.ReadYAMLFile(configPathFlag, cfg); err != nil {
		return nil, err
	}
	if err := initConfig(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func RunGen(cmd *cobra.Command, args []string) {
	err := func() error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		charts, err := gen.SetupInternalChartCache(cfg.RootDir)
		if err != nil {
			return err
		}

		if !genSkipLocalCharts {
			chartsDir := filepath.Join(cfg.RootDir, constants.ChartsDir)
			chartInfos, err := ioutil.ReadDir(chartsDir)
			if err != nil {
				return err
			}
			for _, chartInfo := range chartInfos {
				if !chartInfo.IsDir() {
					continue
				}
				chart, err := gen.SetupExternalChartCache(cfg.RootDir, chartInfo.Name())
				if err != nil {
					return err
				}
				charts = append(charts, chart)
			}
		}

		dnsProvider, ok := providers.DNSProviders[cfg.DNSProvider.Name]
		if !ok {
			return fmt.Errorf("didn't find dns provider %s", cfg.DNSProvider.Name)
		}

		return config.ForCluster(cfg.Clusters, cfg, func(clusterInfo *config.ClusterInfo) error {
			for _, chart := range charts {
				clusterInfo.Logger.Infof("Generating chart %q...", chart.Name)
				if err := gen.GenerateChart(chart, clusterInfo, dnsProvider.ValuesProcessors(), dnsProvider.ChartProcessors()); err != nil {
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
