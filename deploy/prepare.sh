#!/bin/bash

set -e
set -x

# Download lmctfy.
wget http://storage.googleapis.com/cadvisor-bin/lmctfy/lmctfy
chmod +x lmctfy

# Statically build cAdvisor from source and stage it.
go build --ldflags '-extldflags "-static"' github.com/google/cadvisor
