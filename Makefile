
PROJECT_NAME 		?= "kooky-quiz"
IMAGE_NAME  		?= "kooky-core"
GIT_VERSION 		:= $(shell git describe --long --dirty --abbrev=10 --always --tags)
DOCKER_GOPATH 		?= "/go"
DOCKER_WORKSPACE 	?= $(DOCKER_GOPATH)/src/github.com/barreyo/kooky-quiz

ifndef CMD
CMD = "/bin/ash"
endif

.PHONY: version

docker-image:
	docker build -t $(IMAGE_NAME):$(GIT_VERSION) .

docker-run:
	docker run -it --rm -v $(PWD):$(DOCKER_WORKSPACE) $(IMAGE_NAME):$(GIT_VERSION) $(CMD)

version:
	@echo $(GIT_VERSION)
