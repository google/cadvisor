FROM alpine:3.2
MAINTAINER dengnan@google.com vmarmol@google.com vishnuk@google.com jimmidyson@gmail.com

WORKDIR /gopath/src/github.com/google/cadvisor
COPY    . /gopath/src/github.com/google/cadvisor

RUN apk add --update --repository=http://dl-4.alpinelinux.org/alpine/edge/community/ -t build-deps go sqlite-dev btrfs-progs-dev bash linux-headers make git gcc g++ && \
    apk add --update device-mapper && \
    export GOPATH=/gopath && \
    export PATH=$GOPATH/bin:$PATH && \
    make && \
    mv cadvisor /usr/bin/cadvisor && \
    apk del --purge build-deps && \
    rm -rf /var/cache/apk/*

EXPOSE 8080
ENTRYPOINT ["/usr/bin/cadvisor", "-logtostderr"]
