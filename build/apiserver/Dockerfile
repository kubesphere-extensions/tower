# Build the manager binary
FROM golang:1.21 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 go build -a -o apiserver cmd/apiserver/main.go

FROM alpine:3.18.4
COPY --from=builder /app/apiserver /usr/local/bin/
WORKDIR /
CMD ["sh"]
