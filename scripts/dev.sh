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

# Wait for RabbitMQ to be ready
echo "Waiting for email-rabbitmq to be ready..."
until bash -c 'echo > /dev/tcp/localhost/5672' 2>/dev/null; do
  sleep 1
done
echo "email-rabbitmq ready."

# Load environment variables
set -a; source "$REPO_ROOT/.env"; set +a

# Launch services in background, capture PIDs
go run "$REPO_ROOT/services/employee-service/" &
EMP_PID=$!

go run "$REPO_ROOT/services/auth-service/" &
AUTH_PID=$!

go run "$REPO_ROOT/services/api-gateway/" &
GW_PID=$!

(cd "$REPO_ROOT/services/email-service" && go run .) &
EMAIL_PID=$!

go run "$REPO_ROOT/services/account-service/" &
ACC_PID=$!

go run "$REPO_ROOT/services/client-service/" &
CLIENT_PID=$!

go run "$REPO_ROOT/services/exchange-service/" &
EXCHANGE_PID=$!

go run "$REPO_ROOT/services/payment-service/" &
PAYMENT_PID=$!

echo ""
echo "All services started."
echo "  employee-service  PID $EMP_PID   (:50051)"
echo "  auth-service      PID $AUTH_PID  (:50052)"
echo "  email-service     PID $EMAIL_PID (:50053)"
echo "  account-service   PID $ACC_PID    (:50054)"
echo "  client-service    PID $CLIENT_PID   (:50056)"
echo "  exchange-service  PID $EXCHANGE_PID (:50057)"
echo "  payment-service   PID $PAYMENT_PID  (:50055)"
echo "  api-gateway       PID $GW_PID       (:8081)"
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

# On Ctrl+C, kill Go services only — containers are intentionally left running
trap "echo ''; echo 'Stopping Go services...'; kill $EMP_PID $AUTH_PID $GW_PID $EMAIL_PID $ACC_PID $CLIENT_PID $EXCHANGE_PID $PAYMENT_PID 2>/dev/null; exit 0" INT

wait
