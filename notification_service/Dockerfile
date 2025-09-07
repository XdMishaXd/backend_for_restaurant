FROM golang:alpine AS builder

LABEL image_author="Michael Prunchak"

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o notify_service ./cmd/notifications

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/notify_service .
COPY --from=builder /app/config ./config

CMD ["./notify_service"]