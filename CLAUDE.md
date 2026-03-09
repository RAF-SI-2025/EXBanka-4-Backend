# EXBanka-4-Backend

## Project overview
Go-based microservices backend for EXBanka. Each service is independently deployable with its own database.

## Go module
- Module path: `github.com/exbanka/backend`
- `go.mod` and `go.sum` live at the repo root (single-module monorepo)
- Framework: [Gin](https://github.com/gin-gonic/gin)

## Repository structure
```
services/        # One subdirectory per microservice
shared/          # Cross-service libraries and utilities
config/          # Environment-specific configuration
deploy/          # Kubernetes / Helm / Docker deployment manifests
scripts/         # Dev, CI/CD, and ops utility scripts
docs/            # Architecture docs, API contracts, runbooks
```

## Service conventions
Each service under `services/<service-name>/` follows this layout:
```
db/                  # SQL schema files
handlers/            # HTTP handlers (Gin)
models/              # Data models / structs
docker-compose.yml   # Isolated PostgreSQL container for this service
main.go              # Entry point
```

## Database
- Database-per-service pattern: every service has its own PostgreSQL instance via Docker Compose.
- Schema is defined in `db/schema.sql` and auto-applied on first container startup via `/docker-entrypoint-initdb.d/`.
- No `CREATE DATABASE` statement needed in SQL files — the database is created by the `POSTGRES_DB` env var in `docker-compose.yml`.

## Running a service database
```bash
cd services/<service-name>
docker compose up -d
```
