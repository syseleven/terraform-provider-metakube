DOMAIN=syseleven.de
NAMESPACE=syseleven
PKG_NAME=metakube
BINARY=terraform-provider-${PKG_NAME}
VERSION=0.4.4
PLATFORM?=$(shell uname -s | tr A-Z a-z)_$(shell uname -m)
SWEEP_DIR?=./metakube
SWEEP?=all

export GOPATH?=$(shell go env GOPATH)
export GOPROXY=https://proxy.golang.org
export GO111MODULE=on

default: install

build: fmtcheck
	go build -v -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${DOMAIN}/${NAMESPACE}/${PKG_NAME}/${VERSION}/${PLATFORM}
	mv ${BINARY} ~/.terraform.d/plugins/${DOMAIN}/${NAMESPACE}/${PKG_NAME}/${VERSION}/${PLATFORM}

reset:
	rm -rf terraform*
	rm -rf .terraform*
	make install
	terraform init 

test: fmtcheck
	go test ./$(PKG_NAME)

testacc:
# Require following environment variables to be set:
# METAKUBE_TOKEN - access token
# METAKUBE_HOST - example https://metakube.syseleven.de
# METAKUBE_ANOTHER_USER_EMAIL - email of an existing user to test cluster access sharing
# METAKUBE_K8S_VERSION - the kubernetes version
# METAKUBE_K8S_OLDER_VERSION - lower kubernetes version then METAKUBE_K8S_VERSION
# OS_PROJECT - Openstack project name
# OS_AUTH_URL - Openstack auth url
# OS_USERNAME - Openstack username
# OS_PASSWORD - Openstack password
# METAKUBE_OPENSTACK_IMAGE - an image available for openstack clusters
# METAKUBE_OPENSTACK_IMAGE2 - another image available for openstack clusters
# METAKUBE_OPENSTACK_FLAVOR - openstack flavor to use
# METAKUBE_OPENSTACK_USERNAME - openstack credentials username
# METAKUBE_OPENSTACK_PASSWORD - openstack credentials password
# METAKUBE_OPENSTACK_TENANT - openstack tenant to use
# METAKUBE_OPENSTACK_NODE_DC - openstack node datacenter name
# METAKUBE_AZURE_NODE_DC - azure node datacenter name
# METAKUBE_AZURE_NODE_SIZE
# METAKUBE_AZURE_CLIENT_ID
# METAKUBE_AZURE_CLIENT_SECRET
# METAKUBE_AZURE_TENANT_ID
# METAKUBE_AZURE_SUBSCRIPTION_ID
# METAKUBE_AWS_ACCESS_KEY_ID
# METAKUBE_AWS_ACCESS_KEY_SECRET
# METAKUBE_AWS_VPC_ID
# METAKUBE_AWS_NODE_DC
# METAKUBE_AWS_INSTANCE_TYPE
# METAKUBE_AWS_SUBNET_ID
# METAKUBE_AWS_AVAILABILITY_ZONE
# METAKUBE_AWS_DISK_SIZE
	TF_ACC=1 go test ./$(PKG_NAME) -v $(TESTARGS) -timeout 120m

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test $(SWEEP_DIR) -v -sweep=$(SWEEP) $(SWEEPARGS) -timeout 60m

vet:
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	@echo "==> Fixing source code with gofmt..."
	gofmt -s -w $(PKG_NAME)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

.PHONY: build install test testacc sweep vet fmt fmtcheck
