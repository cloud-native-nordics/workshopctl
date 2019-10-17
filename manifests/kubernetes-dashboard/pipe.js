import * as k8s from '@jkcfg/kubernetes/api';
import { kubeMutator, kubePipeWithMutators, withNamespace } from "../../jkcfg/util"

function withoutClusterServiceLabel(name) {
    return kubeMutator(k8s.core.v1.Service, name, function (obj) {
        var svc = new k8s.core.v1.Service(); svc = obj
        delete svc.metadata.labels["kubernetes.io/cluster-service"];
        return svc;
    })
}

const resourceName = "workshopctl-kubernetes-dashboard"
kubePipeWithMutators([
    withoutClusterServiceLabel(resourceName),
    withNamespace("kube-system"), // the dashboard is hardcoded to run in kube-system :(
])