#!/bin/bash

# Parameters from Environment
: "${PROJECT:?Need to export PROJECT}"
: "${TAG:?Need to export TAG}"

for TYPE in "client" "server"
do
  docker build \
  --tag=gcr.io/${PROJECT}/${TYPE}:${TAG} \
  --file=./deployment/Dockerfile.${TYPE} \
  .
done
