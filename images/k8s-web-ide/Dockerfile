# This image has VS Code v1.65.2, built 2022-04-14
FROM codercom/code-server:4.3.0

# Install needed utilities, e.g. Git is essential for the IDE
USER root
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        curl \
        nano \
        jq && \
    apt-get clean

# Install kubectl
ENV K8S_VERSION=v1.22.8
RUN curl -sSL https://dl.k8s.io/release/${K8S_VERSION}/bin/linux/amd64/kubectl > /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# Install helm
ENV HELM_HOME=/root/.helm
ENV HELM_VERSION=v3.8.2
RUN curl -sSL https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | sudo tar -xz -C /usr/local/bin linux-amd64/helm --strip-components=1

USER coder

# Install extensions
ENV EXTENSIONS="redhat.vscode-yaml ms-azuretools.vscode-docker"
# Consider enabling ms-kubernetes-tools.vscode-kubernetes-tools; it's very resource-intensive, though
RUN for ext in ${EXTENSIONS}; do code-server --install-extension ${ext}; done

COPY --chown=coder:coder entrypoint.sh /
COPY --chown=coder:coder settings.json /home/coder/.local/share/code-server/User/settings.json
COPY --chown=coder:coder kubeconfig.yaml /home/coder/.kube/config
COPY --chown=coder:coder .bash_aliases /home/coder/.bash_aliases

ENTRYPOINT ["dumb-init", "/entrypoint.sh"]
