FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

# https://docs.railway.com/guides/dockerfiles#cache-mounts
# IMPORTANT: for e.g like the link of your service is
# https://railway.com/project/<PROJECT_ID>/service/<SERVICE_ID>?environmentId=<ENV_ID>
# then the cache id should be <SERVICE_ID>
# I know this is really bad practice but it's just for demo and save my time, this is not a sensitive information
RUN --mount=type=cache,id=s/e59ccc16-aabd-4d76-947b-eaec4bb2d202-/go/pkg/mod,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,id=s/e59ccc16-aabd-4d76-947b-eaec4bb2d202-/go/pkg/mod,target=/go/pkg/mod \
    --mount=type=cache,id=s/e59ccc16-aabd-4d76-947b-eaec4bb2d202-/root/.cache/go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main ./cmd/aegirus/main.go


FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/internal/modules/botchallenge/templates ./internal/modules/botchallenge/templates

EXPOSE 8080

CMD ["./main"]
