#!/usr/bin/env bash

curl 'https://metakube.syseleven.de/api/v1/upgrades/cluster?type=kubernetes' -H "Authorization: Bearer ${METAKUBE_TOKEN}"
