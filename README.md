# workshopctl

A tool for running workshops easily in the cloud!

**WARNING:** This tool is pre-alpha and under heavy development. Don't use it for anything
very important quite yet! However, contributions are very welcome!

## Quick Start

1. `workshopctl init` -- Give information about what cloud provider to use (and its token),
    and what domain to serve on (e.g. `workshopctl.kubernetesfinland.com`)
1. `workshopctl gen --clusters 40` -- Generate 40 unique sets of Kubernetes manifests, one per cluster.
1. `workshopctl apply` -- Creates the clusters in the cloud, and applies the manifests

Boom! A Visual Studio Code instance running in the browser is now available at [cluster-01.workshopctl.kubernetesfinland.com](https://cluster-01.workshopctl.kubernetesfinland.com).
The VS Code terminal has full privileges to the Kubernetes cluster, so the attendee may easily
access `kubectl`, `helm` and `docker` (if needed) for completing the tasks in your workshop.
You can also provide pre-created materials in VS Code for the attendee.

## How this works

`workshop gen` generates unique manifests for any number of workshop clusters
you need. The base unit for writing the manifests is [Helm](https://helm.sh/) Charts, but with a
twist. We're using [jkcfg](https://jkcfg.github.io/#/) ("Javascript Kubernetes", configuration as
code) to both preprocess the `values.yaml` file, and the output from Helm.

In other words the flow looks like this:

`workshopctl gen` -> `jk run values.js` -> `helm template` -> `jk run pipe.js` -> `clusters/XX/manifest.yaml`

```txt
`workshopctl gen`: from 1 -> {clusters (e.g. 50)}, do:
--> Find Helm chart in manifests/<name>/chart
---> Preprocess `values.yaml` using `jk run values.js`
----> Run `helm template` for the given `values.yaml`
-----> Pipe the content into `jk run pipe.js`, which patches the Helm output on the fly
------> Save content to cluster/$i/<name>.yaml
```
