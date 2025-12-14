#!/bin/sh

# Default port
PORT=8080

# Extract port from the cadvisor process command line
if [ -f /proc/1/cmdline ]; then
    CMDLINE=$(tr '\0' ' ' < /proc/1/cmdline)

    # Look for -port=XXXX or --port=XXXX
    for arg in $CMDLINE; do
        case "$arg" in
            -port=*)
                PORT="${arg#-port=}"
                ;;
            --port=*)
                PORT="${arg#--port=}"
                ;;
        esac
    done
fi

wget --quiet --tries=1 --spider "http://localhost:${PORT}/healthz" || exit 1