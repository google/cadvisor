FROM registry.hub.docker.com/library/golang:1.22-alpine3.20 AS build

# Install build depdencies for all supported arches
RUN apk --no-cache add bash build-base cmake device-mapper findutils git \
                       libc6-compat linux-headers ndctl-dev pkgconfig python3 wget zfs && \
    apk --no-cache add thin-provisioning-tools --repository http://dl-3.alpinelinux.org/alpine/edge/main/ && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \
    rm -rf /var/cache/apk/*

RUN wget https://sourceforge.net/projects/perfmon2/files/libpfm4/libpfm-4.13.0.tar.gz && \
  echo "bcb52090f02bc7bcb5ac066494cd55bbd5084e65 libpfm-4.13.0.tar.gz" | sha1sum -c  && \
  tar -xzf libpfm-4.13.0.tar.gz && \
  rm libpfm-4.13.0.tar.gz

RUN export DBG="-g -Wall" && \
  make -e -C libpfm-4.13.0 && \
  make install -C libpfm-4.13.0

WORKDIR /go/src/github.com/google/cadvisor

# Cache Golang Dependencies for faster incremental builds
ADD go.mod go.sum ./
RUN go mod download
ADD cmd/go.mod cmd/go.sum ./cmd/
RUN cd cmd && go mod download

ADD . .

ARG VERSION

# libipmctl only works on x86_64 CPUs.
RUN export GO_TAGS="libpfm,netgo"; \
    GO_FLAGS="-tags=$GO_TAGS" ./build/build.sh

FROM mirror.gcr.io/library/alpine:3.20
MAINTAINER dengnan@google.com vmarmol@google.com vishnuk@google.com jimmidyson@gmail.com stclair@google.com

RUN apk --no-cache add libc6-compat device-mapper findutils ndctl zfs && \
    apk --no-cache add thin-provisioning-tools --repository http://dl-3.alpinelinux.org/alpine/edge/main/ && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \
    rm -rf /var/cache/apk/*

# Grab cadvisor,libpfm4 and libipmctl from "build" container if they exist (libipmctl only works on amd64/x86_64).
COPY --from=build /usr/local/lib/libpfm.so* /usr/local/lib/
COPY --from=build /go/src/github.com/google/cadvisor/_output/cadvisor /usr/bin/cadvisor

EXPOSE 8080

ENV CADVISOR_HEALTHCHECK_URL=http://localhost:8080/healthz

HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --quiet --tries=1 --spider $CADVISOR_HEALTHCHECK_URL || exit 1

ENTRYPOINT ["/usr/bin/cadvisor", "-logtostderr"]
