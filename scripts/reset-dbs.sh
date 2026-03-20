#!/usr/bin/env bash

# Reset all DBs (everything except employee-service)
# ./scripts/reset-dbs.sh

# Reset only specific ones
# ./scripts/reset-dbs.sh account-service exchange-service

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

SERVICES=(auth-service account-service client-service exchange-service payment-service)

# Allow overriding which services to reset via arguments, e.g.:
#   ./scripts/reset-dbs.sh account-service exchange-service
if [ $# -gt 0 ]; then
  SERVICES=("$@")
fi

echo "Resetting databases for: ${SERVICES[*]}"
echo ""

for svc in "${SERVICES[@]}"; do
  echo "--- $svc ---"
  (cd "$REPO_ROOT/services/$svc" && docker compose down -v && docker compose up -d)
  echo ""
done

echo "All databases reset. Run ./scripts/dev.sh to start the full stack."
