FROM bufbuild/buf AS protobuf-builder
WORKDIR /app
ADD ../api ./api
ADD ../Makefile ./
RUN cd api \
    && buf dep update \
    && buf lint \
    && buf generate

FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY ../../go.mod go.sum ./
COPY ../../Makefile ./
COPY --from=protobuf-builder /app/api/ ./api/
COPY internal/applogger/ ./internal/applogger/
COPY cni/ ./cni/
COPY ../../relay ./relay/
RUN make build-relay

FROM debian:bookworm-slim
COPY ./dockerfiles/relay/socat-aarch64 /usr/local/bin/socat-aarch64
COPY --from=builder /app/bin/wafie-relay /usr/local/bin/wafie-relay
