# Quickstart: Statistics Consistency Audit

## 1. Run package tests

```bash
go test ./...
```

## 2. Start the service

Use the existing project startup flow so the WebUI and proxy ports are available.

## 3. Verify free-pool summary contract

```bash
curl -fsS http://127.0.0.1:7778/api/pool/status
curl -fsS http://127.0.0.1:7778/api/stats
```

Confirm:

- `api/pool/status.Total` is already the free-pool count and does not require subtracting `CustomCount`
- `api/stats.total/http/socks5` matches the same-scope free-pool values

## 4. Verify subscription summary contract

```bash
curl -fsS http://127.0.0.1:7778/api/custom/status
curl -fsS http://127.0.0.1:7778/api/subscriptions
```

Confirm:

- subscription summary cards reconcile with per-subscription active/disabled counts
- free-pool cards and subscription cards no longer mix scopes

## 5. Generate live traffic and confirm interpretation

Send traffic through `7779` and compare:

- free-pool summary counts
- subscription summary counts
- proxy table usage counters

Expected result:

- free-pool cards describe free-pool eligibility
- subscription cards describe custom availability
- per-row usage counters do not imply that an entire protocol is unused
