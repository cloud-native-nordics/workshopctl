package gotk

import (
	"context"
	"fmt"
	"os"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
)

func SetupGitOps(ctx context.Context, info *config.ClusterInfo) error {
	// Make sure we have all prereqs
	kubeConfigArg := "--kubeconfig=" + info.Index.KubeConfigPath()
	_, _, err := util.Command(ctx,
		"gotk",
		kubeConfigArg,
		"check",
		"--pre",
	).Run()
	if err != nil {
		return err
	}

	var provider string
	switch info.GitRepoStruct.Domain {
	case "github.com":
		provider = "github"
	case "gitlab.com":
		provider = "gitlab"
	default:
		return fmt.Errorf("git repo %s: unknown provider domain", info.GitRepo)
	}

	// We assume that the repo is already created, hence we can skip some flags related to that
	// This command installs the toolkit into the target cluster, and starts reconciling our
	// given cluster directory for changes.
	_, _, err = util.Command(ctx,
		"gotk",
		kubeConfigArg,
		"bootstrap",
		provider,
		"--owner="+info.GitRepoStruct.UserLogin,
		"--repository="+info.GitRepoStruct.RepositoryName,
		"--path="+info.Index.ClusterDir(),
		// Only install these two for now. TODO: In the future, also include notifications
		"--components=source-controller,kustomize-controller,helm-controller",
		// Use a short interval as this is a highly dynamic env
		"--interval=30s",
	).WithStdio(nil, os.Stdout, os.Stderr).Run()
	return err
}
