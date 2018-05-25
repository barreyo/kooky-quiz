
FROM golang:1.10.2-alpine3.7

ENV WORKSPACE ${GOPATH}/src/github.com/barreyo/kooky-quiz
COPY . ${WORKSPACE}

WORKDIR ${WORKSPACE}
