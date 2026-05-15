# Contention Demo

The service uses a per-product lock around the reservation read-check-write path. The reviewer-facing contention demo runs many concurrent attempts against one hot SKU and verifies the single-winner shape.

Run the default scenario:

```sh
go run ./cmd/contention-demo
```

Expected output:

```text
product=sku-hot stock=1 requests=500 successes=1 outOfStock=499 otherErrors=0 available=0
```

The command exits nonzero if any invariant fails:

- successes must equal the seeded stock
- out-of-stock failures must equal `requests - stock`
- other errors must be zero
- final available stock must be zero

You can vary the shape while keeping `requests > stock`:

```sh
go run ./cmd/contention-demo --requests=1000 --stock=5
```

## Verification

```sh
go test -race ./...
go run ./cmd/contention-demo
```