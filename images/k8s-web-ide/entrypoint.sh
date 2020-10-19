#!/bin/bash

if [[ ${GIT_REPO} != "" ]]; then
    git clone --depth 5 ${GIT_REPO} /home/coder/gitclone
    mv /home/coder/gitclone/${GIT_SUBDIR}/* /home/coder/project
fi

sudo chown $(id -u):$(id -g) /var/run/docker.sock

exec code-server --host=0.0.0.0 --auth=password --disable-telemetry