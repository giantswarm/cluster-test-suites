FROM golang:1.25 AS build-tests

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

COPY --from=build-tests /app /app
COPY --from=build-tests /go/bin/ginkgo /usr/local/bin/ginkgo

ENTRYPOINT ["/app/entrypoint.sh"]
