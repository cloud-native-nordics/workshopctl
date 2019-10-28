import { valuesPipeForFunctions, valuesMutator } from "../../jkcfg/util"

function withCustomIngressHost() {
    return valuesMutator(function(values) {
        values.ingress.hosts = [
            "dashboard." + values.workshopctl.domain
        ]
        return values
    })
}

valuesPipeForFunctions([
    withCustomIngressHost(),
])