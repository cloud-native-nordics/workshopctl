package apply

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/config/keyval"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/gotk"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider/providers"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
)

func Apply(ctx context.Context, cfg *config.Config) error {
	// TODO: Enforce that gen is up-to-date

	cloudP, err := providers.CloudProviders().NewCloudProvider(ctx, &cfg.CloudProvider)
	if err != nil {
		return err
	}

	dnsP, err := providers.DNSProviders().NewDNSProvider(ctx, &cfg.DNSProvider, cfg.RootDomain)
	if err != nil {
		return err
	}

	// Make sure the domain zone is created before starting to reconcile the clusters
	// Otherwise external-dns nor Traefik will work.
	if err := dnsP.EnsureZone(ctx); err != nil {
		return err
	}

	return config.ForCluster(ctx, cfg.Clusters, cfg, func(clusterCtx context.Context, clusterInfo *config.ClusterInfo) error {
		return ApplyCluster(clusterCtx, clusterInfo, cloudP)
	})
}

func ApplyCluster(ctx context.Context, clusterInfo *config.ClusterInfo, p provider.CloudProvider) error {
	logger := util.Logger(ctx)

	// Add some kind of mark at the end of this procedure in the cluster to say that it's
	// been successfully provisioned (maybe in the workshopctl ConfigMap?). With this feature
	// it's possible at this stage to skip doing the same things over and over again => idempotent

	kubeconfigPath := clusterInfo.Index.KubeConfigPath()
	if !util.FileExists(kubeconfigPath) {
		// TODO: Instead, make provisionCluster idempotent
		logger.Info("Provisioning the Kubernetes cluster")
		if err := provisionCluster(ctx, clusterInfo, p); err != nil {
			return err
		}
	} else {
		logger.Infof("Assuming cluster is already provisioned, as %q exists...", kubeconfigPath)
	}

	logger.Info("Applying workshopctl Namespace")
	if _, err := kubectl(ctx, kubeconfigPath).
		Create("namespace", "", constants.WorkshopctlNamespace, true, false).
		Run(); err != nil {
		return err
	}

	// Setup GitOps sync
	if err := gotk.SetupGitOps(ctx, clusterInfo); err != nil {
		return err
	}

	localKubectl := func() *kubectlExecer {
		return kubectl(ctx, kubeconfigPath).WithNS(constants.WorkshopctlNamespace)
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

	requiredAddons := []string{"core-workshop-infra"}
	for _, addon := range requiredAddons {
		addonPath := fmt.Sprintf("%s/%s/%s.yaml", constants.ClustersDir, clusterInfo.Index, addon)
		logger.Infof("Applying addon %s", addonPath)
		if _, err := localKubectl().WithArgs("apply").WithFile(addonPath).Run(); err != nil {
			return err
		}
	}

	// Wait for the cluster to be healthy
	return NewWaiter(ctx, clusterInfo).WaitForAll()
}

func provisionCluster(ctx context.Context, clusterInfo *config.ClusterInfo, p provider.CloudProvider) error {
	logger := util.Logger(ctx)

	logger.Infof("Provisioning cluster %s...", clusterInfo.Index)
	cluster, err := p.CreateCluster(ctx, provider.ClusterMeta{
		Index:      clusterInfo.Index,
		NamePrefix: clusterInfo.Name,
	}, provider.ClusterSpec{
		Version:    "latest",
		NodeGroups: clusterInfo.NodeGroups,
	})
	if err != nil {
		return fmt.Errorf("encountered an error while creating clusters: %v", err)
	}

	logger.Infof("Provisioning of cluster %s took %s.", cluster.Name(), cluster.Status.ProvisionTime())
	util.DebugObject(ctx, "Returned cluster object", cluster)

	kubeconfigPath := clusterInfo.Index.KubeConfigPath()
	logger.Infof("Writing KubeConfig file to %q", kubeconfigPath)
	return util.WriteFile(ctx, kubeconfigPath, cluster.Status.KubeconfigBytes)
}

type kubectlExecer struct {
	ctx            context.Context
	kubeConfigPath string

	namespace string
	args      []string
	files     []string

	err          error
	ignoreErrors []string
}

func kubectl(ctx context.Context, kubeConfigPath string) *kubectlExecer {
	return &kubectlExecer{
		ctx:            ctx,
		kubeConfigPath: kubeConfigPath,
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

func (e *kubectlExecer) IgnoreErrors(errStrs ...string) *kubectlExecer {
	e.ignoreErrors = append(e.ignoreErrors, errStrs...)
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
		e.IgnoreErrors("AlreadyExists")
	}
	if recreate {
		_, err := kubectl(e.ctx, e.kubeConfigPath).
			WithNS(e.namespace).
			WithArgs("delete", kind, name).
			IgnoreErrors("NotFound"). // Ignore any possible NotFound error here, that is expected
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

	out, _, err := util.Command(e.ctx, "kubectl", kubectlArgs...).Run()
	for _, ignored := range e.ignoreErrors {
		if strings.Contains(out, ignored) {
			return out, nil
		}
	}
	return out, err
}
