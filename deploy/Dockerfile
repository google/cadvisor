FROM alpine:3.10
MAINTAINER dengnan@google.com vmarmol@google.com vishnuk@google.com jimmidyson@gmail.com stclair@google.com

RUN apk --no-cache add libc6-compat device-mapper findutils zfs  build-base linux-headers && \
    apk --no-cache add thin-provisioning-tools --repository http://dl-3.alpinelinux.org/alpine/edge/main/ && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \
    rm -rf /var/cache/apk/*

RUN wget https://sourceforge.net/projects/perfmon2/files/libpfm4/libpfm-4.10.1.tar.gz && \
  tar -xzf libpfm-4.10.1.tar.gz && \
  rm libpfm-4.10.1.tar.gz

RUN export DBG="-g -Wall" && \
  make -e -C libpfm-4.10.1 && \
  make install -C libpfm-4.10.1

# Grab cadvisor from the staging directory.
ADD cadvisor /usr/bin/cadvisor

EXPOSE 8080

ENV CADVISOR_HEALTHCHECK_URL=http://localhost:8080/healthz

HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --quiet --tries=1 --spider $CADVISOR_HEALTHCHECK_URL || exit 1

ENTRYPOINT ["/usr/bin/cadvisor", "-logtostderr"]
