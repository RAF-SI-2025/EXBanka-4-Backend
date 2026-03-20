# EXBanka-4-Backend

## Project overview
Go-based microservices backend for EXBanka. Services communicate via gRPC. The API Gateway is the only HTTP-facing service.

## Go modules
- Multi-module workspace (`go.work` at repo root)
- One `go.mod` per service: `services/<name>/go.mod`
- Shared protobuf bindings: `shared/go.mod` → `github.com/RAF-SI-2025/EXBanka-4-Backend/shared`
- Service module paths: `github.com/RAF-SI-2025/EXBanka-4-Backend/services/<name>`

## Repository structure
```
services/        # One subdirectory per microservice
shared/          # Protobuf definitions and generated Go bindings
config/          # Environment-specific configuration (placeholder)
deploy/          # Kubernetes / Helm / Docker manifests (placeholder)
scripts/         # Dev and ops scripts
docs/            # Architecture docs, runbooks (placeholder)
```

## Services

| Service | Port | Protocol | Database |
|---|---|---|---|
| `employee-service` | 50051 | gRPC | PostgreSQL 16 on 5433 |
| `auth-service` | 50052 | gRPC | PostgreSQL 16 on 5434 |
| `api-gateway` | 8081 | HTTP (Gin) | none |
| `email-service` | 50053 | gRPC | none (uses RabbitMQ 3 on 5672) |

## Service layout conventions
```
services/<name>/
  db/                  # SQL schema (not all services have a DB)
  handlers/            # gRPC or HTTP handler implementations
  models/              # Data structs (where needed)
  queue/               # RabbitMQ producer/consumer (email-service only)
  templates/           # Email HTML templates (email-service only)
  docker-compose.yml   # PostgreSQL or RabbitMQ container
  main.go              # Entry point
```

## Shared Protobuf
- Source definitions: `shared/proto/*.proto`
- Generated Go bindings (committed): `shared/pb/<service>/`
- After editing a `.proto` file, regenerate with:
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
protoc --go_out=shared/pb/<service> --go_opt=paths=source_relative \
       --go-grpc_out=shared/pb/<service> --go-grpc_opt=paths=source_relative \
       -I shared/proto shared/proto/<service>.proto
```
- The generated `*.pb.go` files have a `DO NOT EDIT` comment — this is expected; always regenerate via protoc, never hand-edit them.

## Database
- Database-per-service: every DB-backed service has its own PostgreSQL via Docker Compose.
- Schema in `db/schema.sql`, auto-applied on first container startup via `/docker-entrypoint-initdb.d/`.
- No `CREATE DATABASE` needed in SQL — handled by `POSTGRES_DB` env var.
- `auth-service` stores activation tokens and password-reset tokens in its own PostgreSQL instance (port 5434).

## Environment variables
A `.env` file at the repo root currently holds **email-service** variables:
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `FROM_EMAIL`
- `RABBITMQ_URL`

`JWT_SECRET` and `EMPLOYEE_DB_URL` are still hardcoded in their respective services and will be migrated to `.env` before production.

## Running the full stack
```bash
./scripts/dev.sh
```
This starts the employee database container (waits for readiness), then all Go services. Ctrl+C stops the Go processes; the database containers keep running.

To start only the infrastructure container for a service:
```bash
cd services/<service-name>
docker compose up -d
```
The email-service requires its own RabbitMQ container:
```bash
cd services/email-service
docker compose up -d
```

## API Gateway endpoints
Employee routes require `Authorization: Bearer <access_token>` with the `ADMIN` role.

CORS is enabled for `http://localhost:5173` and `http://localhost:3000` (GET, POST, PUT, DELETE, OPTIONS; credentials allowed).

| Method | Path | Auth |
|---|---|---|
| POST | `/login` | none |
| POST | `/refresh` | none |
| POST | `/auth/activate` | none |
| POST | `/auth/forgot-password` | none |
| POST | `/auth/reset-password` | none |
| GET | `/employees` | ADMIN |
| GET | `/employees/:id` | ADMIN |
| GET | `/employees/search?email=&ime=&prezime=&pozicija=` | ADMIN |
| POST | `/employees` | ADMIN |
| PUT | `/employees/:id` | ADMIN |
| GET | `/swagger/*any` | none |

## Auth
- JWT signed with HMAC-SHA256. Access tokens expire in 15 min, refresh tokens in 7 days.
- The JWT secret is currently hardcoded in `services/auth-service/handlers/grpc_server.go` and `services/api-gateway/middleware/auth.go` — move to environment variable before production.
- `ADMIN` role bypasses all other role checks.
- New employees are created with `aktivan=false` and no password; they cannot log in until activated.
