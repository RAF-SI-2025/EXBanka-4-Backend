# EXBanka-4-Backend

## Project overview
Go-based microservices backend for EXBanka. Services communicate via gRPC. The API Gateway is the only HTTP-facing service.

## Go modules
- Multi-module workspace (`go.work` at repo root)
- One `go.mod` per service: `services/<name>/go.mod`
- Shared protobuf bindings: `shared/go.mod` → `github.com/RAF-SI-2025/EXBanka-4-Backend/shared`
- Service module paths: `github.com/RAF-SI-2025/EXBanka-4-Backend/services/<name>`
- Modules in `go.work`: `./services/account-service`, `./services/api-gateway`, `./services/auth-service`, `./services/card-service`, `./services/client-service`, `./services/email-service`, `./services/employee-service`, `./services/exchange-service`, `./services/loan-service`, `./services/order-service`, `./services/payment-service`, `./services/portfolio-service`, `./services/securities-service`, `./shared`

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

| Service | gRPC Port | DB (PostgreSQL 16) |
|---|---|---|
| `employee-service` | 50051 | 5433 |
| `auth-service` | 50052 | 5434 |
| `email-service` | 50053 | none (RabbitMQ on 5672) |
| `account-service` | 50054 | 5436 |
| `payment-service` | 50055 | 5437 |
| `client-service` | 50056 | 5435 |
| `exchange-service` | 50057 | 5438 |
| `loan-service` | 50058 | 5439 |
| `card-service` | 50059 | 5440 |
| `securities-service` | 50060 | 5441 |
| `order-service` | 50061 | 5442 |
| `portfolio-service` | 50062 | 5443 |
| `api-gateway` | **8083** (HTTP) | none |

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
- Source definitions: `shared/proto/*.proto` — one file per service: `account.proto`, `auth.proto`, `card.proto`, `client.proto`, `email.proto`, `employee.proto`, `exchange.proto`, `loan.proto`, `order.proto`, `payment.proto`, `portfolio.proto`, `securities.proto`
- Generated Go bindings (committed): `shared/pb/<service>/`
- After editing a `.proto` file, regenerate with:
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
protoc --go_out=shared/pb/<service> --go_opt=paths=source_relative \
       --go-grpc_out=shared/pb/<service> --go-grpc_opt=paths=source_relative \
       -I shared/proto shared/proto/<service>.proto
```
- The generated `*.pb.go` files have a `DO NOT EDIT` comment — always regenerate via protoc, never hand-edit them.

## Database
- Database-per-service: every DB-backed service has its own PostgreSQL via Docker Compose.
- Schema in `db/schema.sql`, auto-applied on first container startup via `/docker-entrypoint-initdb.d/`.
- No `CREATE DATABASE` needed in SQL — handled by `POSTGRES_DB` env var.

## Environment variables
Service-to-service addresses (set by `dev.sh` or deployment):
- `EMPLOYEE_SERVICE_ADDR`, `AUTH_SERVICE_ADDR`, `CLIENT_SERVICE_ADDR`, `ACCOUNT_SERVICE_ADDR`
- `PAYMENT_SERVICE_ADDR`, `EXCHANGE_SERVICE_ADDR`, `LOAN_SERVICE_ADDR`, `CARD_SERVICE_ADDR`
- `SECURITIES_SERVICE_ADDR`, `ORDER_SERVICE_ADDR`, `PORTFOLIO_SERVICE_ADDR`, `EMAIL_SERVICE_ADDR`

Database URLs (one per service): `EMPLOYEE_DB_URL`, `AUTH_DB_URL`, `CLIENT_DB_URL`, `ACCOUNT_DB_URL`, `PAYMENT_DB_URL`, `EXCHANGE_DB_URL`, `LOAN_DB_URL`, `CARD_DB_URL`, `SECURITIES_DB_URL`, `ORDER_DB_URL`, `PORTFOLIO_DB_URL`

A `.env` file at the repo root holds **email-service** variables:
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `FROM_EMAIL`, `RABBITMQ_URL`

`JWT_SECRET` is still hardcoded in `services/auth-service/handlers/grpc_server.go` and `services/api-gateway/middleware/auth.go` — move to env before production.

## Running the full stack
```bash
./scripts/dev.sh
```
Starts all database containers (waits for readiness), RabbitMQ, then all 12 Go services. Ctrl+C stops the Go processes; containers keep running.

To reset all databases:
```bash
./scripts/reset-dbs.sh
```

To start only the infrastructure container for a service:
```bash
cd services/<service-name>
docker compose up -d
```

## API Gateway endpoints
Base URL: `http://localhost:8083`

CORS is enabled for `http://localhost:5173` and `http://localhost:3000` (GET, POST, PUT, DELETE, OPTIONS; credentials allowed).

### Auth (public)
| Method | Path |
|---|---|
| POST | `/login` |
| POST | `/refresh` |
| POST | `/client/login` |
| POST | `/client/refresh` |
| POST | `/auth/activate` |
| POST | `/auth/forgot-password` |
| POST | `/auth/reset-password` |

### Employees (ADMIN)
| Method | Path |
|---|---|
| GET | `/employees` |
| GET | `/employees/:id` |
| GET | `/employees/search?email=&ime=&prezime=&pozicija=` |
| POST | `/employees` |
| PUT | `/employees/:id` |

