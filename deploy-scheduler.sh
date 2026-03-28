#!/bin/bash

make WHAT=cmd/kube-scheduler KUBE_BUILD_PLATFORMS=linux/arm64 
mv _output/local/bin/linux/arm64/kube-scheduler kdaptManifests/kube-scheduler

cd kdaptManifests

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t kvv1618/kdapt-scheduler:latest \
  -f kdapt.Dockerfile \
  --push .
kubectl rollout restart deployment kdapt-scheduler -n kube-system
