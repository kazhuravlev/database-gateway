FROM golang:1.23 as builder

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/app ./cmd/app

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/app /app/app

CMD ["/app/app"]
