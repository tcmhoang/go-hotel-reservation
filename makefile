SHELL := /bin/bash

# Container

run:
	go run main.go

all: service

VERSION := 0.1

service:
	docker build \
		-f zarf/docker/dockerfile \
		-t service-amd64:${VERSION} \
		--build-arg BUILD_REF=${VERSION} \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-$dT%H:%M:%SZ"` \
		.


# Running within k8s

KIND_CLUSTER := starter-cluster

kind-up:
	kind create cluster \
		--image kindest/node:v1.27.2@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72 \
		--name ${KIND_CLUSTER} \
		--config zarf/k8s/kind/config.yaml
	kubectl config set-context --current --namespace=service-sys

kind-down: 
	kind delete cluster --name ${KIND_CLUSTER}


kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

kind-status-service:
	kubectl get pods -o wide --watch --all-namespaces 

kind-load:
	kind load docker-image service-amd64:${VERSION} --name ${KIND_CLUSTER}

kind-apply:
	kustomize build zarf/k8s/kind/service-pod | kubectl apply -f -

kind-logs:
	kubectl logs -l app=service --all-containers=true -f --tail=100 

kind-restart:
	kubectl rollout restart deployment service-pod 

kind-update: all kind-load kind-restart

kind-update-apply: all kind-load kind-apply

kind-describe:
	kubectl describe pod -l app=service 

# Mod support

tidy:
	go mod tidy
	go mod vendor

