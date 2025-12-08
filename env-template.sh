#!/usr/bin/env bash

###
#   The environment variables which need to be set to run acceptance tests.
#   Fill in the empty values.
###

export METAKUBE_TOKEN="" # service account credential on NCS
export METAKUBE_HOST="" # acceptance tests run against prod in the CI pipeline
export METAKUBE_NCS_PROJECT_ID="" # metakube / openstack project id (same on NCS)
export METAKUBE_SERVICE_ACCOUNT_CREDENTIAL="" # same as METAKUBE_TOKEN on NCS
export METAKUBE_NCS_OPENSTACK_REGION="" # dus2 or ham1

export TF_ACC=1

export METAKUBE_K8S_VERSION_OS=$(./scripts/get_versions.sh openstack | jq -r '.[] | select(.default == true) | .version')
export METAKUBE_K8S_OLDER_VERSION=$(./scripts/get_versions.sh openstack | jq -r 'map(select(.default == null)) | last | .version')

export METAKUBE_OPENSTACK_AUTH_URL="https://keystone.cloud.syseleven.net:5000/v3"
export METAKUBE_NCS_OPENSTACK_NODE_DC="syseleven-${METAKUBE_NCS_OPENSTACK_REGION}"
export METAKUBE_OPENSTACK_IMAGE="24.04"
export METAKUBE_OPENSTACK_IMAGE2="22.04"
export METAKUBE_OPENSTACK_FLAVOR="m2.small"
