#!/bin/bash

set -e
set -x

godep go build -a github.com/google/cadvisor

sudo docker build -t google/cadvisor:test .
