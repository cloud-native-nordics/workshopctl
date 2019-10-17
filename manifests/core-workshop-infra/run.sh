#!/bin/bash

if [[ ! -x node_modules ]]; then
    ln -s ../../node_modules/ .
fi

jk run -p clusterNumber=02 ../values.js | helm template workshopctl chart -f - | jk run ../pipe.js
