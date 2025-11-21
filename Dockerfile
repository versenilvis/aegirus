FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# for cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server/main.go

FROM alpine:latest

# install SSL cert (for HTTPS to Google/Target) and timezone
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .

# OPTIONAL: copy .env file instead of using cloud's env
# WARN: PLEASE CAREFUL WITH IT
# COPY .env .

# same port as your server
EXPOSE 8080

CMD ["./main"]
