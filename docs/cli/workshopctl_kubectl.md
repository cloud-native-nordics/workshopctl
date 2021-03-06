## workshopctl kubectl

An alias for the kubectl command, pointing the KUBECONFIG to the right place

```
workshopctl kubectl [kubectl commands] [flags]
```

### Options

```
  -c, --cluster cluster-number   What cluster number you want to connect to. Env var WORKSHOPCTL_CLUSTER can also be used.
  -h, --help                     help for kubectl
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

