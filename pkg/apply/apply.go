package apply

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/config/keyval"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider/providers"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	"github.com/sirupsen/logrus"
)

func Apply(ctx context.Context, cfg *config.Config, dryrun bool) error {
	// TODO: Enforce that gen is up-to-date

	cp, err := providers.CloudProviders().NewCloudProvider(ctx, &cfg.CloudProvider, dryrun)
	if err != nil {
		return err
	}

	return config.ForCluster(ctx, cfg.Clusters, cfg, func(clusterCtx context.Context, clusterInfo *config.ClusterInfo) error {
		return ApplyCluster(clusterCtx, clusterInfo, cp, dryrun)
	})
}

func ApplyCluster(ctx context.Context, clusterInfo *config.ClusterInfo, p provider.CloudProvider, dryrun bool) error {
	logger := util.Logger(ctx)

	// Add some kind of mark at the end of this procedure in the cluster to say that it's
	// been successfully provisioned (maybe in the workshopctl ConfigMap?). With this feature
	// it's possible at this stage to skip doing the same things over and over again => idempotent

	kubeconfigPath := clusterInfo.KubeConfigPath()
	if !util.FileExists(kubeconfigPath) {
		logger.Info("Provisioning the Kubernetes cluster")
		if err := provisionCluster(ctx, clusterInfo, p, dryrun); err != nil {
			return err
		}
	} else {
		logger.Infof("Assuming cluster is already provisioned, as %q exists...", kubeconfigPath)
	}

	localKubectl := func() *kubectlExecer {
		return kubectl(kubeconfigPath, dryrun).WithNS(constants.WorkshopctlNamespace)
	}

	logger.Info("Applying workshopctl Namespace")
	if _, err := kubectl(kubeconfigPath, dryrun).
		Create("namespace", "", constants.WorkshopctlNamespace, true, false).
		Run(); err != nil {
		return err
	}

	paramFlags := []string{}
	// Append secret parameters
	parameters := keyval.FromClusterInfo(clusterInfo)
	for k, v := range parameters.ToMap() {
		paramFlags = append(paramFlags, fmt.Sprintf("--from-literal=%s=%s", k, v))
	}

	logger.Info("Applying workshopctl Secret")
	if _, err := localKubectl().
		Create("secret", "generic", constants.WorkshopctlSecret, true, true).
		WithArgs(paramFlags...).
		Run(); err != nil {
		return err
	}

	requiredAddons := []string{"core-workshop-infra"} // TODO: GOTK
	for _, addon := range requiredAddons {
		addonPath := fmt.Sprintf("clusters/%s/%s.yaml", clusterInfo.Index, addon)
		logger.Infof("Applying addon %s", addonPath)
		if _, err := localKubectl().WithArgs("apply").WithFile(addonPath).Run(); err != nil {
			return err
		}
	}

	// Wait for the cluster to be healthy
	return NewWaiter(ctx, clusterInfo, dryrun).WaitForAll()
}

func provisionCluster(ctx context.Context, clusterInfo *config.ClusterInfo, p provider.CloudProvider, dryrun bool) error {
	logger := util.Logger(ctx)

	logger.Infof("Provisioning cluster %s...", clusterInfo.Index)
	cluster, err := p.CreateCluster(ctx, clusterInfo.Index, provider.ClusterSpec{
		Name:       fmt.Sprintf("workshopctl-cluster-%s", clusterInfo.Index),
		Version:    "latest",
		NodeGroups: clusterInfo.NodeGroups,
	})
	if err != nil {
		return fmt.Errorf("encountered an error while creating clusters: %v", err)
	}
	if dryrun {
		return nil
	}

	logger.Infof("Provisioning of cluster %s took %s.", cluster.Spec.Name, cluster.Status.ProvisionDone.Sub(*cluster.Status.ProvisionStart))
	util.DebugObject("Returned cluster object", *cluster)

	kubeconfigPath := clusterInfo.KubeConfigPath()
	logger.Infof("Writing KubeConfig file to %q", kubeconfigPath)
	return ioutil.WriteFile(kubeconfigPath, cluster.Status.KubeconfigBytes, 0600)
}

type kubectlExecer struct {
	kubeConfigPath string
	dryRun         bool

	namespace string
	args      []string
	files     []string

	err          error
	ignoreErrors []string
}

func kubectl(kubeConfigPath string, dryRun bool) *kubectlExecer {
	return &kubectlExecer{
		kubeConfigPath: kubeConfigPath,
		dryRun:         dryRun,
	}
}

func (e *kubectlExecer) WithNS(ns string) *kubectlExecer {
	e.namespace = ns
	return e
}

func (e *kubectlExecer) WithFile(file string) *kubectlExecer {
	e.files = append(e.files, file)
	e.args = append(e.args, []string{"-f", file}...)
	return e
}

func (e *kubectlExecer) WithArgs(args ...string) *kubectlExecer {
	e.args = append(e.args, args...)
	return e
}

func (e *kubectlExecer) Create(kind, subkind, name string, ignoreExists, recreate bool) *kubectlExecer {
	e.args = append(e.args, "create", kind)
	if len(subkind) > 0 {
		e.args = append(e.args, subkind)
	}
	e.args = append(e.args, name)
	if ignoreExists {
		// if we're idempotent, we don't care about "already exists" errors
		e.ignoreErrors = append(e.ignoreErrors, "AlreadyExists")
	}
	if recreate {
		_, err := kubectl(e.kubeConfigPath, e.dryRun).
			WithNS(e.namespace).
			WithArgs("delete", kind, name).
			Run()
		if err != nil {
			e.err = err
		}
	}
	return e
}

func (e *kubectlExecer) Run() (string, error) {
	if e.err != nil {
		return "", e.err
	}

	kubectlArgs := []string{"--kubeconfig", e.kubeConfigPath}
	if len(e.namespace) != 0 {
		kubectlArgs = append(kubectlArgs, []string{"-n", e.namespace}...)
	}
	kubectlArgs = append(kubectlArgs, e.args...)
	if e.dryRun {
		logrus.Infof("Would run kubectl command: 'kubectl %s'", strings.Join(kubectlArgs, " "))
		if len(e.files) > 0 {
			logrus.Infof("Would send files: %v", e.files)
		}
		return "", nil
	}

	out, err := util.ExecuteCommand("kubectl", kubectlArgs...)
	for _, ignored := range e.ignoreErrors {
		if strings.Contains(out, ignored) {
			return out, nil
		}
	}
	return out, err
}
