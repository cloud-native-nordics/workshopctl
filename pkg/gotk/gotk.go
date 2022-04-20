package gotk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
)

func SetupGitOps(ctx context.Context, info *config.ClusterInfo) error {
	mux, ok := util.GetMutex(ctx)
	if !ok || mux == nil {
		return fmt.Errorf("SetupGitOps: programmer error, couldn't get mutex for locking: %v", mux)
	}

	logger := util.Logger(ctx)
	logger.Debug("Waiting for mutex unlock in SetupGitOps")
	// Lock during this operation, as the git repo is mutually exclusive
	mux.Lock()
	logger.Infof("Bootstrapping GitOps for cluster %s...", info.Index)
	defer mux.Unlock()
	defer logger.Infof("Bootstrapping GitOps for cluster %s is done!", info.Index)

	// Make sure we have all prereqs
	kubeConfigArg := "--kubeconfig=" + info.Index.KubeConfigPath()
	/*_, _, err := util.Command(ctx,
		"gotk",
		kubeConfigArg,
		"check",
		"--pre",
	).Run()
	if err != nil {
		return err
	}*/

	var provider string
	switch info.Git.RepoStruct.Domain {
	case "github.com":
		provider = "github"
	case "gitlab.com":
		provider = "gitlab"
	default:
		return fmt.Errorf("git repo %s: unknown provider domain", info.Git.Repo)
	}

	// TODO: Upstream gotk doesn't support the --kubeconfig flag in install/bootstrap at least
	// Instead, we use the KUBECONFIG env var for now
	kubeConfigEnv := "KUBECONFIG=" + util.JoinPaths(ctx, info.Index.KubeConfigPath())

	// We assume that the repo is already created, hence we can skip some flags related to that
	// TODO: That doesn't work in current gotk, rework that maybe upstream too?
	// This command installs the toolkit into the target cluster, and starts reconciling our
	// given cluster directory for changes.
	_, _, err := util.Command(ctx,
		"flux",
		kubeConfigArg,
		"bootstrap",
		provider,
		"--owner="+info.Git.RepoStruct.UserLogin,
		"--repository="+info.Git.RepoStruct.RepositoryName,
		"--path="+info.Index.ClusterDir(),
		// Only install these two for now. TODO: In the future, also include notifications
		"--components=source-controller,kustomize-controller,helm-controller",
		// Use a short interval as this is a highly dynamic env
		"--interval=30s",
		// TODO: Assuming personal for now
		"--personal",
	).WithStdio(nil, os.Stdout, os.Stderr).
		WithEnv(
			// Forward the {GITHUB,GITLAB}_TOKEN variable from the config file
			fmt.Sprintf("%s_TOKEN=%s", strings.ToUpper(provider), info.Git.ServiceAccountContent),
			// Forward the PATH variable
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			kubeConfigEnv,
		).Run()
	return err
}
