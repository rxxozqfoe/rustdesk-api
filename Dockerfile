# syntax=docker/dockerfile:1.22@sha256:4a43a54dd1fedceb30ba47e76cfcf2b47304f4161c0caeac2db1c61804ea3c91

# ---- Build stage ----
# CGO is required by github.com/mattn/go-sqlite3. We build a fully static
# binary against musl so the runtime image can still be distroless/static.
FROM golang:1.25.9-alpine@sha256:7a00384194cf2cb68924bbb918d675f1517357433c8541bac0ab2f929b9d5447 AS builder

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
FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1

WORKDIR /app

COPY --from=builder /out/apimain      /app/apimain
COPY --from=builder /src/resources    /app/resources
COPY --from=builder /src/conf         /app/conf
COPY --from=builder /src/docs         /app/docs

USER nonroot:nonroot
EXPOSE 21114
VOLUME ["/app/data", "/app/runtime"]

ENTRYPOINT ["/app/apimain"]
