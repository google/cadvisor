#!/bin/bash

set -e
set -x

rice -i github.com/google/cadvisor/pages/static embed-go
godep go build -a github.com/google/cadvisor

docker build -t google/cadvisor:beta .
