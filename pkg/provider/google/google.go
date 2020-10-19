package google

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/provider"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
)

// --scopes "https://www.googleapis.com/auth/devstorage.read_only","https://www.googleapis.com/auth/logging.write","https://www.googleapis.com/auth/monitoring","https://www.googleapis.com/auth/servicecontrol","https://www.googleapis.com/auth/service.management.readonly","https://www.googleapis.com/auth/trace.append","https://www.googleapis.com/auth/ndev.clouddns.readwrite"  \

const cmd = `gcloud beta container \
	--project "dx-training" \
	clusters create "%s" \
	--zone "us-central1-a" \
	--no-enable-basic-auth \
	--cluster-version "1.14.6-gke.13" \
	--machine-type "n1-standard-4" \
	--image-type "COS" \
	--disk-type "pd-standard" \
	--disk-size "100" \
	--metadata disable-legacy-endpoints=true \
	--scopes "https://www.googleapis.com/auth/cloud-platform" \
	--num-nodes "1" \
	--enable-stackdriver-kubernetes \
	--enable-ip-alias \
	--network "projects/dx-training/global/networks/default-2" \
	--subnetwork "projects/dx-training/regions/us-central1/subnetworks/default-2" \
	--default-max-pods-per-node "110" \
	--addons HorizontalPodAutoscaling,HttpLoadBalancing \
	--enable-autoupgrade \
	--enable-autorepair \
	--no-shielded-integrity-monitoring
`

func NewGoogleProvider(sa *config.ServiceAccount, dryRun bool) provider.Provider {
	p := &GoogleProvider{
		sa:     sa,
		dryRun: dryRun,
	}
	return p
}

type GoogleProvider struct {
	sa     *config.ServiceAccount
	dryRun bool
}

func (google *GoogleProvider) CreateCluster(i config.ClusterNumber, c provider.ClusterSpec) (*provider.Cluster, error) {
	logger := i.NewLogger()

	start := time.Now().UTC()
	cluster := &provider.Cluster{
		Spec: c,
		Status: provider.ClusterStatus{
			ProvisionStart: &start,
			Index:          i,
		},
	}

	command := fmt.Sprintf(cmd, c.Name)

	if google.dryRun {
		logger.Infof("Would send this request to DO: %s", cmd)
		return cluster, nil
	}

	if out, err := util.ExecuteCommand("gcloud", "container", "clusters", "list"); err != nil {
		return nil, err
	} else if !strings.Contains(out, c.Name) {
		// If the cluster isn't in the list, create it
		out, err := util.ExecuteCommand("/bin/bash", "-c", command)
		logger.Infof("Cmd output: %s", out)
		if err != nil {
			return nil, err
		}
	}

	d, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(d)

	kubeconfigPath := filepath.Join(d, "kube.conf")
	authCmd := fmt.Sprintf(`KUBECONFIG=%s gcloud --project dx-training container clusters get-credentials %s --zone us-central1-a`, kubeconfigPath, c.Name)
	if _, err := util.ExecuteCommand("/bin/bash", "-c", authCmd); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	cluster.Status.ProvisionDone = &now

	cluster.Status.KubeconfigBytes, err = ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}
