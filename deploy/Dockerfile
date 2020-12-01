FROM alpine:3.12 AS build

RUN apk --no-cache add libc6-compat device-mapper findutils zfs  build-base linux-headers go bash git wget && \
    apk --no-cache add thin-provisioning-tools --repository http://dl-3.alpinelinux.org/alpine/edge/main/ && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \
    rm -rf /var/cache/apk/*

RUN wget https://sourceforge.net/projects/perfmon2/files/libpfm4/libpfm-4.11.0.tar.gz && \
  echo "112bced9a67d565ff0ce6c2bb90452516d1183e5  libpfm-4.11.0.tar.gz" | sha1sum -c  && \
  tar -xzf libpfm-4.11.0.tar.gz && \
  rm libpfm-4.11.0.tar.gz

RUN export DBG="-g -Wall" && \
  make -e -C libpfm-4.11.0 && \
  make install -C libpfm-4.11.0

ADD . /go/src/github.com/google/cadvisor
WORKDIR /go/src/github.com/google/cadvisor

ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV GO_FLAGS="-tags=libpfm,netgo"
RUN ./build/build.sh


FROM alpine:3.12
MAINTAINER dengnan@google.com vmarmol@google.com vishnuk@google.com jimmidyson@gmail.com stclair@google.com

RUN apk --no-cache add libc6-compat device-mapper findutils zfs && \
    apk --no-cache add thin-provisioning-tools --repository http://dl-3.alpinelinux.org/alpine/edge/main/ && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \
    rm -rf /var/cache/apk/*

# Grab cadvisor and libpfm4 from "build" container.
COPY --from=build /usr/local/lib/libpfm.so* /usr/local/lib/
COPY --from=build /go/src/github.com/google/cadvisor/cadvisor /usr/bin/cadvisor

EXPOSE 8080

ENV CADVISOR_HEALTHCHECK_URL=http://localhost:8080/healthz

HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --quiet --tries=1 --spider $CADVISOR_HEALTHCHECK_URL || exit 1

ENTRYPOINT ["/usr/bin/cadvisor", "-logtostderr"]
