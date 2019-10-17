import { valuesPipeForFunctions, valuesMutator } from "../../jkcfg/util"

function withCustomIngressHost() {
    return valuesMutator(function(values) {
        values.ingress.hosts = [
            "dashboard.cluster-" + values.workshopctl.clusterNumber + "." + values.workshopctl.domain
        ]
        return values
    })
}

valuesPipeForFunctions([
    withCustomIngressHost(),
])