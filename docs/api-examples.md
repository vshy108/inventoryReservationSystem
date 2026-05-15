# HTTP API Examples

These examples are copy/paste checks for the HTTP boundary. Start the server in one terminal:

```sh
go run ./cmd/server --addr=127.0.0.1:8080 --seed="sku-1:1,sku-2:1" --hold=2m
```

Use this base URL in another terminal:

```sh
BASE_URL=http://127.0.0.1:8080
```

## Liveness

```sh
curl -i "$BASE_URL/healthz"
```

Expected status: `200 OK`

Expected body:

```json
{"status":"ok"}
```

## Reserve One Unit

```sh
curl -i -X POST "$BASE_URL/reservations" \
  -H 'Content-Type: application/json' \
  -d '{"productId":"sku-1","userId":"user-a"}'
```

Expected status: `201 Created`

The response includes a generated `reservationId`:

```json
{
  "reservationId": "res-...",
  "productId": "sku-1",
  "userId": "user-a",
  "state": "Active",
  "createdAt": "...",
  "expiresAt": "..."
}
```

## Stock Lookup

```sh
curl -i "$BASE_URL/products/sku-1/stock"
```

Expected status: `200 OK`

Expected body after reserving the only `sku-1` unit:

```json
{"productId":"sku-1","available":0}
```

## Out Of Stock

```sh
curl -i -X POST "$BASE_URL/reservations" \
  -H 'Content-Type: application/json' \
  -d '{"productId":"sku-1","userId":"user-b"}'
```

Expected status: `409 Conflict`

Expected body:

```json
{"error":"out of stock"}
```

## Confirm Reservation

Replace `res-...` with the `reservationId` returned by the reserve call.

```sh
curl -i -X POST "$BASE_URL/reservations/res-.../confirm"
```

Expected status: `200 OK`

Expected response state: `Confirmed`

## Cancel Reservation

Create a second reservation against `sku-2`, then cancel it:

```sh
curl -i -X POST "$BASE_URL/reservations" \
  -H 'Content-Type: application/json' \
  -d '{"productId":"sku-2","userId":"user-c"}'

curl -i -X POST "$BASE_URL/reservations/res-.../cancel"
```

Expected status for cancel: `200 OK`

Expected response state: `Cancelled`

## Error Status Reference

| Scenario | Example | Expected status |
|----------|---------|-----------------|
| Unknown product stock lookup | `GET /products/ghost/stock` | `404 Not Found` |
| Unknown product reservation | `POST /reservations` with `productId=ghost` | `404 Not Found` |
| Out-of-stock reservation | Second reserve for `sku-1` when stock is 1 | `409 Conflict` |
| Finalized reservation transition | Confirm an already confirmed reservation | `409 Conflict` |
| Malformed JSON | `POST /reservations` with invalid JSON | `400 Bad Request` |
| Missing required fields | `POST /reservations` with blank `productId` or `userId` | `400 Bad Request` |

## One-Command Smoke Check

Run the local smoke script to start the server, execute the examples above, and assert status codes:

```sh
bash scripts/http_smoke.sh
```