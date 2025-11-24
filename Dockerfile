FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

# https://docs.railway.com/guides/dockerfiles#cache-mounts
# IMPORTANT: GO TO YOUR RAILWAY PROJECT SETTING TO FIND THE ID AND REPLACE WITH THIS
RUN --mount=type=cache,id=s/22a828da-b3c4-4c0a-902f-be8474c6ac94-/go/pkg/mod,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,id=s/22a828da-b3c4-4c0a-902f-be8474c6ac94-/go/pkg/mod,target=/go/pkg/mod \
    --mount=type=cache,id=s/22a828da-b3c4-4c0a-902f-be8474c6ac94-/root/.cache/go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main ./cmd/server/main.go

# https://render.com/docs/docker
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .

# same port as the app
EXPOSE 8080

CMD ["./main"]
