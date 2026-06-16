# Pin the build stage to the host platform so its RUN steps never run under
# QEMU emulation. Cross-compilation is driven by TARGETOS/TARGETARCH instead.
FROM --platform=$BUILDPLATFORM golang:1.26 AS build-tests

ARG TARGETOS
ARG TARGETARCH
ARG CRUST_GATHER_VERSION=0.15.1

RUN curl -sSfL \
    "https://github.com/crust-gather/crust-gather/releases/download/v${CRUST_GATHER_VERSION}/kubectl-crust-gather_${CRUST_GATHER_VERSION}_linux_${TARGETARCH}.tar.gz" \
    | tar -xz -C /tmp

WORKDIR /app

# ginkgo here runs natively (host arch) to orchestrate the build.
RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

ADD go.mod go.sum ./

RUN go mod download

ADD . .

# GOOS/GOARCH make the underlying `go test -c` cross-compile the .test binaries
# for the target architecture, without emulating ginkgo itself.
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} ginkgo build --skip-package /X -r ./

# Cross-build the ginkgo runner that ships in the final image (must be target arch).
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/ginkgo github.com/onsi/ginkgo/v2/ginkgo

FROM debian:bookworm-slim

WORKDIR /app

# Copy the CA bundle from the build stage instead of `apt-get install`, so the
# target stage has no RUN step and is never emulated.
COPY --from=build-tests /etc/ssl/certs /etc/ssl/certs

COPY --from=build-tests /tmp/kubectl-crust-gather /usr/local/bin/crust-gather
COPY --from=build-tests /app /app
COPY --from=build-tests /out/ginkgo /usr/local/bin/ginkgo

ENTRYPOINT ["/app/entrypoint.sh"]
