# Research: Statistics Consistency Audit

## Decision 1: `PoolStatus` should be free-pool scoped for core counts and state

**Decision**: Treat `PoolStatus.Total`, `HTTP`, `SOCKS5`, `AvgLatencyHTTP`, `AvgLatencySocks5`, and `State` as free-pool metrics, not mixed free+custom metrics.

**Rationale**:

- The dashboard section consuming these fields is explicitly labeled as the free pool.
- The same `PoolStatus` object is used by refill and monitoring logic, which manage only the free pool.
- Allowing `custom` nodes to inflate these fields hides real free-pool shortages and causes UI drift.

**Alternatives considered**:

- Keep `PoolStatus` mixed in mixed mode and let the frontend subtract custom counts.
  Rejected because it already caused inconsistent totals and does not work for protocol counts or latency scope.
- Split runtime and UI status into entirely separate APIs.
  Rejected because it adds more contract surface than needed for this fix.

## Decision 2: Frontend cards should consume explicit backend values, not hidden arithmetic

**Decision**: The dashboard should render free-pool cards directly from backend values whose meaning matches the labels, without subtracting custom counts in JavaScript.

**Rationale**:

- Client-side arithmetic obscured the true contract and only partially corrected the mismatch.
- Direct mapping makes the dashboard readable and easier to test.

**Alternatives considered**:

- Keep subtraction in the client and document it.
  Rejected because it still leaves protocol and latency scopes inconsistent.

## Decision 3: Audit scope covers adjacent pool-related statistics that can contradict the same view

**Decision**: Audit and correct any additional mismatches within the pool-status view, including protocol counts, latency averages, and any duplicate API that exposes the same concept with a conflicting scope.

**Rationale**:

- Fixing only `stat-total` would leave the root cause intact.
- `/api/stats` and `/api/pool/status` currently overlap in meaning and can drift.

**Alternatives considered**:

- Restrict the change to one dashboard number.
  Rejected because the user explicitly asked for broader frontend/backend consistency review.

## Decision 4: Add automated coverage at both calculation and API levels

**Decision**: Add package tests for pool-status calculation and WebUI endpoint responses using mixed free/custom fixtures.

**Rationale**:

- The bug originated in a contract boundary between storage, pool manager, and dashboard.
- Coverage at one layer alone would miss future drift in another layer.

**Alternatives considered**:

- Rely only on manual UI checks.
  Rejected because this class of regression is easy to reintroduce.
