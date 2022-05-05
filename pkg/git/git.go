package git

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
)

func PushManifests(ctx context.Context, cfg *config.Config) error {
	isNew := false
	if ok, fi := util.PathExists(".git"); !ok {
		if _, _, err := util.Command(ctx, "git", "init").Run(); err != nil {
			return err
		}
		isNew = true
	} else if !fi.IsDir() {
		return fmt.Errorf(".git must be a directory")
	}

	if !isNew {
		out, err := gitRun(ctx, "branch")
		if err != nil {
			return err
		}
		if len(out) != 0 {
			out, err := gitRun(ctx, "describe", "--dirty", "--always")
			if err != nil {
				return err
			}
			if strings.Contains(out, "dirty") {
				//return fmt.Errorf("won't do anything for dirty git state: %s", out)
				fmt.Println("git state is dirty")
			}
		}
	}

	gitIgnoreBytes, err := os.ReadFile(".gitignore")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	oldGitIgnore := string(gitIgnoreBytes)

	var foundTokens = map[string]bool{
		".cache":      false,
		".kube":       false,
		".kubeconfig": false,
	}
	foundTokens[cfg.DNSProvider.ServiceAccountPath] = false
	foundTokens[cfg.CloudProvider.ServiceAccountPath] = false
	foundTokens[cfg.Git.ServiceAccountPath] = false

	fileScanner := bufio.NewScanner(strings.NewReader(oldGitIgnore))
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		_, exists := foundTokens[fileScanner.Text()]
		if !exists {
			continue
		}
		foundTokens[fileScanner.Text()] = true
	}

	newGitIgnore := oldGitIgnore
	altered := false
	for name, exists := range foundTokens {
		if exists {
			continue
		}
		newGitIgnore += "\n" + name
		altered = true
	}
	if altered {
		newGitIgnore += "\n"
	}

	if oldGitIgnore != newGitIgnore {
		if err := os.WriteFile(".gitignore", []byte(newGitIgnore), 0644); err != nil {
			return err
		}
		if _, err := gitRun(ctx, "add", ".gitignore"); err != nil {
			return err
		}
	}

	if _, err := gitRun(ctx, "add", "clusters", "workshopctl.yaml"); err != nil {
		return err
	}

	out, err := gitRun(ctx, "remote")
	if err != nil {
		return err
	}
	if len(out) == 0 {
		_, err := gitRun(ctx, "remote", "add", "origin", cfg.Git.Repo)
		if err != nil {
			return err
		}
	}

	fmt.Println("Now run:\ngit commit -m 'Initial commit' && git push --set-upstream origin master")
	return nil
}

func gitRun(ctx context.Context, args ...string) (string, error) {
	out, _, err := util.Command(ctx, "git", args...).Run()
	return out, err
}
