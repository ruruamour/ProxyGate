# Data Model: Statistics Consistency Audit

## Entity: Pool Status

Represents the free-pool summary consumed by runtime refill logic and the free-pool dashboard cards.

### Fields

- `Total`: Count of eligible free proxies.
- `HTTP`: Count of eligible free HTTP proxies.
- `SOCKS5`: Count of eligible free SOCKS5 proxies.
- `HTTPSlots`: Configured HTTP slot target for the free pool.
- `SOCKS5Slots`: Configured SOCKS5 slot target for the free pool.
- `State`: Health classification derived from free-pool counts versus slot targets.
- `AvgLatencyHTTP`: Average latency for eligible free HTTP proxies with valid latency data.
- `AvgLatencySocks5`: Average latency for eligible free SOCKS5 proxies with valid latency data.
- `CustomCount`: Count of eligible custom proxies, reported separately for subscription visibility.

### Validation Rules

- `Total = HTTP + SOCKS5` for eligible free proxies.
- `State` must be derived from the same free-pool counts used for `Total`, `HTTP`, and `SOCKS5`.
- Average latency fields must use the same source scope as the counts they accompany.

## Entity: Proxy Usage Record

Represents per-proxy historical usage shown in the detailed table.

### Fields

- `UseCount`
- `SuccessCount`
- `FailCount`
- `LastUsed`

### Validation Rules

- Usage fields are historical per-row counters and must not be interpreted as pool availability counts.

## Entity: Statistics Contract

Defines the mapping between backend response fields and dashboard labels.

### Relationships

- `Pool Status` supplies free-pool summary cards.
- Subscription/custom status supplies subscription summary cards.
- Proxy list rows supply historical usage fields.

### State Transitions

- Contract is valid when each dashboard card maps to one backend scope without additional hidden reinterpretation.
- Contract is invalid when frontend arithmetic or naming causes a card to describe a different scope than the backend field it uses.
