# IRS k6 Load Test Report

Generated: 2026-05-17 14:27 UTC

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


running (0m01.0s), 01/50 VUs, 442 complete and 0 interrupted iterations
default   [   2% ] 01/50 VUs  01.0s/45.0s

running (0m02.0s), 01/50 VUs, 812 complete and 0 interrupted iterations
default   [   4% ] 01/50 VUs  02.0s/45.0s

running (0m03.0s), 02/50 VUs, 1199 complete and 0 interrupted iterations
default   [   7% ] 02/50 VUs  03.0s/45.0s

running (0m04.0s), 02/50 VUs, 1585 complete and 0 interrupted iterations
default   [   9% ] 02/50 VUs  04.0s/45.0s

running (0m05.0s), 02/50 VUs, 1934 complete and 0 interrupted iterations
default   [  11% ] 02/50 VUs  05.0s/45.0s

running (0m06.0s), 03/50 VUs, 2263 complete and 0 interrupted iterations
default   [  13% ] 03/50 VUs  06.0s/45.0s

running (0m07.0s), 03/50 VUs, 2561 complete and 0 interrupted iterations
default   [  16% ] 03/50 VUs  07.0s/45.0s

running (0m08.0s), 04/50 VUs, 2844 complete and 0 interrupted iterations
default   [  18% ] 04/50 VUs  08.0s/45.0s

running (0m09.0s), 04/50 VUs, 3097 complete and 0 interrupted iterations
default   [  20% ] 04/50 VUs  09.0s/45.0s

running (0m10.0s), 04/50 VUs, 3333 complete and 0 interrupted iterations
default   [  22% ] 04/50 VUs  10.0s/45.0s

running (0m11.0s), 06/50 VUs, 3577 complete and 0 interrupted iterations
default   [  24% ] 06/50 VUs  11.0s/45.0s

running (0m12.0s), 07/50 VUs, 3801 complete and 0 interrupted iterations
default   [  27% ] 07/50 VUs  12.0s/45.0s

running (0m13.0s), 09/50 VUs, 4006 complete and 0 interrupted iterations
default   [  29% ] 09/50 VUs  13.0s/45.0s

running (0m14.0s), 10/50 VUs, 4207 complete and 0 interrupted iterations
default   [  31% ] 10/50 VUs  14.0s/45.0s

running (0m15.0s), 12/50 VUs, 4405 complete and 0 interrupted iterations
default   [  33% ] 12/50 VUs  15.0s/45.0s

running (0m16.0s), 13/50 VUs, 4599 complete and 0 interrupted iterations
default   [  36% ] 13/50 VUs  16.0s/45.0s

running (0m17.0s), 15/50 VUs, 4780 complete and 0 interrupted iterations
default   [  38% ] 15/50 VUs  17.0s/45.0s

running (0m18.0s), 16/50 VUs, 4955 complete and 0 interrupted iterations
default   [  40% ] 16/50 VUs  18.0s/45.0s

running (0m19.0s), 18/50 VUs, 5127 complete and 0 interrupted iterations
default   [  42% ] 18/50 VUs  19.0s/45.0s

running (0m20.0s), 19/50 VUs, 5290 complete and 0 interrupted iterations
default   [  44% ] 19/50 VUs  20.0s/45.0s

running (0m21.0s), 21/50 VUs, 5438 complete and 0 interrupted iterations
default   [  47% ] 21/50 VUs  21.0s/45.0s

running (0m22.0s), 22/50 VUs, 5598 complete and 0 interrupted iterations
default   [  49% ] 22/50 VUs  22.0s/45.0s

running (0m23.0s), 24/50 VUs, 5717 complete and 0 interrupted iterations
default   [  51% ] 24/50 VUs  23.0s/45.0s

running (0m24.0s), 25/50 VUs, 5852 complete and 0 interrupted iterations
default   [  53% ] 25/50 VUs  24.0s/45.0s

running (0m25.0s), 27/50 VUs, 6007 complete and 0 interrupted iterations
default   [  56% ] 27/50 VUs  25.0s/45.0s

running (0m26.0s), 28/50 VUs, 6154 complete and 0 interrupted iterations
default   [  58% ] 28/50 VUs  26.0s/45.0s

running (0m27.0s), 30/50 VUs, 6289 complete and 0 interrupted iterations
default   [  60% ] 30/50 VUs  27.0s/45.0s

running (0m28.0s), 31/50 VUs, 6422 complete and 0 interrupted iterations
default   [  62% ] 31/50 VUs  28.0s/45.0s

