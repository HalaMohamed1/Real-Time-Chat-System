#!/bin/bash

mkdir -p certs
cd certs

if [[ -f "server.key" && -f "server.crt" ]]; then
    echo "Certificates already exist."
else
    echo "Generating new certificates..."
    openssl genrsa -out server.key 2048
    openssl req -new -x509 -sha256 -key server.key -out server.crt -days 365 -subj "/CN=localhost"
    chmod 644 server.crt
    chmod 600 server.key
    echo "Certificates generated and permissions set."
fi
