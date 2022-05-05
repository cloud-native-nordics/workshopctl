# k8s-web-ide

A Docker image that builds upon [cdr/code-server](https://github.com/cdr/code-server), and adds

- `kubectl` -- the Kubernetes CLI
- `helm` -- the Kubernetes package manager
- Kubernetes syntax highlighting for YAML files (from the [YAML VSCode extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml))
- Common development utilities -- `curl`, `nano`, `jq`, and `git`
