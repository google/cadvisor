FROM scratch
MAINTAINER dengnan@google.com vmarmol@google.com

# NOTE: Run prepare.sh before building this Docker image.

# Grab lmctfy and cadvisor from the staging directory.
ADD lmctfy /usr/bin/lmctfy
ADD cadvisor /usr/bin/cadvisor

EXPOSE 8080
ENTRYPOINT ["/usr/bin/cadvisor"]
