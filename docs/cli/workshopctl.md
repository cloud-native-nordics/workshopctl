## workshopctl

workshopctl: easily run Kubernetes workshops

### Options

```
      --config-path string   Where to find the config file (default "workshopctl.yaml")
      --dry-run              Whether to apply the selected operation, or just print what would happen (to dry-run) (default true)
  -h, --help                 help for workshopctl
      --log-level loglevel   Specify the loglevel for the program (default info)
      --root-dir string      Where the workshopctl directory is. Must be a Git repo. (default ".")
```

### SEE ALSO

* [workshopctl apply](workshopctl_apply.md)	 - Create a Kubernetes cluster and apply the desired manifests
* [workshopctl cleanup](workshopctl_cleanup.md)	 - Delete the k8s-managed cluster
* [workshopctl gen](workshopctl_gen.md)	 - Generate a set of manifests based on the configuration
* [workshopctl init](workshopctl_init.md)	 - Setup the user configuration interactively
* [workshopctl kubectl](workshopctl_kubectl.md)	 - An alias for the kubectl command, pointing the KUBECONFIG to the right place
* [workshopctl version](workshopctl_version.md)	 - Print the version

