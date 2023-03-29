#!/usr/bin/env bash

curl "https://metakube.syseleven.de/api/v2/providers/$1/versions" -H "Authorization: Bearer ${METAKUBE_TOKEN}"
