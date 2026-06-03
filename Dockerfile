# syntax=docker/dockerfile:1.22@sha256:4a43a54dd1fedceb30ba47e76cfcf2b47304f4161c0caeac2db1c61804ea3c91

# ---- Build stage ----
# CGO is required by github.com/mattn/go-sqlite3. We build a fully static
# binary against musl so the runtime image can still be distroless/static.
FROM golang:1.26-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache build-base musl-dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate Swagger docs. Worker controller is internal-only and excluded
# from both public API and admin doc bundles.
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.3 \
 && swag init -g cmd/apimain.go --output docs/api   --instanceName api   --exclude internal/http/controller/admin,internal/http/controller/worker \
 && swag init -g cmd/apimain.go --output docs/admin --instanceName admin --exclude internal/http/controller/api,internal/http/controller/worker

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
        -ldflags="-s -w -linkmode external -extldflags '-static'" \
        -o /out/apimain ./cmd/apimain.go

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot@sha256:d093aa3e30dbadd3efe1310db061a14da60299baff8450a17fe0ccc514a16639

WORKDIR /app

COPY --from=builder /out/apimain      /app/apimain
COPY --from=builder /src/resources    /app/resources
COPY --from=builder /src/conf         /app/conf
COPY --from=builder /src/docs         /app/docs

USER nonroot:nonroot
EXPOSE 21114
VOLUME ["/app/data", "/app/runtime"]

ENTRYPOINT ["/app/apimain"]
