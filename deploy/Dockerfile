FROM alpine:3.15 AS build

RUN apk --no-cache add libc6-compat device-mapper findutils zfs build-base linux-headers python3 bash git wget cmake pkgconfig ndctl-dev && \
    apk --no-cache add thin-provisioning-tools --repository http://dl-3.alpinelinux.org/alpine/edge/main/ && \
    # apk --no-cache add go==1.18 --repository http://dl-3.alpinelinux.org/alpine/v3.14/community/ && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \
    rm -rf /var/cache/apk/*

RUN wget -O - https://golang.org/dl/go1.18.linux-amd64.tar.gz \
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.linux-amd64.tar.gz \
    export PATH=$PATH:/usr/local/go/bin \
    go version

RUN wget https://sourceforge.net/projects/perfmon2/files/libpfm4/libpfm-4.11.0.tar.gz && \
    echo "112bced9a67d565ff0ce6c2bb90452516d1183e5  libpfm-4.11.0.tar.gz" | sha1sum -c  && \
    tar -xzf libpfm-4.11.0.tar.gz && \
    rm libpfm-4.11.0.tar.gz

RUN export DBG="-g -Wall" && \
    make -e -C libpfm-4.11.0 && \
    make install -C libpfm-4.11.0

RUN git clone -b v02.00.00.3820 https://github.com/intel/ipmctl/ && \
    cd ipmctl && \
    mkdir output && \
    cd output && \
    cmake -DRELEASE=ON -DCMAKE_INSTALL_PREFIX=/ -DCMAKE_INSTALL_LIBDIR=/usr/local/lib .. && \
    make -j all && \
    make install

ADD . /go/src/github.com/google/cadvisor
WORKDIR /go/src/github.com/google/cadvisor

ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV GO_FLAGS="-tags=libpfm,netgo,libipmctl"

RUN ./build/build.sh

FROM alpine:3.15
MAINTAINER dengnan@google.com vmarmol@google.com vishnuk@google.com jimmidyson@gmail.com stclair@google.com devesh.kumar@lookout.com

RUN apk --no-cache add libc6-compat device-mapper findutils zfs ndctl && \
    apk --no-cache add thin-provisioning-tools --repository http://dl-3.alpinelinux.org/alpine/edge/main/ && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \
    rm -rf /var/cache/apk/*

# Grab cadvisor,libpfm4 and libipmctl from "build" container.
COPY --from=build /usr/local/lib/libpfm.so* /usr/local/lib/
COPY --from=build /usr/local/lib/libipmctl.so* /usr/local/lib/
COPY --from=build /go/src/github.com/google/cadvisor/cadvisor /usr/bin/cadvisor

EXPOSE 8080

ENV CADVISOR_HEALTHCHECK_URL=http://localhost:8080/healthz

HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget --quiet --tries=1 --spider $CADVISOR_HEALTHCHECK_URL || exit 1

ENTRYPOINT ["/usr/bin/cadvisor", "-logtostderr"]
