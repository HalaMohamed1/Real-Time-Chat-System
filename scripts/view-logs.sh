#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: $0 <service_name>"
  echo "Available services: app, gateway, postgres, redis, mqtt"
  exit 1
fi

SERVICE=$1
CONTAINER_NAME="rtcs_${SERVICE}_1"

if ! docker ps -q -f name=$CONTAINER_NAME | grep -q .; then
  echo "Container $CONTAINER_NAME not found"
  echo "Available containers:"
  docker ps --format "{{.Names}}"
  exit 1
fi

docker logs -f $CONTAINER_NAME