# Kooky Quiz
Fully web-based collaborative/competitive game platform.

# Project Goals

* Scale websocket connection services horizontally using a pub/sub model
* Stateless game backends
* Low latency connections, master and phone controllers
* No dependencies on CSP managed services
* Run on Raspberry Pi based k8s cluster :D

# Learning Goals

* Learn how to setup and securely manage k8s
* Trade offs and fundamentals of REST API and Websockets

# Running

**Requirements:**
* Docker
* Minikube
* Virtualbox / VMWare Fusion / Hyperkit

Setup once:

```
$ minikube start --vm-driver=hyperkit --disk-size=80g --memory 4096
$ sudo sed -i "$(minikube ip) dev.kooky.app" /etc/hosts
```

To build and deploy a service run:

```
$ SERVICE=game_session make build-service
$ SERVICE=game_session make deploy-service
```

There is currently no deploy script for all service and setup so to get the
whole thing setup run (Big TODO to fix this):

```
$ make build
$ kubectl create -f config/deployment/dev-config-map.yaml
$ kubectl create -f config/deployment/dev-secret.yaml
$ kubectl create -f services/redis/k8s/redis-deployment.yaml -f services/redis/k8s/redis-service.yaml
$ kubectl create secret tls tls-certificate --key config/dev-certs/server.key --cert config/dev-certs/server.crt
$ kubectl create -f services/ingress/k8s/ingress.yaml
$ SERVICE=game_session make build-service
$ SERVICE=game_session make deploy-service
```
