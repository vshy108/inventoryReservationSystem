#!/usr/bin/env bash
set -euo pipefail

container_name="${INVENTORY_PG_MIGRATION_CONTAINER:-inventory-pg-migration-$$}"
pg_port="${INVENTORY_PG_MIGRATION_PORT:-$((55432 + RANDOM % 1000))}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
migration_dir="$repo_root/internal/infrastructure/postgres/migrations"

cleanup() {
  docker rm -f "$container_name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker run --rm -d \
  --name "$container_name" \
  -e POSTGRES_USER=inventory \
  -e POSTGRES_PASSWORD=inventory \
  -e POSTGRES_DB=inventory \
  -p "127.0.0.1:${pg_port}:5432" \
  postgres:18-alpine >/dev/null

until docker exec "$container_name" pg_isready -U inventory -d inventory >/dev/null 2>&1; do
  sleep 1
done

docker exec -i "$container_name" psql -v ON_ERROR_STOP=1 -U inventory -d inventory < "$migration_dir/0001_shadow_persistence.sql" >/dev/null
docker exec -i "$container_name" psql -v ON_ERROR_STOP=1 -U inventory -d inventory < "$migration_dir/0001_shadow_persistence.down.sql" >/dev/null

echo "Postgres migration smoke passed"
echo "container=$container_name"
echo "database_url=postgres://inventory:inventory@127.0.0.1:${pg_port}/inventory"