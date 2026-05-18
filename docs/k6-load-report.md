# IRS k6 Load Test Report

Summary evidence: [`persistence-load-evidence-2026-05-18.md`](./persistence-load-evidence-2026-05-18.md).

Generated: 2026-05-18 12:05 UTC

## Setup

- Service: Go inventory reservation service + PostgreSQL 17 + Redis 8
- Seed: `sku-load:1000000` (1 M units — stock does not exhaust during test)
- Scenario: `POST /reservations` with unique `userId` per VU/iteration
- Stages: 10 s warmup (5 VUs) → 30 s load (50 VUs) → 5 s ramp-down
- Thresholds: p(95) < 500 ms, http error rate < 1 %

## k6 Output

```

         /\      Grafana   /‾‾/  
    /\  /  \     |\  __   /  /   
   /  \/    \    | |/ /  /   ‾‾\ 
  /          \   |   (  |  (‾)  |
 / __________ \  |_|\_\  \_____/ 


     execution: local
        script: /Users/00156637hopenghou/Repo/inventoryReservationSystem/k6/reservation_load.js
        output: -

     scenarios: (100.00%) 1 scenario, 50 max VUs, 1m15s max duration (incl. graceful stop):
              * default: Up to 50 looping VUs for 45s over 3 stages (gracefulRampDown: 30s, gracefulStop: 30s)


running (0m01.0s), 01/50 VUs, 353 complete and 0 interrupted iterations
default   [   2% ] 01/50 VUs  01.0s/45.0s

running (0m02.0s), 01/50 VUs, 667 complete and 0 interrupted iterations
default   [   4% ] 01/50 VUs  02.0s/45.0s

running (0m03.0s), 02/50 VUs, 1004 complete and 0 interrupted iterations
default   [   7% ] 02/50 VUs  03.0s/45.0s

running (0m04.0s), 02/50 VUs, 1375 complete and 0 interrupted iterations
default   [   9% ] 02/50 VUs  04.0s/45.0s

running (0m05.0s), 02/50 VUs, 1705 complete and 0 interrupted iterations
default   [  11% ] 02/50 VUs  05.0s/45.0s

running (0m06.0s), 03/50 VUs, 2021 complete and 0 interrupted iterations
default   [  13% ] 03/50 VUs  06.0s/45.0s

running (0m07.0s), 03/50 VUs, 2274 complete and 0 interrupted iterations
default   [  16% ] 03/50 VUs  07.0s/45.0s

running (0m08.0s), 04/50 VUs, 2551 complete and 0 interrupted iterations
default   [  18% ] 04/50 VUs  08.0s/45.0s

running (0m09.0s), 04/50 VUs, 2805 complete and 0 interrupted iterations
default   [  20% ] 04/50 VUs  09.0s/45.0s

running (0m10.0s), 04/50 VUs, 3053 complete and 0 interrupted iterations
default   [  22% ] 04/50 VUs  10.0s/45.0s

running (0m11.0s), 06/50 VUs, 3298 complete and 0 interrupted iterations
default   [  24% ] 06/50 VUs  11.0s/45.0s

running (0m12.0s), 07/50 VUs, 3532 complete and 0 interrupted iterations
default   [  27% ] 07/50 VUs  12.0s/45.0s

running (0m13.0s), 09/50 VUs, 3743 complete and 0 interrupted iterations
default   [  29% ] 09/50 VUs  13.0s/45.0s

running (0m14.0s), 10/50 VUs, 3942 complete and 0 interrupted iterations
default   [  31% ] 10/50 VUs  14.0s/45.0s

running (0m15.0s), 12/50 VUs, 4130 complete and 0 interrupted iterations
default   [  33% ] 12/50 VUs  15.0s/45.0s

running (0m16.0s), 13/50 VUs, 4311 complete and 0 interrupted iterations
default   [  36% ] 13/50 VUs  16.0s/45.0s

running (0m17.0s), 15/50 VUs, 4488 complete and 0 interrupted iterations
default   [  38% ] 15/50 VUs  17.0s/45.0s

running (0m18.0s), 16/50 VUs, 4660 complete and 0 interrupted iterations
default   [  40% ] 16/50 VUs  18.0s/45.0s

running (0m19.0s), 18/50 VUs, 4814 complete and 0 interrupted iterations
default   [  42% ] 18/50 VUs  19.0s/45.0s

running (0m20.0s), 19/50 VUs, 4978 complete and 0 interrupted iterations
default   [  44% ] 19/50 VUs  20.0s/45.0s

running (0m21.0s), 21/50 VUs, 5144 complete and 0 interrupted iterations
default   [  47% ] 21/50 VUs  21.0s/45.0s

running (0m22.0s), 22/50 VUs, 5307 complete and 0 interrupted iterations
default   [  49% ] 22/50 VUs  22.0s/45.0s

running (0m23.0s), 24/50 VUs, 5465 complete and 0 interrupted iterations
default   [  51% ] 24/50 VUs  23.0s/45.0s

running (0m24.0s), 25/50 VUs, 5622 complete and 0 interrupted iterations
default   [  53% ] 25/50 VUs  24.0s/45.0s

running (0m25.0s), 27/50 VUs, 5774 complete and 0 interrupted iterations
default   [  56% ] 27/50 VUs  25.0s/45.0s

running (0m26.0s), 28/50 VUs, 5920 complete and 0 interrupted iterations
default   [  58% ] 28/50 VUs  26.0s/45.0s

running (0m27.0s), 30/50 VUs, 6062 complete and 0 interrupted iterations
default   [  60% ] 30/50 VUs  27.0s/45.0s

running (0m28.0s), 31/50 VUs, 6203 complete and 0 interrupted iterations
default   [  62% ] 31/50 VUs  28.0s/45.0s

running (0m29.0s), 33/50 VUs, 6339 complete and 0 interrupted iterations
default   [  64% ] 33/50 VUs  29.0s/45.0s

running (0m30.0s), 34/50 VUs, 6471 complete and 0 interrupted iterations
default   [  67% ] 34/50 VUs  30.0s/45.0s

running (0m31.0s), 36/50 VUs, 6595 complete and 0 interrupted iterations
default   [  69% ] 36/50 VUs  31.0s/45.0s

running (0m32.0s), 37/50 VUs, 6720 complete and 0 interrupted iterations
default   [  71% ] 37/50 VUs  32.0s/45.0s

running (0m33.0s), 39/50 VUs, 6846 complete and 0 interrupted iterations
default   [  73% ] 39/50 VUs  33.0s/45.0s

running (0m34.0s), 40/50 VUs, 6966 complete and 0 interrupted iterations
default   [  76% ] 40/50 VUs  34.0s/45.0s

running (0m35.0s), 42/50 VUs, 7080 complete and 0 interrupted iterations
default   [  78% ] 42/50 VUs  35.0s/45.0s

running (0m36.0s), 43/50 VUs, 7198 complete and 0 interrupted iterations
default   [  80% ] 43/50 VUs  36.0s/45.0s

running (0m37.0s), 45/50 VUs, 7312 complete and 0 interrupted iterations
default   [  82% ] 45/50 VUs  37.0s/45.0s

running (0m38.0s), 46/50 VUs, 7432 complete and 0 interrupted iterations
default   [  84% ] 46/50 VUs  38.0s/45.0s

running (0m39.0s), 48/50 VUs, 7548 complete and 0 interrupted iterations
default   [  87% ] 48/50 VUs  39.0s/45.0s

running (0m40.0s), 49/50 VUs, 7666 complete and 0 interrupted iterations
default   [  89% ] 49/50 VUs  40.0s/45.0s

running (0m41.0s), 41/50 VUs, 7783 complete and 0 interrupted iterations
default   [  91% ] 41/50 VUs  41.0s/45.0s

running (0m42.0s), 34/50 VUs, 7897 complete and 0 interrupted iterations
default   [  93% ] 34/50 VUs  42.0s/45.0s

running (0m43.0s), 21/50 VUs, 8011 complete and 0 interrupted iterations
default   [  96% ] 21/50 VUs  43.0s/45.0s

running (0m44.0s), 11/50 VUs, 8121 complete and 0 interrupted iterations
default   [  98% ] 11/50 VUs  44.0s/45.0s

running (0m45.0s), 01/50 VUs, 8220 complete and 0 interrupted iterations
default   [ 100% ] 01/50 VUs  45.0s/45.0s


  █ THRESHOLDS 

    http_req_duration
    ✓ 'p(95)<500' p(95)=386.51ms

    http_req_failed
    ✓ 'rate<0.01' rate=0.00%

    reservation_created
    ✓ 'count>0' count=8222


  █ TOTAL RESULTS 

    checks_total.......: 16444   365.359226/s
    checks_succeeded...: 100.00% 16444 out of 16444
    checks_failed......: 0.00%   0 out of 16444

    ✓ status is 201 or 409
    ✓ no server error

    CUSTOM
    reservation_created............: 8222   182.679613/s

    HTTP
    http_req_duration..............: avg=117.52ms min=1.74ms med=59.98ms max=432.25ms p(50)=59.98ms p(90)=349.56ms p(95)=386.51ms p(99)=418.87ms
      { expected_response:true }...: avg=117.52ms min=1.74ms med=59.98ms max=432.25ms p(50)=59.98ms p(90)=349.56ms p(95)=386.51ms p(99)=418.87ms
    http_req_failed................: 0.00%  0 out of 8222
    http_reqs......................: 8222   182.679613/s

    EXECUTION
    iteration_duration.............: avg=117.71ms min=1.78ms med=60.1ms  max=432.35ms p(50)=60.1ms  p(90)=349.72ms p(95)=386.71ms p(99)=419.16ms
    iterations.....................: 8222   182.679613/s
    vus............................: 1      min=1         max=49
    vus_max........................: 50     min=50        max=50

    NETWORK
    data_received..................: 2.4 MB 54 kB/s
    data_sent......................: 1.5 MB 33 kB/s




running (0m45.0s), 00/50 VUs, 8222 complete and 0 interrupted iterations
default ✓ [ 100% ] 00/50 VUs  45s
```