### Actuary limits (SUPERVISOR)
| Method | Path |
|---|---|
| GET | `/api/actuaries` |
| PUT | `/api/actuaries/:id/limit` |
| POST | `/api/actuaries/:id/reset-used-limit` |
| PUT | `/api/actuaries/:id/need-approval` |

### Clients (EMPLOYEE)
| Method | Path |
|---|---|
| GET | `/clients` |
| GET | `/clients/:id` |
| POST | `/clients` |
| PUT | `/clients/:id` |
| POST | `/client/activate` |
| GET | `/client/me` |

### Accounts
| Method | Path | Auth |
|---|---|---|
| GET | `/api/accounts/my` | client |
| GET | `/api/accounts/:accountId` | client |
| GET | `/api/accounts` | EMPLOYEE |
| GET | `/api/admin/accounts/:accountId` | EMPLOYEE |
| GET | `/api/bank-accounts` | EMPLOYEE |
| POST | `/api/accounts/create` | EMPLOYEE |
| PUT | `/api/accounts/:accountId/name` | client |
| PUT | `/api/accounts/:accountId/limits` | EMPLOYEE |
| DELETE | `/api/accounts/:accountId` | EMPLOYEE |

### Payments
| Method | Path |
|---|---|
| POST | `/api/payments/create` |
| GET | `/api/payments` |
| GET | `/api/payments/:paymentId` |
| POST | `/api/transfers` |
| GET | `/api/transfers` |
| GET | `/api/transfers/my` |
| POST | `/api/recipients` |
| GET | `/api/recipients` |
| PUT | `/api/recipients/:id` |
| DELETE | `/api/recipients/:id` |
| PUT | `/api/recipients/reorder` |

### Cards
| Method | Path | Auth |
|---|---|---|
| GET | `/api/cards` | client |
| GET | `/api/cards/by-account/:accountNumber` | EMPLOYEE |
| GET | `/api/cards/:number` | client |
| GET | `/api/cards/id/:id` | client |
| POST | `/api/cards/request` | client |
| POST | `/api/cards/request/confirm` | client |
| PUT | `/api/cards/:id/block` | client |
| PUT | `/api/cards/:id/unblock` | EMPLOYEE |
| PUT | `/api/cards/:id/deactivate` | EMPLOYEE |
| PUT | `/api/cards/:id/limit` | EMPLOYEE |

### Exchange / FX
| Method | Path | Auth |
|---|---|---|
| GET | `/exchange/rates` | public |
| GET | `/exchange/rate` | public |
| POST | `/exchange/convert` | client |
| GET | `/exchange/history` | client |
| POST | `/exchange/preview` | client |

### Loans
| Method | Path | Auth |
|---|---|---|
| GET | `/loans` | client |
| GET | `/loans/:id` | client |
| GET | `/loans/:id/installments` | client |
| POST | `/loans/apply` | client |
| GET | `/admin/loans/applications` | ADMIN |
| PUT | `/admin/loans/:id/approve` | ADMIN |
| PUT | `/admin/loans/:id/reject` | ADMIN |
| GET | `/admin/loans` | ADMIN |
| POST | `/admin/loans/trigger-installments` | ADMIN |

### Securities / Stock exchanges
| Method | Path | Auth |
|---|---|---|
| GET | `/stock-exchanges` | EMPLOYEE |
| POST | `/stock-exchanges` | ADMIN |
| GET | `/stock-exchanges/test-mode` | ADMIN |
| POST | `/stock-exchanges/test-mode` | ADMIN |
| GET | `/stock-exchanges/:id` | EMPLOYEE |
| PUT | `/stock-exchanges/:id` | ADMIN |
| DELETE | `/stock-exchanges/:id` | ADMIN |
| GET | `/stock-exchanges/:id/hours` | EMPLOYEE |
| POST | `/stock-exchanges/hours` | ADMIN |
| GET | `/stock-exchanges/:id/holidays` | EMPLOYEE |
| POST | `/stock-exchanges/holidays` | ADMIN |
| DELETE | `/stock-exchanges/holidays/:polity/:date` | ADMIN |
| GET | `/stock-exchanges/:mic/is-open` | EMPLOYEE |

### Orders
| Method | Path | Auth |
|---|---|---|
| POST | `/orders` | authenticated |
| GET | `/orders` | SUPERVISOR |
| GET | `/orders/:id` | authenticated |
| PUT | `/orders/:id/approve` | SUPERVISOR |
| PUT | `/orders/:id/decline` | SUPERVISOR |
| DELETE | `/orders/:id/portions` | authenticated |
| DELETE | `/orders/:id` | authenticated |

### Two-factor authentication
| Method | Path |
|---|---|
| GET | `/api/approvals/:id/poll` |
| POST | `/api/mobile/approvals` |
| GET | `/api/mobile/approvals` |
| GET | `/api/mobile/approvals/:id` |
| PUT | `/api/twofactor/:id/approve` |
| PUT | `/api/twofactor/:id/reject` |
| POST | `/api/mobile/push-token` |
| DELETE | `/api/mobile/push-token` |

### Docs
| Method | Path |
|---|---|
| GET | `/swagger/*any` |

## Auth
- JWT signed with HMAC-SHA256. Access tokens expire in 15 min, refresh tokens in 7 days.
- `ADMIN` role bypasses all other role checks.
- New employees are created with `active=false` (DB column) / `Aktivan=false` (Go model field) and no password; they cannot log in until activated via `/auth/activate`.
