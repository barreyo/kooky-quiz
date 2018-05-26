
# Build runtime
FROM golang:alpine

ENV GOROOT /usr/local/go

RUN apk update && apk add git

ENV WORKSPACE ${GOPATH}/src/github.com/barreyo/kooky-quiz

WORKDIR ${WORKSPACE}

COPY . ${WORKSPACE}

RUN mkdir /build

# Fetch all dependencies, make sure they are available during build
RUN go get -d ./...
