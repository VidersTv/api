FROM golang:1.17.6 as builder

WORKDIR /tmp/api

COPY . .

ARG BUILDER
ARG VERSION

ENV API_BUILDER=${BUILDER}
ENV API_VERSION=${VERSION}

RUN apt-get update && apt-get install make git gcc -y && \
    make build_deps && \
    make

FROM alpine:latest

WORKDIR /app

COPY --from=builder /tmp/api/bin/api .

CMD ["/app/api"]
