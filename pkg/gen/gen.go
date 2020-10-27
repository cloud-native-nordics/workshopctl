package gen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-native-nordics/workshopctl/pkg/charts"
	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/config/keyval"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

type ChartData struct {
	Name        string
	CacheDir    string
	CopiedFiles map[string]string
}

type Processor interface {
	Process(ctx context.Context, cd *ChartData, p *keyval.Parameters, r io.Reader, w io.Writer) error
}

func SetupInternalChartCache(ctx context.Context, rootPath string) ([]*ChartData, error) {
	// Restore built-in charts/* to .cache/*
	if err := charts.RestoreAssets(filepath.Join(rootPath, constants.CacheDir), ""); err != nil {
		return nil, err
	}
	// List internal chart names
	charts, err := charts.AssetDir("")
	if err != nil {
		return nil, err
	}
	// Now that the internal files are extracted to disk,
	// process them exactly as normal "external" charts
	chartCache := make([]*ChartData, 0, len(charts))
	for _, chart := range charts {
		cd, err := SetupExternalChartCache(ctx, rootPath, chart)
		if err != nil {
			return nil, err
		}
		chartCache = append(chartCache, cd)
	}
	return chartCache, nil
}

func SetupExternalChartCache(ctx context.Context, rootPath, chartName string) (*ChartData, error) {
	cd := &ChartData{
		CacheDir:    filepath.Join(rootPath, constants.CacheDir, chartName),
		Name:        chartName,
		CopiedFiles: map[string]string{},
	}

	// Create the .cache directory for the chart
	if err := os.MkdirAll(cd.CacheDir, 0755); err != nil {
		return nil, err
	}

	chartDir := filepath.Join(rootPath, constants.ChartsDir, chartName)
	for _, f := range constants.KnownChartFiles {
		from := filepath.Join(chartDir, f)
		to := filepath.Join(cd.CacheDir, f)

		fromExists, _ := util.PathExists(from)
		toExists, _ := util.PathExists(to)
		if !fromExists && !toExists {
			continue // nothing to do
		}
		if fromExists { // if from exists, always copy to make sure to is up-to-date
			if err := util.Copy(from, to); err != nil {
				return nil, err
			}
		}
		// if to exists, but not from, just proceed and register to
		cd.CopiedFiles[f] = to
	}

	// Download the chart if it's explicitely said to be external
	if externalChartFile, ok := cd.CopiedFiles[constants.ExternalChartFile]; ok {
		if err := downloadChart(ctx, rootPath, externalChartFile); err != nil {
			return nil, err
		}
	}

	return cd, nil
}

func downloadChart(ctx context.Context, rootPath, externalChartFile string) error {
	// Read contents of the external-chart file
	b, err := ioutil.ReadFile(externalChartFile)
	if err != nil {
		return err
	}
	externalChart := string(b)

	// Expecting something like:
	// "stable/kubernetes-dashboard"
	// "https://charts.fluxcd.io/flux"
	u, err := url.Parse(externalChart)
	if err != nil {
		return err
	}
	if len(u.Scheme) > 0 {
		// Remove the last path element from the URL; that's the name of the chart
		cname := filepath.Base(u.Path)
		u.Path = filepath.Dir(u.Path)
		// Replace dots with dashes in order to craft the name of the repo
		crepo := strings.ReplaceAll(u.Host, ".", "-")
		// The chart name is "${repo}/${name}"
		externalChart = filepath.Join(crepo, cname)

		// Make sure the repo is registered correctly
		out, _, err := util.Command(ctx, "helm", "repo", "list").Run()
		if err != nil {
			return err
		}
		// Only add the repo if it doesn't already exist
		if !strings.Contains(out, crepo) {
			log.Infof("Adding a new helm repo called %q pointing to %q", crepo, u.String())
			_, _, err = util.Command(ctx, "helm", "repo", "add", crepo, u.String()).Run()
			if err != nil {
				return err
			}
		}
	} else {
		arr := strings.Split(externalChart, "/")
		if len(arr) != 2 {
			return fmt.Errorf("invalid format of %q: %q. Should be either {stable,test}/{name} or {repo-url}/{name}", constants.ExternalChartFile, externalChart)
		}
	}

	log.Infof("Found external chart to download %q", externalChart)
	// This extracts the chart to e.g. .cache/kubernetes-dashboard/{Chart.yaml,values.yaml,templates}
	cacheDir := filepath.Join(rootPath, constants.CacheDir)
	_, _, err = util.Command(ctx, "helm", "fetch", externalChart, "--untar", "--untardir", cacheDir).Run()
	return err
}

