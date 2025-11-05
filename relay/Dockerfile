FROM bufbuild/buf AS protobuf-builder
WORKDIR /app
ADD ../api ./api
RUN cd api \
    && buf dep update \
    && buf lint \
    && buf generate

FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY ../logger ./logger
COPY ../go.mod go.sum ./
COPY ../Makefile Makefile
COPY --from=protobuf-builder /app/api ./api
COPY relay/cmd/ relay/cmd
COPY relay/pkg relay/pkg
RUN make build.relay

#FROM debian:bookworm-slim
#COPY --from=builder /app/.bin/discovery-agent /usr/local/bin/discovery-agent
#COPY --from=builder /app/.bin/api-server /usr/local/bin/api-server


