# k8s-web-ide

A Docker image that builds upon [cdr/code-server](https://github.com/cdr/code-server), and adds

- `kubectl` -- the Kubernetes CLI
- `helm` -- the Kubernetes package manager
- `docker` -- the Docker client in order to be able to push and pull images
- Kubernetes syntax highlighting for YAML files (from the [YAML VSCode extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml))
- Docker integration available to see images, containers, etc. (from the [Docker VSCode extension](https://github.com/microsoft/vscode-docker/tree/v0.6.2/))
- Common development utilities -- `curl`, `nano` and `git`
