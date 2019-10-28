## workshopctl apply

Create a Kubernetes cluster and apply the desired manifests

### Synopsis

Create a Kubernetes cluster and apply the desired manifests

```
workshopctl apply [flags]
```

### Options

```
  -c, --clusters uint16          How many clusters to create (default 1)
      --dry-run                  Whether to dry-run or not (default true)
  -r, --git-repo string          What git repo to use (default "https://github.com/luxas/workshopctl")
  -h, --help                     help for apply
      --node-count uint16        How many nodes per cluster (default 1)
      --node-cpus uint16         How much CPUs to use per-node (default 2)
      --node-ram uint16          How much RAM to use per-node (default 2)
      --provider string          What provider to use (default "digitalocean")
      --root-dir string          Where the workshopctl directory is (default ".")
  -d, --root-domain string       What domain to use (default "workshopctl.kubernetesfinland.com")
      --service-account string   What serviceaccount/token to use. Can be a string or a file
      --vscode-password string   What the password for Visual Studio Code should be (default "kubernetesrocks")
```

### Options inherited from parent commands

```
      --log-level loglevel   Specify the loglevel for the program (default info)
```

### SEE ALSO

* [workshopctl](workshopctl.md)	 - workshopctl: easily run Kubernetes workshops

