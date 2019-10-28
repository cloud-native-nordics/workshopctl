import { kubePipeWithMutators, withNamespace } from "../../jkcfg/util"

kubePipeWithMutators([
    withNamespace("workshopctl"),
])