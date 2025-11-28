FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

# https://docs.railway.com/guides/dockerfiles#cache-mounts
# IMPORTANT: for e.g like the link of your service is
# https://railway.com/project/<PROJECT_ID>/service/<SERVICE_ID>?environmentId=<ENV_ID>
# then the cache id should be <SERVICE_ID>
# I know this is really bad practice but it's just for demo and save my time, this is not a sensitive information
RUN --mount=type=cache,id=s/e1470051-aa02-4bac-982d-b1712ddbfc44-/go/pkg/mod,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,id=s/e1470051-aa02-4bac-982d-b1712ddbfc44-/go/pkg/mod,target=/go/pkg/mod \
    --mount=type=cache,id=s/e1470051-aa02-4bac-982d-b1712ddbfc44-/root/.cache/go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main ./cmd/server/main.go

# https://render.com/docs/docker
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .

# same port as the app
EXPOSE 8080

CMD ["./main"]
