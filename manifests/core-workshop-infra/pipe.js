import * as k8s from '@jkcfg/kubernetes/api';
import { kubeMutator, kubePipeWithMutators, withNamespace } from "../../jkcfg/util"

function withDOTokenEnvVar() {
    return kubeMutator(k8s.apps.v1.Deployment, "external-dns", function (obj, workshopctl) {
        var dep = new k8s.apps.v1.Deployment(); dep = obj
        // This code transforms the environment variable name the serviceaccount token is exposed as
        // e.g. external-dns expects the variable to be named DO_TOKEN when the provider is digitalocean
        dep.spec.template.spec.containers[0].env.forEach((env) => {
            if (env.name == "PROVIDER_SERVICEACCOUNT") {
                if (workshopctl.provider == "digitalocean") {
                    env.name = "DO_TOKEN"
                } // support for more providers can be added here
            }
        })
        return dep;
    })
}

kubePipeWithMutators([
    withDOTokenEnvVar(),
    withNamespace("workshopctl"),
])