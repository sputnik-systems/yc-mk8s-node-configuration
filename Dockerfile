FROM golang:1.21 as builder

ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/sputnik-systems/yc-mk8s-node-configuration/
COPY . .
RUN go mod download -x
RUN go build ./cmd/containerd-registry-mirrors-updater
RUN go build ./cmd/iptables-rules-updater


FROM ubuntu

RUN apt update \
    && apt install -y iptables \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/src/github.com/sputnik-systems/yc-mk8s-node-configuration/containerd-registry-mirrors-updater /usr/local/bin/containerd-registry-mirrors-updater
COPY --from=builder /go/src/github.com/sputnik-systems/yc-mk8s-node-configuration/iptables-rules-updater /usr/local/bin/iptables-rules-updater
