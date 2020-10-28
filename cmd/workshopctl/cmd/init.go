package cmd

import (
	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type InitFlags struct {
	*RootFlags
	*config.Config

	Yes bool
}

// NewInitCommand returns the "init" command
func NewInitCommand(rf *RootFlags) *cobra.Command {
	inf := &InitFlags{
		RootFlags: rf,
		Config:    &config.Config{},
	}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Setup the user configuration interactively",
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunInit(inf); err != nil {
				log.Fatal(err)
			}
		},
	}

	addInitFlags(cmd.Flags(), inf)
	return cmd
}

func addInitFlags(fs *pflag.FlagSet, inf *InitFlags) {
	fs.StringVar(&inf.Name, "name", inf.Name, "What name this workshop should have")
	fs.StringVar(&inf.CloudProvider.ServiceAccountPath, "cloud-provider-service-account-path", inf.CloudProvider.ServiceAccountPath, "Path to service account for cloud provider")
	fs.StringVar(&inf.DNSProvider.ServiceAccountPath, "dns-provider-service-account-path", inf.DNSProvider.ServiceAccountPath, "Path to service account for dns provider")
	fs.StringVar(&inf.RootDomain, "root-domain", inf.RootDomain, "What the root domain to be managed is")
	fs.StringVar(&inf.LetsEncryptEmail, "lets-encrypt-email", inf.LetsEncryptEmail, "What Let's Encrypt email to use")
	fs.StringVar(&inf.Git.Repo, "git-repo", inf.Git.Repo, "What git repo to use. By default, try to auto-detect git remote origin.")
	fs.StringVar(&inf.Git.ServiceAccountPath, "git-provider-service-account-path", inf.Git.ServiceAccountPath, "Path to service account for git provider")

	fs.BoolVarP(&inf.Yes, "yes", "y", inf.Yes, "Overwrite the workshopctl.yaml file although it exists")
}

func RunInit(inf *InitFlags) error {
	// TODO: Make this a command-line-input based workflow?
	// Don't dry-run, no need for that
	ctx := util.NewContext(false, inf.RootDir)
	if util.FileExists(inf.ConfigPath) && !inf.Yes {
		log.Infof("%s already exists, and --yes isn't specified, won't overwrite file", inf.ConfigPath)
		return nil
	}

	// Try to dynamically figure out the git origin
	if inf.Git.Repo == "" {
		rootPath := util.JoinPaths(ctx)
		origin, _, err := util.ShellCommand(ctx, `git -C %s remote -v | grep push | grep origin | awk '{print $2}'`, rootPath).Run()
		if err != nil {
			return err
		}
		inf.Git.Repo = origin
	}

	if err := inf.Config.Complete(ctx); err != nil {
		return err
	}

	return util.WriteYAMLFile(ctx, inf.ConfigPath, inf.Config)
}