running (0m29.0s), 33/50 VUs, 6549 complete and 0 interrupted iterations
default   [  64% ] 33/50 VUs  29.0s/45.0s

running (0m30.0s), 34/50 VUs, 6661 complete and 0 interrupted iterations
default   [  67% ] 34/50 VUs  30.0s/45.0s

running (0m31.0s), 36/50 VUs, 6780 complete and 0 interrupted iterations
default   [  69% ] 36/50 VUs  31.0s/45.0s

running (0m32.0s), 37/50 VUs, 6908 complete and 0 interrupted iterations
default   [  71% ] 37/50 VUs  32.0s/45.0s

running (0m33.0s), 39/50 VUs, 7043 complete and 0 interrupted iterations
default   [  73% ] 39/50 VUs  33.0s/45.0s

running (0m34.0s), 40/50 VUs, 7177 complete and 0 interrupted iterations
default   [  76% ] 40/50 VUs  34.0s/45.0s

running (0m35.0s), 42/50 VUs, 7306 complete and 0 interrupted iterations
default   [  78% ] 42/50 VUs  35.0s/45.0s

running (0m36.0s), 43/50 VUs, 7434 complete and 0 interrupted iterations
default   [  80% ] 43/50 VUs  36.0s/45.0s

running (0m37.0s), 45/50 VUs, 7560 complete and 0 interrupted iterations
default   [  82% ] 45/50 VUs  37.0s/45.0s

running (0m38.0s), 46/50 VUs, 7688 complete and 0 interrupted iterations
default   [  84% ] 46/50 VUs  38.0s/45.0s

running (0m39.0s), 48/50 VUs, 7817 complete and 0 interrupted iterations
default   [  87% ] 48/50 VUs  39.0s/45.0s

running (0m40.0s), 49/50 VUs, 7945 complete and 0 interrupted iterations
default   [  89% ] 49/50 VUs  40.0s/45.0s

running (0m41.0s), 43/50 VUs, 8074 complete and 0 interrupted iterations
default   [  91% ] 43/50 VUs  41.0s/45.0s

running (0m42.0s), 31/50 VUs, 8193 complete and 0 interrupted iterations
default   [  93% ] 31/50 VUs  42.0s/45.0s

running (0m43.0s), 21/50 VUs, 8310 complete and 0 interrupted iterations
default   [  96% ] 21/50 VUs  43.0s/45.0s

running (0m44.0s), 11/50 VUs, 8428 complete and 0 interrupted iterations
default   [  98% ] 11/50 VUs  44.0s/45.0s

running (0m45.0s), 01/50 VUs, 8543 complete and 0 interrupted iterations
default   [ 100% ] 01/50 VUs  45.0s/45.0s


  █ THRESHOLDS 

    http_req_duration
    ✓ 'p(95)<500' p(95)=359.57ms

    http_req_failed
    ✓ 'rate<0.01' rate=0.00%

    reservation_created
    ✓ 'count>0' count=8544


  █ TOTAL RESULTS 

    checks_total.......: 17088   379.710348/s
    checks_succeeded...: 100.00% 17088 out of 17088
    checks_failed......: 0.00%   0 out of 17088

    ✓ status is 201 or 409
    ✓ no server error

    CUSTOM
    reservation_created............: 8544   189.855174/s

    HTTP
    http_req_duration..............: avg=113.05ms min=1.39ms med=52.46ms max=394.66ms p(50)=52.46ms p(90)=321.89ms p(95)=359.57ms p(99)=382.88ms
      { expected_response:true }...: avg=113.05ms min=1.39ms med=52.46ms max=394.66ms p(50)=52.46ms p(90)=321.89ms p(95)=359.57ms p(99)=382.88ms
    http_req_failed................: 0.00%  0 out of 8544
    http_reqs......................: 8544   189.855174/s

    EXECUTION
    iteration_duration.............: avg=113.2ms  min=1.43ms med=52.66ms max=394.75ms p(50)=52.66ms p(90)=322.01ms p(95)=359.68ms p(99)=382.96ms
    iterations.....................: 8544   189.855174/s
    vus............................: 1      min=1         max=49
    vus_max........................: 50     min=50        max=50

    NETWORK
    data_received..................: 2.5 MB 56 kB/s
    data_sent......................: 1.5 MB 34 kB/s




running (0m45.0s), 00/50 VUs, 8544 complete and 0 interrupted iterations
default ✓ [ 100% ] 00/50 VUs  45s
```
