import * as k8s from '@jkcfg/kubernetes/api';
import { kubeMutator, kubePipeWithMutators, withNamespace } from "../../jkcfg/util"

function withDOTokenEnvVar() {
    return kubeMutator(k8s.apps.v1.Deployment, "external-dns", function (obj) {
        var dep = new k8s.apps.v1.Deployment(); dep = obj
        var env = new k8s.core.v1.EnvVar()
        env.name = "DO_TOKEN"
        env.valueFrom = new k8s.core.v1.EnvVarSource()
        env.valueFrom.secretKeyRef = new k8s.core.v1.SecretKeySelector()
        env.valueFrom.secretKeyRef.name = "external-dns"
        env.valueFrom.secretKeyRef.key = "DO_TOKEN"
        if (!dep.spec.template.spec.containers[0].env) dep.spec.template.spec.containers[0].env = []
        dep.spec.template.spec.containers[0].env.push(env)
        return dep;
    })
}

kubePipeWithMutators([
    withDOTokenEnvVar(),
    withNamespace("workshopctl"),
])