# HTTP Boundary Validation

The HTTP adapter validates request syntax before inputs reach the application service. This keeps malformed requests as `400 Bad Request` and reserves `404 Not Found` for syntactically valid resources that do not exist.

## Identifier Rules

| Field | Rule | Invalid example | Status |
|-------|------|-----------------|--------|
| `productId` | 1-64 characters; letters, digits, `-`, `_` only | `bad id` | `400 Bad Request` |
| `userId` | 1-64 characters; letters, digits, `-`, `_` only | `bad id` | `400 Bad Request` |
| `reservationId` path value | 1-64 characters; letters, digits, `-`, `_` only | `bad id` | `400 Bad Request` |

Valid but missing IDs still reach the application service:

| Request | Status | Reason |
|---------|--------|--------|
| `GET /products/ghost/stock` | `404 Not Found` | `ghost` is syntactically valid but not seeded |
| `POST /reservations/nope/confirm` | `404 Not Found` | `nope` is syntactically valid but no reservation exists |

## Reservation Payload Rules

`POST /reservations` accepts exactly this shape:

```json
{"productId":"sku-1","userId":"user-a"}
```

The decoder rejects malformed or unsupported shapes before calling `ReserveItem`:

| Payload | Status |
|---------|--------|
| `not-json` | `400 Bad Request` |
| `{}` | `400 Bad Request` |
| `{"productId":"","userId":""}` | `400 Bad Request` |
| `{"productId":"sku-1","userId":"user-a","quantity":1}` | `400 Bad Request` |
| `{"productId":"sku-1","userId":"user-a"}{}` | `400 Bad Request` |

## Verification

```sh
make test
make http-smoke
```