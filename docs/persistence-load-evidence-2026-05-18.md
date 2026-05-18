# Persistence and Load Evidence - 2026-05-18

This note records a fresh validation pass for the Go inventory reservation service with HTTP behavior, Postgres migrations, and k6 load evidence.

## Goal

Prove the Go backend capacity and persistence claims are current and reproducible.

## Commands Run

```sh
make test
make http-smoke
scripts/postgres_migration_smoke.sh
make load-test
```

## Results

| Check | Result | Evidence |
|-------|--------|----------|
| Go tests | Passed | `go test ./...` passed across app, server, domain, application, infrastructure, migrations, and HTTP packages |
| HTTP smoke | Passed | Temporary server completed reservation, conflict, confirm, cancel, unknown product, and bad JSON checks |
| Postgres migration smoke | Passed | Migration up/down cycle completed against `postgres:18-alpine` |
| k6 load test | Passed | 8,222 HTTP requests, 0.00% failures, p95 386.51 ms, all thresholds passed |

## k6 Snapshot

```text
http_req_duration p(95)=386.51ms
http_req_failed=0.00% 0 out of 8222
reservation_created=8222
checks_succeeded=100.00% 16444 out of 16444
http_reqs=8222 at 182.679613/s
```

## Smoke Fix

The migration smoke initially raced Postgres database creation. The script now waits on a real `SELECT 1` against the target database before applying migrations, which makes the smoke deterministic during fast local runs.

## Reviewer Signal

The Go backend still proves reproducible HTTP behavior, Postgres migration safety, Redis/Postgres-backed load execution, and threshold-backed capacity evidence.

Next useful slice: compare this refreshed Go load result against the Rust implementation's equivalent critical behavior or load profile.