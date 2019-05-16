#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

#
# Supported Targets:
#
# unit-test:                  runs unit tests
# lint:                       runs linters
# checks:                     runs code checks
# docker-thirdparty:          pulls thirdparty images (couchdb)
# populate-fixtures:          populate crypto directory and channel configuration for bddtests
# crypto-gen:                 generates crypto directory
# channel-config-gen:         generates test channel configuration transactions and blocks
# bddtests:                   run bddtests
#

ARCH=$(shell go env GOARCH)


# Tool commands (overridable)
DOCKER_CMD ?= docker

# Local variables used by makefile
PROJECT_NAME            = fabric-peer-ext
CONTAINER_IDS           = $(shell docker ps -a -q)
DEV_IMAGES              = $(shell docker images dev-* -q)
export GO111MODULE      = on

# Fabric tools docker image (overridable)
FABRIC_TOOLS_IMAGE   ?= hyperledger/fabric-tools
FABRIC_TOOLS_VERSION ?= 2.0.0-alpha
FABRIC_TOOLS_TAG     ?= $(ARCH)-$(FABRIC_TOOLS_VERSION)


checks: version license lint

lint:
	@scripts/check_lint.sh

license: version
	@scripts/check_license.sh

all: checks unit-test bddtests

unit-test: export FABRIC_COMMAND=unit-test
unit-test: checks docker-thirdparty
	@scripts/unit.sh
	@scripts/build_fabric.sh

bddtests: checks build-fabric-images populate-fixtures
	@scripts/integration.sh

build-fabric-images: export FABRIC_COMMAND=peer-docker orderer-docker ccenv
build-fabric-images:
	@scripts/build_fabric.sh
	@docker tag hyperledger/fabric-peer:latest     trustbloc/fabric-peer:latest
	@docker tag hyperledger/fabric-ccenv:latest    trustbloc/fabric-ccenv:latest
	@docker tag hyperledger/fabric-orderer:latest  trustbloc/fabric-orderer:latest


crypto-gen:
	@echo "Generating crypto directory ..."
	@$(DOCKER_CMD) run -i \
		-v /$(abspath .):/opt/workspace/$(PROJECT_NAME) -u $(shell id -u):$(shell id -g) \
		$(FABRIC_TOOLS_IMAGE):$(FABRIC_TOOLS_TAG) \
		//bin/bash -c "FABRIC_VERSION_DIR=fabric /opt/workspace/${PROJECT_NAME}/scripts/generate_crypto.sh"

channel-config-gen:
	@echo "Generating test channel configuration transactions and blocks ..."
	@$(DOCKER_CMD) run -i \
		-v /$(abspath .):/opt/workspace/$(PROJECT_NAME) -u $(shell id -u):$(shell id -g) \
		$(FABRIC_TOOLS_IMAGE):$(FABRIC_TOOLS_TAG) \
		//bin/bash -c "FABRIC_VERSION_DIR=fabric/ /opt/workspace/${PROJECT_NAME}/scripts/generate_channeltx.sh"

populate-fixtures:
	@scripts/populate-fixtures.sh

version:
	@scripts/check_version.sh

docker-thirdparty:
	docker pull couchdb:2.2.0

clean-images:
	@echo "Stopping all containers, pruning containers and images, deleting dev images"
ifneq ($(strip $(CONTAINER_IDS)),)
	@docker stop $(CONTAINER_IDS)
endif
	@docker system prune -f
ifneq ($(strip $(DEV_IMAGES)),)
	@docker rmi $(DEV_IMAGES) -f
endif
	@docker rmi $(docker images securekey/* -aq)

.PHONY: all version clean-images unit-test docker-thirdparty license bddtests build-fabric-images crypto-gen channel-config-gen populate-fixtures