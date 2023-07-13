FROM golang:1.20

WORKDIR /app

RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

ADD . .

RUN go mod tidy

RUN ginkgo build --skip-package /X -r ./

ENTRYPOINT ["/app/entrypoint.sh"]
