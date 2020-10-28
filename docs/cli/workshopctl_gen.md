## workshopctl gen

Generate a set of manifests based on the configuration

```
workshopctl gen [flags]
```

### Options

```
  -h, --help                help for gen
      --skip-local-charts   Don't consider the local directory's charts/ directory
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

