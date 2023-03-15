# Build the manager binary
FROM golang:1.19 as builder

WORKDIR /app

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.io
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 go build -a -o apiserver cmd/apiserver/main.go

FROM alpine:3.17.2
COPY --from=builder /app/apiserver /usr/local/bin/
WORKDIR /
CMD ["sh"]
