FROM gliderlabs/alpine:3.1
MAINTAINER dengnan@google.com vmarmol@google.com vishnuk@google.com

# Grab cadvisor from the staging directory.
ADD cadvisor /usr/bin/cadvisor

EXPOSE 8080
ENTRYPOINT ["/usr/bin/cadvisor"]
