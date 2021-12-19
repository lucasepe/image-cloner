# Set the shell to bash always
SHELL := /bin/bash

# Variables
KIND_CLUSTER_NAME ?= local-dev
KUBECONFIG ?= $(HOME)/.kube/config

# Tools
KIND=$(shell which kind)
KUBECTL=$(shell which kubectl)


.DEFAULT_GOAL := help

.PHONY: help
## help: Print this help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECT_NAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

.PHONY: kind.up
## kind.up: Starts KinD cluster for local development
kind.up:
	@$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) >/dev/null 2>&1 || @$(shell ./kind-with-registry.sh)

## kind.down: Shuts down the KinD cluster
kind.down:
	@$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)
	@docker stop kind-registry || true
	@docker rm kind-registry || true

.PHONY: cr.login
## cr.login: Create the secret with the backup container registry credentials
cr.login:
	$(KUBECTL) apply -f namespace.yml
	$(KUBECTL) create secret generic image-cloner-cr-login \
		--namespace image-cloner-system \
		--from-file=IMAGE_CLONER_USER=cr-user.txt \
		--from-file=IMAGE_CLONER_PASS=cr-pass.txt || true

.PHONY: deploy
## deploy: Deploy the Image Cloner controller
deploy: cr.login
	$(KUBECTL) apply -f rbac.yml
	$(KUBECTL) apply -f image-cloner.yml
