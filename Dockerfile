FROM golang:1.23 as builder

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/gateway ./cmd/gateway

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/gateway /app/gateway

CMD ["/app/gateway"]
