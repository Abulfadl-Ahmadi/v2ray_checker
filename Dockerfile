FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /v2ray_checker main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /v2ray_checker /app/v2ray_checker
COPY config.yaml /app/config.yaml
COPY channels.csv /app/channels.csv

VOLUME ["/app/data"]
EXPOSE 8080

ENTRYPOINT ["/app/v2ray_checker"]
CMD ["-config", "config.yaml"]
