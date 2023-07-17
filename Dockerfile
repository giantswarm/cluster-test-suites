FROM golang:1.20

WORKDIR /app

RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

ADD go.mod go.sum ./

RUN go mod tidy

ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build -o standup ./cmd/standup/
RUN CGO_ENABLED=0 GOOS=linux go build -o teardown ./cmd/teardown/

RUN ginkgo build --skip-package /X -r ./

ENTRYPOINT ["/app/entrypoint.sh"]
