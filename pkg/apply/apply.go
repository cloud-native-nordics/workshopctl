package apply

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/luxas/workshopctl/pkg/config"
	"github.com/luxas/workshopctl/pkg/provider"
	"github.com/luxas/workshopctl/pkg/provider/digitalocean"
	"github.com/luxas/workshopctl/pkg/util"
)

var providers = map[string]provider.ProviderFunc{
	"digitalocean": digitalocean.NewDigitalOceanProvider,
}

func Apply(cfg *config.Config, dryrun bool) error {
	pFunc, ok := providers[cfg.Provider]
	if !ok {
		return fmt.Errorf("Provider %s is not supported!", cfg.Provider)
	}
	if len(cfg.ServiceAccountStr) == 0 {
		return fmt.Errorf("A ServiceAccount token for the provider is required")
	}
	cfg.ServiceAccount = config.NewServiceAccount(cfg.ServiceAccountStr)
	if _, err := cfg.ServiceAccount.Get(); err != nil {
		return err
	}
	p := pFunc(cfg.ServiceAccount, dryrun)

	return config.ForCluster(cfg.Clusters, cfg, func(clusterInfo *config.ClusterInfo) error {
		return ApplyCluster(clusterInfo, p, dryrun)
	})
}

func ApplyCluster(clusterInfo *config.ClusterInfo, p provider.Provider, dryrun bool) error {
	// Add some kind of mark at the end of this procedure in the cluster to say that it's
	// been successfully provisioned (maybe in the workshopctl ConfigMap?). With this feature
	// it's possible at this stage to skip doing the same things over and over again => idempotent

	kubeconfigPath := clusterInfo.KubeConfigPath()
	if !util.FileExists(kubeconfigPath) {
		if err := provisionCluster(clusterInfo, p, dryrun); err != nil {
			return err
		}
	}

	if out, err := execKubectl(kubeconfigPath, "create", "ns", "workshopctl"); err != nil {
		// Allow/Ignore the AlreadyExists error
		if !strings.Contains(out, "AlreadyExists") {
			return err
		}
	}

	// Read the token; it could be from a file, too
	// Ignore the error here safely as it's been verified already
	token, _ := clusterInfo.ServiceAccount.Get()

	args := []string{
		"-n",
		"workshopctl",
		"create",
		"secret",
		"generic",
		"workshopctl",
		fmt.Sprintf("--from-literal=PROVIDER=%s", clusterInfo.Provider),
		fmt.Sprintf("--from-literal=PROVIDER_SERVICEACCOUNT=%s", token),
		fmt.Sprintf("--from-literal=GIT_REPO=%s", clusterInfo.GitRepo),
		fmt.Sprintf("--from-literal=ROOT_DOMAIN=%s", clusterInfo.RootDomain),
		fmt.Sprintf("--from-literal=CLUSTER_NUMBER=%s", clusterInfo.Index),
		fmt.Sprintf("--from-literal=VSCODE_PASSWORD=%s", clusterInfo.VSCodePassword),
	}
	if out, err := execKubectl(kubeconfigPath, args...); err != nil {
		// Allow/Ignore the AlreadyExists error
		if !strings.Contains(out, "AlreadyExists") {
			return err
		}
	}

	requiredAddons := []string{"flux", "core-workshop-infra"}
	for _, addon := range requiredAddons {
		addonPath := fmt.Sprintf("clusters/%s/%s.yaml", clusterInfo.Index, addon)
		clusterInfo.Logger.Infof("Applying addon %s", addonPath)
		if _, err := execKubectl(kubeconfigPath, "apply", "-f", addonPath); err != nil {
			return err
		}
	}

	// Wait for the cluster to be healthy
	w := NewWaiter(clusterInfo)
	return w.WaitForAll()
}

func provisionCluster(clusterInfo *config.ClusterInfo, p provider.Provider, dryrun bool) error {
	clusterInfo.Logger.Infof("Provisioning cluster %s...", clusterInfo.Index)
	cluster, err := p.CreateCluster(clusterInfo.Index, provider.ClusterSpec{
		Name: fmt.Sprintf("workshopctl-cluster-%s", clusterInfo.Index),
		NodeSize: provider.NodeSize{
			CPUs: clusterInfo.CPUs,
			RAM:  clusterInfo.RAM,
		},
		NodeCount: clusterInfo.Clusters,
		Version:   "latest",
	})
	if err != nil {
		return fmt.Errorf("encountered an error while creating clusters: %v", err)
	}
	if dryrun {
		return nil
	}

	clusterInfo.Logger.Infof("Provisioning of cluster %s took %s.", cluster.Spec.Name, cluster.Status.ProvisionDone.Sub(*cluster.Status.ProvisionStart))
	util.DebugObject("Returned cluster object", *cluster)

	kubeconfigPath := clusterInfo.KubeConfigPath()
	clusterInfo.Logger.Infof("Writing KubeConfig file to %q", kubeconfigPath)
	return ioutil.WriteFile(kubeconfigPath, cluster.Status.KubeconfigBytes, 0600)
}

func execKubectl(kubeconfigPath string, args ...string) (string, error) {
	kubectlArgs := []string{"--kubeconfig", kubeconfigPath}
	kubectlArgs = append(kubectlArgs, args...)
	return util.ExecuteCommand("kubectl", kubectlArgs...)
}
