# Contract: Pool Statistics

## `/api/pool/status`

Purpose: provide the free-pool summary used by pool-health logic and the free-pool dashboard cards.

### Response fields

- `Total`: eligible free proxies count
- `HTTP`: eligible free HTTP proxies count
- `SOCKS5`: eligible free SOCKS5 proxies count
- `HTTPSlots`: configured HTTP target slots for the free pool
- `SOCKS5Slots`: configured SOCKS5 target slots for the free pool
- `State`: health state derived from the same free-pool counts
- `AvgLatencyHTTP`: average latency of eligible free HTTP proxies with latency data
- `AvgLatencySocks5`: average latency of eligible free SOCKS5 proxies with latency data
- `CustomCount`: eligible custom proxies count, reported separately and not included in `Total`

### Invariants

- `Total = HTTP + SOCKS5`
- `CustomCount` is displayed separately and MUST NOT require the frontend to subtract it from `Total`
- `State` MUST be based on the same free-pool scope as `Total`, `HTTP`, and `SOCKS5`

## `/api/stats`

Purpose: compatibility endpoint for basic free-pool counts.

### Response fields

- `total`: eligible free proxies count
- `http`: eligible free HTTP proxies count
- `socks5`: eligible free SOCKS5 proxies count
- `custom_count`: eligible custom proxies count
- `port`: HTTP random-rotation proxy port

### Invariants

- `/api/stats` free-pool counts must agree with the same-scope values exposed by `/api/pool/status`

## Dashboard Mapping

- Free-pool total card -> `/api/pool/status.Total`
- Free-pool HTTP card -> `/api/pool/status.HTTP`
- Free-pool SOCKS5 card -> `/api/pool/status.SOCKS5`
- Free-pool latency meta -> `/api/pool/status.AvgLatencyHTTP`, `/api/pool/status.AvgLatencySocks5`
- Subscription available card -> `/api/custom/status.custom_count`
- Subscription disabled card -> `/api/custom/status.disabled_count`
- Proxy table usage column -> per-proxy historical counters only
