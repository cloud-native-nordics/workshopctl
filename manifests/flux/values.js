import { valuesPipeForFunctions, valuesMutator } from "../../jkcfg/util"

function withClusterNumber() {
    return valuesMutator(function(values) {
        values.git.url = values.workshopctl.gitRepo
        values.git.path = "clusters/" + values.workshopctl.clusterNumber + "/"
        return values
    })
}

valuesPipeForFunctions([
    withClusterNumber(),
])