#!/usr/bin/env bash

curl 'https://stage.metakube.de/api/v1/upgrades/cluster?type=kubernetes' -H "Authorization: Bearer ${METAKUBE_TOKEN}"
