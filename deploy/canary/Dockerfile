FROM golang:latest
MAINTAINER vmarmol@google.com

RUN apt-get install -y git dmsetup
RUN git clone https://github.com/google/cadvisor.git /go/src/github.com/google/cadvisor
RUN go get github.com/Masterminds/glide && cd /go/src/github.com/Masterminds/glide && make bootstrap && make build
RUN cd /go/src/github.com/google/cadvisor && GO15VENDOREXPERIMENT=1 go build .

ENTRYPOINT ["/go/src/github.com/google/cadvisor/cadvisor"]

