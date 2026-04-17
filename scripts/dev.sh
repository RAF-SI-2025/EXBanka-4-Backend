#!/usr/bin/env bash
set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Start employee DB
echo "Starting employee-db..."
(cd "$REPO_ROOT/services/employee-service" && docker compose up -d)

# Start auth DB
echo "Starting auth-db..."
(cd "$REPO_ROOT/services/auth-service" && docker compose up -d)

# Start email RabbitMQ
echo "Starting email-rabbitmq..."
(cd "$REPO_ROOT/services/email-service" && docker compose up -d)

# Start account DB
echo "Starting account-db..."
(cd "$REPO_ROOT/services/account-service" && docker compose up -d)

# Start client DB
echo "Starting client-db..."
(cd "$REPO_ROOT/services/client-service" && docker compose up -d)

# Start exchange DB
echo "Starting exchange-db..."
(cd "$REPO_ROOT/services/exchange-service" && docker compose up -d)

# Start payment DB
echo "Starting payment-db..."
(cd "$REPO_ROOT/services/payment-service" && docker compose up -d)

# Start card DB
echo "Starting card-db..."
(cd "$REPO_ROOT/services/card-service" && docker compose up -d)

# Start loan DB
echo "Starting loan-db..."
(cd "$REPO_ROOT/services/loan-service" && docker compose up -d)

# Start securities DB
echo "Starting securities-db..."
(cd "$REPO_ROOT/services/securities-service" && docker compose up -d)

# Start portfolio DB
echo "Starting portfolio-db..."
(cd "$REPO_ROOT/services/portfolio-service" && docker compose up -d)

# Wait for PostgreSQL to accept connections
echo "Waiting for employee-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/employee-service/docker-compose.yml" ps -q employee-db) \
    pg_isready -U employee_user -d employee_db -q 2>/dev/null; do
  sleep 1
done
echo "employee-db ready."

echo "Waiting for auth-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/auth-service/docker-compose.yml" ps -q auth-db) \
    pg_isready -U auth_user -d auth_db -q 2>/dev/null; do
  sleep 1
done
echo "auth-db ready."

echo "Waiting for account-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/account-service/docker-compose.yml" ps -q account-db) \
    pg_isready -U account_user -d account_db -q 2>/dev/null; do
  sleep 1
done
echo "account-db ready."

echo "Waiting for client-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/client-service/docker-compose.yml" ps -q client-db) \
    pg_isready -U client_user -d client_db -q 2>/dev/null; do
  sleep 1
done
echo "client-db ready."

echo "Waiting for exchange-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/exchange-service/docker-compose.yml" ps -q exchange-db) \
    pg_isready -U exchange_user -d exchange_db -q 2>/dev/null; do
  sleep 1
done
echo "exchange-db ready."

echo "Waiting for payment-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/payment-service/docker-compose.yml" ps -q payment-db) \
    pg_isready -U payment_user -d payment_db -q 2>/dev/null; do
  sleep 1
done
echo "payment-db ready."

echo "Waiting for card-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/card-service/docker-compose.yml" ps -q card-db) \
    pg_isready -U card_user -d card_db -q 2>/dev/null; do
  sleep 1
done
echo "card-db ready."

echo "Waiting for loan-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/loan-service/docker-compose.yml" ps -q loan-db) \
    pg_isready -U loan_user -d loan_db -q 2>/dev/null; do
  sleep 1
done
echo "loan-db ready."

echo "Waiting for securities-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/securities-service/docker-compose.yml" ps -q securities-db) \
    pg_isready -U securities_user -d securities_db -q 2>/dev/null; do
  sleep 1
done
echo "securities-db ready."

echo "Waiting for portfolio-db to be ready..."
until docker exec $(docker compose -f "$REPO_ROOT/services/portfolio-service/docker-compose.yml" ps -q portfolio-db) \
    pg_isready -U portfolio_user -d portfolio_db -q 2>/dev/null; do
  sleep 1
done
echo "portfolio-db ready."

# Wait for RabbitMQ to be ready
echo "Waiting for email-rabbitmq to be ready..."
until bash -c 'echo > /dev/tcp/localhost/5672' 2>/dev/null; do
  sleep 1
done
echo "email-rabbitmq ready."

# Load environment variables
set -a; source "$REPO_ROOT/.env"; set +a

# Launch services in background, capture PIDs
# Note: auth-service, client-service, employee-service each use DB_URL — pass per-process
DB_URL="$EMPLOYEE_DB_URL" go run "$REPO_ROOT/services/employee-service/" &
EMP_PID=$!

DB_URL="$AUTH_DB_URL" go run "$REPO_ROOT/services/auth-service/" &
AUTH_PID=$!

go run "$REPO_ROOT/services/api-gateway/" &
GW_PID=$!

(cd "$REPO_ROOT/services/email-service" && go run .) &
EMAIL_PID=$!

go run "$REPO_ROOT/services/account-service/" &
ACC_PID=$!

DB_URL="$CLIENT_DB_URL" go run "$REPO_ROOT/services/client-service/" &
CLIENT_PID=$!

go run "$REPO_ROOT/services/exchange-service/" &
EXCHANGE_PID=$!

go run "$REPO_ROOT/services/payment-service/" &
PAYMENT_PID=$!

go run "$REPO_ROOT/services/card-service/" &
CARD_PID=$!

go run "$REPO_ROOT/services/loan-service/" &
LOAN_PID=$!

go run "$REPO_ROOT/services/securities-service/" &
SEC_PID=$!

go run "$REPO_ROOT/services/portfolio-service/" &
PORTFOLIO_PID=$!

echo ""
echo "All services started."
echo "  employee-service  PID $EMP_PID   (:50051)"
echo "  auth-service      PID $AUTH_PID  (:50052)"
echo "  email-service     PID $EMAIL_PID (:50053)"
echo "  account-service   PID $ACC_PID    (:50054)"
echo "  client-service    PID $CLIENT_PID   (:50056)"
echo "  exchange-service  PID $EXCHANGE_PID (:50057)"
echo "  payment-service   PID $PAYMENT_PID  (:50055)"
echo "  card-service      PID $CARD_PID     (:50059)"
echo "  loan-service        PID $LOAN_PID        (:50058)"
echo "  securities-service  PID $SEC_PID         (:50060)"
echo "  portfolio-service   PID $PORTFOLIO_PID   (:50062)"
echo "  api-gateway         PID $GW_PID          (:8083)"
echo ""
echo "Press Ctrl+C to stop all services."
echo "Note: the database and RabbitMQ containers keep running after Ctrl+C."
echo "      To stop them manually:"
echo "        cd services/employee-service && docker compose down"
echo "        cd services/auth-service && docker compose down"
echo "        cd services/email-service && docker compose down"
echo "        cd services/account-service && docker compose down"
echo "        cd services/client-service && docker compose down"
echo "        cd services/exchange-service && docker compose down"
echo "        cd services/payment-service && docker compose down"
echo "        cd services/card-service && docker compose down"
echo "        cd services/loan-service && docker compose down
        cd services/securities-service && docker compose down
        cd services/portfolio-service && docker compose down"

# On Ctrl+C, kill Go services only — containers are intentionally left running
trap "echo ''; echo 'Stopping Go services...'; kill $EMP_PID $AUTH_PID $GW_PID $EMAIL_PID $ACC_PID $CLIENT_PID $EXCHANGE_PID $PAYMENT_PID $CARD_PID $LOAN_PID $SEC_PID $PORTFOLIO_PID 2>/dev/null; exit 0" INT

wait
