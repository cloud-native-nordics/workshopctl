## workshopctl gen

Generate a set of manifests based on the configuration

### Synopsis

Generate a set of manifests based on the configuration

```
workshopctl gen [flags]
```

### Options

```
  -c, --clusters uint16      How many clusters to create (default 1)
  -r, --git-repo string      What git repo to use (default "https://github.com/luxas/workshopctl")
  -h, --help                 help for gen
      --provider string      What provider to use (default "digitalocean")
      --root-dir string      Where the workshopctl directory is (default ".")
  -d, --root-domain string   What domain to use (default "workshopctl.kubernetesfinland.com")
```

### Options inherited from parent commands

```
      --log-level loglevel   Specify the loglevel for the program (default info)
```

### SEE ALSO

* [workshopctl](workshopctl.md)	 - workshopctl: easily run Kubernetes workshops

