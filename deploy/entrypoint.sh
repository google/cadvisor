#!/bin/sh
set -e

# Execute cadvisor with all provided arguments
exec /usr/bin/cadvisor -logtostderr "$@"