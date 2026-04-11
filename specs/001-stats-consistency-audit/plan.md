# Implementation Plan: Statistics Consistency Audit

**Branch**: `001-stats-consistency-audit` | **Date**: 2026-04-11 | **Spec**: `specs/001-stats-consistency-audit/spec.md`
**Input**: Feature specification from `/specs/001-stats-consistency-audit/spec.md`

## Summary

Restore a single, explicit statistics contract for the WebUI by making pool status calculations free-pool scoped where the product labels describe the free pool, exposing any needed derived counts without client-side hidden arithmetic, and adding tests plus a focused audit for other pool-related frontend/backend mismatches.

## Technical Context

**Language/Version**: Go 1.25.0  
**Primary Dependencies**: Go standard library, `github.com/mattn/go-sqlite3`, `gopkg.in/yaml.v3`, embedded HTML/JavaScript in Go source  
**Storage**: SQLite database in `data/` via `storage.Storage`  
**Testing**: `go test ./...` with package tests using `testing` and `net/http/httptest`  
**Target Platform**: Linux server and Docker deployment running a single Go service plus sing-box for custom nodes  
**Project Type**: Embedded-dashboard web service with HTTP/SOCKS5 proxy management  
**Performance Goals**: Pool/status endpoints remain safe for 5s dashboard polling without noticeable operator lag or extra background refill churn  
**Constraints**: Preserve free-only, custom-only, and mixed routing behavior; avoid schema migrations; keep API changes backward-compatible where practical; make counting scope explicit instead of implicit  
**Scale/Scope**: One embedded WebUI, roughly 100 free pool slots plus dozens of custom nodes, package-level tests rather than a separate frontend test runner

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

This feature satisfies the repository constitution:

- Runtime truth remains backend-owned by removing client-side subtraction and aligning pool statistics scopes.
- Proxy mode semantics remain backward-compatible.
- Automated regression coverage was added for the corrected operator-facing statistics contract.

Result: PASS for planning and implementation.

## Project Structure

### Documentation (this feature)

```text
specs/001-stats-consistency-audit/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── pool-stats.md
└── tasks.md
```

### Source Code (repository root)

```text
main.go
config/
custom/
pool/
proxy/
storage/
validator/
webui/
test/
```

**Structure Decision**: This is a single Go service with embedded WebUI assets in `webui/dashboard.go`, HTTP handlers in `webui/server.go`, pool calculations in `pool/manager.go`, and shared statistics queries in `storage/storage.go`. Tests live alongside packages, primarily in `webui/server_test.go` and `pool/manager_test.go`.

## Phase 0: Research

- Confirm whether `PoolStatus` is consumed by runtime refill/health logic in addition to the WebUI.
- Identify every dashboard statistic that performs client-side reinterpretation of backend values.
- Decide whether to fix drift by changing field meaning, adding explicit fields, or both.

## Phase 1: Design

- Define the statistics contract for free-pool counts, custom counts, protocol counts, and latency scope.
- Map each dashboard card and table statistic to one backend source of truth.
- Define the audit scope for adjacent pool-related inconsistencies and the minimum tests required.

## Phase 2: Implementation Strategy

1. Normalize pool-status counting and latency scope in the backend.
2. Remove hidden client-side arithmetic or ambiguous field mapping in the dashboard.
3. Add tests for mixed free/custom data and audit/fix any additional in-scope mismatches discovered.
4. Validate with package tests and live endpoint checks against the running service where possible.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
