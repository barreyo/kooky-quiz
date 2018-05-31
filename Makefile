
# General
PROJECT_NAME 		?= kooky-quiz
GIT_VERSION 		:= $(shell git describe --long --dirty --abbrev=10 --always --tags)

# Docker
DOCKER_GOPATH 		?= /go
DOCKER_WORKSPACE 	?= $(DOCKER_GOPATH)/src/github.com/barreyo/kooky-quiz
IMAGE_NAME  		?= kooky-base
SERVICE_IMAGE_NAME	?= kooky-service
IMAGE_VERSION		?= latest

# Protocol buffers
PROTO_DIR			?= pb

# Services
SERVICES_DIR		?= services
SERVICES			?= game_session redis ingress
SERVICE_NU			:= $(shell echo $(SERVICE) | tr - _ )
SERVICE_ND			:= $(shell echo $(SERVICE) | tr _ - )

# Formatting variables
BOLD				?= $(tput bold)
NORMAL				?= $(tput sgr0)

# Set the default command that will be passed to run target.
# This can be customized for example: CMD=ls make docker-run. Now 'ls' will be
# run instead of putting the user straight into the shell. Mostly there for use
# in a CI system.
ifndef CMD
CMD = "/bin/ash"
endif

.PHONY: docker-image docker-run build build-service deploy-service version \
		proto lint clean docker-clean help

docker-image: ## Building base images for GO services (building/running), also used for running tests and tools
	@echo $(BOLD)"--- Building base build image"$(NORMAL)
	docker build -t $(IMAGE_NAME):$(IMAGE_VERSION) .
	@echo $(BOLD)"--- Building base runtime image"$(NORMAL)
	docker build -t $(SERVICE_IMAGE_NAME):$(IMAGE_VERSION) -f Dockerfile.service .

docker-run: ## Run the base image with a CMD, otherwise drops into shell
	docker run -it --rm -v $(PWD):$(DOCKER_WORKSPACE) $(IMAGE_NAME):$(IMAGE_VERSION) $(CMD)

build: proto docker-image ## Build all docker images
	@$(foreach SERVICE, $(SERVICES), echo "-- Building ${SERVICE}" && \
		cd $(SERVICES_DIR)/$(SERVICE) && \
		docker build -t $(SERVICE):$(IMAGE_VERSION) . && cd - ;)

build-service:
ifeq ($(filter $(SERVICE_NU),$(SERVICES)),)
	$(error SERVICE env variable needs to be set to any of: $(SERVICES))
endif
	@echo "-- Building ${SERVICE}"
	cd $(SERVICES_DIR)/$(shell echo $(SERVICE) | tr - _) && \
		docker build -t $(shell echo $(SERVICE) | tr - _):$(IMAGE_VERSION) .

deploy-service:
ifeq ($(filter $(SERVICE_NU),$(SERVICES)),)
	$(error SERVICE env variable needs to be set to any of: $(SERVICES))
endif
	@echo "-- Deploying ${SERVICE}"
	@kubectl delete -f $(SERVICES_DIR)/$(SERVICE_NU)/k8s/$(SERVICE_ND)-service.yaml \
		-f $(SERVICES_DIR)/$(SERVICE_NU)/k8s/$(SERVICE_ND)-deployment.yaml || :
	@kubectl create -f $(SERVICES_DIR)/$(SERVICE_NU)/k8s/$(SERVICE_ND)-service.yaml \
		-f $(SERVICES_DIR)/$(SERVICE_NU)/k8s/$(SERVICE_ND)-deployment.yaml

version: ## Print the current version
	@echo $(GIT_VERSION)

proto: ## Build all Proto definitions
	@echo $(BOLD)"--- Generating Proto files"$(NORMAL)
	protoc -I $(PROTO_DIR)/ --go_out=plugins=grpc:$(PROTO_DIR)/ $(PROTO_DIR)/*.proto

lint: ## Run linter on protos and source files
	@echo $(BOLD)"--- Linting Proto files in ${PROTO_DIR}/"$(NORMAL)
	@protoc --lint_out=. $(PROTO_DIR)/*.proto
	@echo $(BOLD)"--- Linting GO source files in ${SERVICES_DIR}/"$(NORMAL)
	@golint -set_exit_status $(SERVICES_DIR)/...

clean: docker-clean ## Clean up all dependencies, docker stuff and output files
	rm -rf $(PROTO_DIR)/*.pb.go
	go clean

docker-clean:
	docker ps -a -q | xargs docker rm
	docker images -q | xargs docker rmi
	docker rm $(docker ps -a -q)
	docker rmi $(docker images -q -f dangling=true)

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
