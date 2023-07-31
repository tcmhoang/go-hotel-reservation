SHELL := /bin/bash

# Container

all: sales-api

VERSION := 0.1

sales-api:
	docker build \
		-f zarf/docker/dockerfile.sales-api \
		-t sales-api-amd64:${VERSION} \
		--build-arg BUILD_REF=${VERSION} \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-$dT%H:%M:%SZ"` \
		.


# Running within k8s

KIND_CLUSTER := sales-cluster

kind-up:
	kind create cluster \
		--image kindest/node:v1.27.2@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72 \
		--name ${KIND_CLUSTER} \
		--config zarf/k8s/kind/config.yaml
	kubectl config set-context --current --namespace=sales-sys

kind-down: 
	kind delete cluster --name ${KIND_CLUSTER}


kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

kind-status-sales:
	kubectl get pods -o wide --watch --all-namespaces 

kind-load:
	cd zarf/k8s/kind/sales-pod; kustomize edit set image sales-api-image=sales-api-amd64:${VERSION}
	kind load docker-image sales-api-amd64:${VERSION} --name ${KIND_CLUSTER}

kind-apply:
	kustomize build zarf/k8s/kind/sales-pod | kubectl apply -f -

kind-logs:
	kubectl logs -l app=sales --all-containers=true -f --tail=100 

kind-restart:
	kubectl rollout restart deployment sales-pod 

kind-update: all kind-load kind-restart

kind-update-apply: all kind-load kind-apply

kind-describe:
	kubectl describe pod -l app=sales 

# Mod support

tidy:
	go mod tidy
	go mod vendor

