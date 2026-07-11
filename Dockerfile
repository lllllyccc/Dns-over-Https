FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /doh-server ./cmd/doh-server

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /doh-server .
COPY config.example.yaml .
COPY blocklist.txt .

RUN mkdir -p data certs

EXPOSE 443 8443

ENTRYPOINT ["./doh-server"]
CMD ["config.yaml"]
