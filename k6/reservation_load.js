// k6/reservation_load.js
// Measures throughput and latency of POST /reservations against
// the Postgres-backed inventory reservation service.
//
// Stages: 10s warmup (5 VUs) → 30s sustained load (50 VUs) → 5s ramp-down.
// Stock is seeded to 1 000 000 units so the happy path (201) dominates
// and measures pure Postgres round-trip performance under concurrency.
//
// Run:
//   k6 run k6/reservation_load.js
// Override target:
//   k6 run k6/reservation_load.js --env BASE_URL=http://localhost:8090

import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

// Tell k6 that 409 (out of stock) is not a failure — it is a valid domain outcome.
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 299 }, 409));

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8090';
const PRODUCT_ID = __ENV.PRODUCT_ID || 'sku-load';

export const reservationCreated = new Counter('reservation_created');
export const reservationOutOfStock = new Counter('reservation_out_of_stock');

export const options = {
  stages: [
    { duration: '10s', target: 5 },  // warmup
    { duration: '30s', target: 50 }, // sustained load
    { duration: '5s', target: 0 },   // ramp-down
  ],
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(50)', 'p(90)', 'p(95)', 'p(99)'],
  thresholds: {
    http_req_failed: ['rate<0.01'],      // <1% 5xx or network errors
    http_req_duration: ['p(95)<500'],    // p(95) under 500 ms with Postgres
    reservation_created: ['count>0'],    // at least one reservation must succeed
  },
};

export default function () {
  // Unique userId per VU per iteration prevents artificial uniqueness conflicts
  // and ensures each request exercises the full reservation write path.
  const userId = `user-${__VU}-${__ITER}`;

  const res = http.post(
    `${BASE_URL}/reservations`,
    JSON.stringify({ productId: PRODUCT_ID, userId }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'create-reservation' },
    },
  );

  if (res.status === 201) reservationCreated.add(1);
  if (res.status === 409) reservationOutOfStock.add(1);

  check(res, {
    'status is 201 or 409': (r) => r.status === 201 || r.status === 409,
    'no server error': (r) => r.status < 500,
  });
}
