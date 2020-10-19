package gen

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	pipeJS     = "pipe.js"
	valuesJS   = "values.js"
	valuesYAML = "values.yaml"
	chartDir   = "chart"
	cacheDir   = ".cache"
)

type ChartData struct {
	Name        string
	ManifestDir string
	CacheDir    string
	CopiedFiles map[string]struct{}
}

func SetupChartCache(rootPath, chartName string) (*ChartData, error) {
	cd := &ChartData{
		CacheDir:    filepath.Join(rootPath, cacheDir, chartName),
		ManifestDir: filepath.Join(rootPath, "manifests", chartName),
		Name:        chartName,
		CopiedFiles: map[string]struct{}{},
	}

	// Create the .cache directory for the chart
	if err := os.MkdirAll(cd.CacheDir, 0755); err != nil {
		return nil, err
	}

	// Create the node_modules symlink if it doesn't exist
	log.Debugf("Symlinking node_modules...")
	nmPath := filepath.Join(cd.CacheDir, "node_modules")
	if exists, _ := util.PathExists(nmPath); !exists {
		if err := os.Symlink(filepath.Join(rootPath, "node_modules"), nmPath); err != nil {
			return nil, err
		}
	}

	// Download the chart if it doesn't exist to chartPath
	chartPath := filepath.Join(cd.CacheDir, "chart")
	externalChartFile := filepath.Join(cd.ManifestDir, "external-chart")
	if util.FileExists(externalChartFile) {
		b, err := ioutil.ReadFile(externalChartFile)
		if err != nil {
			return nil, err
		}
		externalChart := string(b)

		u, err := url.Parse(externalChart)
		if err != nil {
			return nil, err
		}
		if len(u.Scheme) > 0 {
			// Remove the last path element from the URL; that's the chartName
			cname := filepath.Base(u.Path)
			u.Path = filepath.Dir(u.Path)
			crepo := strings.ReplaceAll(u.Host, ".", "-")
			externalChart = fmt.Sprintf("%s/%s", crepo, cname)

			out, err := util.ExecuteCommand("helm", "repo", "list")
			if err != nil {
				return nil, err
			}
			// Only add the repo if it doesn't already exist
			if !strings.Contains(out, crepo) {
				log.Infof("Adding a new helm repo called %q pointing to %q", crepo, u.String())
				_, err = util.ExecuteCommand("helm", "repo", "add", crepo, u.String())
				if err != nil {
					return nil, err
				}
			}
		} else {
			arr := strings.Split(externalChart, "/")
			if len(arr) != 2 {
				return nil, fmt.Errorf("invalid format of %q: %q. Should be either {stable,test}/{name} or {repo-url}/{name}", externalChartFile, externalChart)
			}
		}

		log.Infof("Found external chart to download %q", externalChart)
		// this extracts the chart to e.g. .cache/kubernetes-dashboard/kubernetes-dashboard
		// although it should be .cache/kubernetes-dashboard/chart
		_, err = util.ExecuteCommand("helm", "fetch", externalChart, "--untar", "--untardir", cd.CacheDir)
		if err != nil {
			return nil, err
		}

		// Remove chartPath if it already exists
		if err := os.RemoveAll(chartPath); err != nil {
			return nil, err
		}

		// as described above, e.g. .cache/kubernetes-dashboard/kubernetes-dashboard
		wrongPath := filepath.Join(cd.CacheDir, filepath.Base(externalChart))
		// make the path right
		log.Debugf("Renaming %q to %q", wrongPath, chartPath)
		if err := os.Rename(wrongPath, chartPath); err != nil {
			return nil, err
		}
	}

	for _, f := range []string{pipeJS, valuesJS, valuesYAML, chartDir} {
		from := filepath.Join(cd.ManifestDir, f)
		to := filepath.Join(cd.CacheDir, f)
		if exists, _ := util.PathExists(from); exists {
			cd.CopiedFiles[f] = struct{}{}
			if err := util.Copy(from, to); err != nil {
				return nil, err
			}
		}
	}
	return cd, nil
}

func GenerateChart(cd *ChartData, clusterInfo *config.ClusterInfo) error {
	pipeJSPath := pipeJS
	if _, ok := cd.CopiedFiles[pipeJS]; !ok {
		pipeJSPath = "../../jkcfg/default-pipe.js"
	}
	valuesJSPath := valuesJS
	if _, ok := cd.CopiedFiles[valuesJS]; !ok {
		valuesJSPath = "../../jkcfg/default-values.js"
	}
	valuesArgMap := map[string]string{
		"cluster-number": fmt.Sprintf("%q", clusterInfo.Index), // pass this as a string to preserve the 0-padding
		"root-domain":    clusterInfo.RootDomain,
		"git-repo":       clusterInfo.GitRepo,
		"provider":       clusterInfo.Provider,
	}
	valuesArgStr := ""
	for k, v := range valuesArgMap {
		valuesArgStr += fmt.Sprintf("-p=%s=%s ", k, v)
	}
	namespace := "default"
	nsFile := filepath.Join(cd.ManifestDir, "namespace")
	if util.FileExists(nsFile) {
		b, err := ioutil.ReadFile(nsFile)
		if err != nil {
			return err
		}
		namespace = string(b)
	}

	cmd := fmt.Sprintf(
		"cd %s && jk run %s %s | helm template -n %s workshopctl chart -f - | jk run %s %s",
		cd.CacheDir,
		valuesArgStr,
		valuesJSPath,
		namespace,
		valuesArgStr,
		pipeJSPath,
	)
	content, err := util.ExecuteCommand("/bin/bash", "-c", cmd)
	if err != nil {
		return err
	}

	outputFile := filepath.Join(clusterInfo.RootDir, "clusters", clusterInfo.Index.String(), fmt.Sprintf("%s.yaml", cd.Name))
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(outputFile, []byte(content), 0644)
}
