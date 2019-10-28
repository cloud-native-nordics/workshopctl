FROM codercom/code-server:2.1650-vsc1.39.2

# Install needed utilities, e.g. Git is essential for the IDE
USER root
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        curl \
        nano \
        docker.io \
        npm \
        bsdtar && \
    apt-get clean

# Install kubectl
ENV K8S_VERSION=v1.16.2
RUN curl -sSL https://dl.k8s.io/release/${K8S_VERSION}/bin/linux/amd64/kubectl > /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# Install helm
ENV HELM_HOME=/root/.helm
ENV HELM_VERSION=v3.0.0-beta.5
RUN curl -sSL https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | sudo tar -xz -C /usr/local/bin linux-amd64/helm --strip-components=1

USER coder

# Install extensions
ENV EXTENSIONS="redhat.vscode-yaml PeterJausovec.vscode-docker"
# Maybe upgrade to ms-azuretools.vscode-docker that is v0.8.2 instead of
# PeterJausovec.vscode-docker that's pinned to v0.6.2?
# https://github.com/microsoft/vscode-docker/blob/master/CHANGELOG.md#L79
# Consider enabling ms-kubernetes-tools.vscode-kubernetes-tools; it's very resource-intensive, though
RUN for ext in ${EXTENSIONS}; do code-server --install-extension ${ext} && \
    (cd /home/coder/.local/share/code-server/extensions/${ext}*/ && \
    npm install || echo "ignore possible errors"); done

COPY --chown=coder:coder entrypoint.sh /
COPY --chown=coder:coder settings.json /home/coder/.local/share/code-server/User/settings.json

# By default run behind a Let's Encrypt proxy
ENTRYPOINT ["dumb-init", "/entrypoint.sh"]
