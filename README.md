# workshopctl

A tool for running workshops easily in the cloud!

**WARNING:** This tool is pre-alpha and under heavy development. Don't use it for anything
very important quite yet! However, contributions are very welcome!

Please check out [these slides](https://docs.google.com/presentation/d/10OxH3s_dFDZ362NIy013LD5Gs78vBXeg2Mp9c5eEjoQ/edit#slide=id.ga4596f4c55_0_201) for an up-to-date description of this project.

## Quick Start

1. `workshopctl init` -- Give information about what cloud provider to use (and its token),
    and what domain to serve on (e.g. `workshopctl.kubernetesfinland.com`)
1. `workshopctl gen` -- Generate unique sets of Kubernetes manifests, one per cluster.
1. `workshopctl apply` -- Creates the clusters in the cloud, and applies the manifests

Boom! A Visual Studio Code instance running in the browser is now available at e.g. `cluster-01.workshopctl.kubernetesfinland.com` in the given example.
The VS Code terminal has full privileges to the Kubernetes cluster, so the attendee may easily
access `kubectl`, `helm` and `docker` (if needed) for completing the tasks in your workshop.
You can also provide pre-created materials in VS Code for the attendee.

## How this works

TODO: Write more docs here.
