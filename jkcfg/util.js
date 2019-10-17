import * as std from '@jkcfg/std';
import * as param from '@jkcfg/std/param';

export async function kubePipeWithMutators(mutators) {
    const resources = await(std.read('', { format: std.Format.YAMLStream }));
    resources.forEach((obj) => {
        if (obj === null) return;
        mutators.forEach(function(m){
            var meta = m.resourceFunc ? new m.resourceFunc() : null
            if (!meta || (meta.apiVersion == obj.apiVersion && meta.kind == obj.kind)) {
                if (!m.name || (m.name && m.name == obj.metadata.name)) {
                    obj = m.func(obj)
                }
            }
        })
    });
    std.log(resources, { format: std.Format.YAMLStream })
}

export async function valuesPipeForFunctions(mutators) {
    const clusterNumber = param.String("cluster-number", "01");
    const domain = param.String("domain", "kubernetesfinland.com");
    const gitRepo = param.String("git-repo", "https://github.com/luxas/workshopctl");
    const provider = param.String("provider", "digitalocean");
    const workshopctl = {
        clusterNumber,
        domain,
        gitRepo,
        provider,
    }
    std.read('values.yaml', { format: std.Format.YAMLStream }).then((valuesList) => {
        var values = valuesList[0] ? valuesList[0] : {};
        // Set the pre-made values
        values.workshopctl = workshopctl
        mutators.forEach((mutator) => {
            if (!mutator.func) return;
            values = mutator.func(values)
        });
        std.log([values], { format: std.Format.YAMLStream })
    }, () => {
        std.log([{workshopctl}], { format: std.Format.YAMLStream })
    })
}

export function withNamespace(ns) {
    return kubeMutator(null, null, function (obj) {
        obj.metadata.namespace = ns
        return obj
    })
}

export function valuesMutator(func) {
    return {
        func,
    }
}

export function kubeMutator(resourceFunc, name, func) {
    return {
        resourceFunc,
        name,
        func,
    }
}
