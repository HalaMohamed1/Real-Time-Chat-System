#!/bin/bash

echo "Running integration tests..."
go test -v ./test/integration/...

if [ $? -eq 0 ]; then
  echo "Integration tests passed!"
  exit 0
else
  echo "Integration tests failed!"
  exit 1
fi