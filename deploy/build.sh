#!/bin/bash

set -e
set -x

godep go build -a github.com/google/cadvisor

docker build -t google/cadvisor:beta .