func GenerateChart(ctx context.Context, cd *ChartData, clusterInfo *config.ClusterInfo, valuesProcessors, chartProcessors []Processor) error {
	logger := util.Logger(ctx)

	namespace := constants.DefaultNamespace
	if nsFile, ok := cd.CopiedFiles[constants.NamespaceFile]; ok {
		b, err := ioutil.ReadFile(nsFile)
		if err != nil {
			return err
		}
		namespace = string(b)
	}
	// 1. Read values.yaml, if exists, otherwise start with an empty buffer as the first io.Reader
	// 2. Attach the addParamsToValues{} values processor which adds the parameters as needed
	// 3. Invoke other values processors as needed in a chain
	// 4. Run "helm template -n %s workshopctl chart -f -" with values as stdin
	// 5. Invoke other chart processors, but always the \{\{ => {{ one
	// 6. Write output to ./clusters/001/<name>.yaml

	processorChain := []Processor{
		&addParamsToValues{},
	}
	processorChain = append(processorChain, valuesProcessors...)
	processorChain = append(processorChain, []Processor{
		&helmTemplateProcessor{namespace},
		&unescapeGoTmpls{},
	}...)
	processorChain = append(processorChain, chartProcessors...)

	p := keyval.FromClusterInfo(clusterInfo)

	var initialData []byte
	if valuesFile, ok := cd.CopiedFiles[constants.ValuesYAML]; ok {
		var err error
		initialData, err = ioutil.ReadFile(valuesFile)
		if err != nil {
			return err
		}
	}

	input := bytes.NewBuffer(initialData)
	output := new(bytes.Buffer)
	for i, processor := range processorChain {
		logger.Tracef("Before processor %d: %s", i, input.String())
		if err := processor.Process(ctx, cd, p, input, output); err != nil {
			logger.Errorf("error: %v, output: %s", err, output.String())
			return err
		}
		logger.Tracef("After processor %d: %s", i, output.String())
		// Reset the input array, that is no longer needed
		input.Reset()
		// Now we can set the output pointer to be the next input, and the reset output to be an
		// empty buffer but with pre-created capacity
		var tmp = input
		input = output
		output = tmp
	}

	outputFile := filepath.Join(clusterInfo.RootDir, constants.ClustersDir, clusterInfo.Index.String(), fmt.Sprintf("%s.yaml", cd.Name))
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, input)
	return err
}

type addParamsToValues struct{}

func (pr *addParamsToValues) Process(ctx context.Context, _ *ChartData, p *keyval.Parameters, r io.Reader, w io.Writer) error {
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

type helmTemplateProcessor struct {
	namespace string
}

func (pr *helmTemplateProcessor) Process(ctx context.Context, cd *ChartData, _ *keyval.Parameters, r io.Reader, w io.Writer) error {
	_, _, err := util.ShellCommand(ctx, `helm template -n %s workshopctl . -f -`, pr.namespace).
		WithStdio(r, w, nil).
		WithPwd(cd.CacheDir).
		Run()
	return err
}

type unescapeGoTmpls struct{}

func (pr *unescapeGoTmpls) Process(ctx context.Context, _ *ChartData, _ *keyval.Parameters, r io.Reader, w io.Writer) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	b = bytes.ReplaceAll(b, []byte(`\{`), []byte(`{`))
	b = bytes.ReplaceAll(b, []byte(`\}`), []byte(`}`))

	_, err = w.Write(b)
	return err
}
