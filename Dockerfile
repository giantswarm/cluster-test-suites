FROM golang:1.18

WORKDIR /app

RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

ADD . .

RUN go mod tidy

ENTRYPOINT ["/app/entrypoint.sh"]
