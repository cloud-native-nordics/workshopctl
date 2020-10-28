## workshopctl init

Setup the user configuration interactively

```
workshopctl init [flags]
```

### Options

```
      --cloud-provider-service-account-path string   Path to service account for cloud provider
      --dns-provider-service-account-path string     Path to service account for dns provider
      --git-provider-service-account-path string     Path to service account for git provider
      --git-repo string                              What git repo to use. By default, try to auto-detect git remote origin.
  -h, --help                                         help for init
      --lets-encrypt-email string                    What Let's Encrypt email to use
      --name string                                  What name this workshop should have
      --root-domain string                           What the root domain to be managed is
  -y, --yes                                          Overwrite the workshopctl.yaml file although it exists
```

### Options inherited from parent commands

```
      --config-path string   Where to find the config file (default "workshopctl.yaml")
      --dry-run              Whether to apply the selected operation, or just print what would happen (to dry-run) (default true)
      --log-level loglevel   Specify the loglevel for the program (default info)
      --root-dir string      Where the workshopctl directory is. Must be a Git repo. (default ".")
```

### SEE ALSO

* [workshopctl](workshopctl.md)	 - workshopctl: easily run Kubernetes workshops

