# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RustDesk API server implemented in Go — provides the backend for RustDesk remote desktop clients, a web admin panel, and a web client. Supports user management, address books, device groups, OAuth/OIDC/LDAP authentication, and audit logging.
The main branch is "main".

## Build & Development Commands

```bash
# Run directly
go run cmd/apimain.go

# Build (generates Swagger docs + compiles to release/)
make build
# Cross-compile: make build GOOS=linux GOARCH=arm64
# Skip swagger:  make build DOCS=

# Run tests
go test ./...

# Run a single test
go test ./internal/utils -run TestLoginLimiter

# Generate Swagger docs (requires swag: go install github.com/swaggo/swag/cmd/swag@latest)
swag init -g cmd/apimain.go --output docs/api --instanceName api --exclude internal/http/controller/admin
swag init -g cmd/apimain.go --output docs/admin --instanceName admin --exclude internal/http/controller/api

# Docker dev build
docker-compose -f docker-compose-dev.yaml up

# CLI commands
./release/apimain reset-admin-pwd <password>
./release/apimain reset-pwd <userId> <password>
```

## Architecture

**Layered structure:** `cmd/` → `internal/http/` (router/controller/middleware) → `internal/service/` → `internal/model/` (GORM)

All non-entrypoint Go code lives under `internal/` so nothing outside this module can import it. Only `cmd/` (main package) and `docs/` (swag-generated) sit at the repo root.

- **cmd/apimain.go** — Entry point using Cobra CLI. Builds the dependency graph in `InitApp()` then starts Gin via `apphttp.ApiInit(handlers)`.
- **internal/http/router/** — Three route groups: web (static UI), admin API (`/api/admin/*`), PC client API (`/api/*`). Routes defined in `router.go`, `admin.go`, `api.go`.
- **internal/http/controller/** — Handlers split into `admin/` (web admin), `api/` (PC client), and `web/` (web UI). Each controller typically maps 1:1 to a service.
- **internal/http/middleware/** — Auth (`RustAuth` for PC clients, `BackendUserAuth` for admin JWT), `AdminPrivilege`, rate limiting, logging, CORS.
- **internal/service/** — Business logic layer. Services initialized in `service.go` with shared dependencies (DB, Config, Logger, JWT, Lock).
- **internal/model/** — GORM models with auto-migration on startup. Schema versioning tracked via `Version` model (current: v265).
- **internal/app/** — AppContext, i18n localizer constructor, validator constructor. Pure constructors — no package-level globals.
- **internal/lib/** — Internal libraries: `cache/` (file/Redis), `orm/` (MySQL/PostgreSQL/SQLite drivers), `jwt/`, `logger/`, `lock/`, `upload/`.
- **internal/config/** — Viper-based config loading. Reads `conf/config.yaml` (or `conf/config.dev.yaml` if present). All values mappable to `RUSTDESK_API_*` env vars.
- **internal/utils/** — Shared helpers (captcha, login limiter, password hashing).
- **conf/** — Runtime config **files** (yaml, hello.html, keys). Not to be confused with `internal/config/` (the Go package). `conf/` is user-facing and shipped with deployments; `internal/config/` is code.

**Request/Response DTOs** live in `internal/http/request/` and `internal/http/response/`, separate from models.

## Key Conventions

- **Database:** Supports SQLite (default), MySQL, PostgreSQL — configured via `conf/config.yaml` under `gorm.type`. Auto-migration runs on startup.
- **Config:** Viper reads `conf/config.yaml` with env var overrides prefixed `RUSTDESK_API_` (e.g., `RUSTDESK_API_GIN_API_ADDR`).
- **Auth:** PC client API uses `RustAuth` middleware (header-based). Admin API uses JWT via `BackendUserAuth`. OAuth2/OIDC and LDAP also supported.
- **i18n:** Uses `go-i18n/v2` with translation files in `resources/i18n/`. Language set in config.
- **Swagger annotations:** Controllers use swag comment annotations. Two separate doc instances: `api` and `admin`.
- **Go version:** 1.23. CGO enabled (required for SQLite driver).
- **Port:** Default 21114, configured via `gin.api-addr`.
