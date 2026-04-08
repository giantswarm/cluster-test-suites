FROM curlimages/curl:latest AS crust-gather
ARG CRUST_GATHER_VERSION=0.13.0
RUN curl -sSfL "https://github.com/crust-gather/crust-gather/releases/download/v${CRUST_GATHER_VERSION}/kubectl-crust-gather_${CRUST_GATHER_VERSION}_linux_amd64.tar.gz" \
    | tar -xz -C /tmp

FROM golang:1.26 AS build-tests

WORKDIR /app

RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

ADD go.mod go.sum ./

RUN go mod download

ADD . .

RUN ginkgo build --skip-package /X -r ./

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update \
  && apt-get install --no-install-recommends --no-install-suggests -y ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY --from=crust-gather /tmp/kubectl-crust-gather /usr/local/bin/crust-gather
COPY --from=build-tests /app /app
COPY --from=build-tests /go/bin/ginkgo /usr/local/bin/ginkgo

ENTRYPOINT ["/app/entrypoint.sh"]
