#!/usr/bin/env bash

curl 'https://stage.metakube.de/api/v1/providers/openstack/images' -H 'Domain: Default' -H "Username: ${METAKUBE_OPENSTACK_USERNAME}" -H "Password: ${METAKUBE_OPENSTACK_PASSWORD}" -H "Tenant: ${METAKUBE_OPENSTACK_TENANT}" -H "DatacenterName: ${METAKUBE_OPENSTACK_NODE_DC}" -H "Authorization: Bearer ${METAKUBE_TOKEN}"

