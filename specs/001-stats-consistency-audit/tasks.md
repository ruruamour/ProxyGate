# Tasks: Statistics Consistency Audit

**Input**: Design documents from `/specs/001-stats-consistency-audit/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: This feature requires automated regression coverage for pool statistics calculations and WebUI endpoint contracts.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock the feature contract and identify the concrete implementation files.

- [x] T001 Capture the pool statistics contract in `specs/001-stats-consistency-audit/contracts/pool-stats.md`
- [x] T002 Capture validation steps for the feature in `specs/001-stats-consistency-audit/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared aggregation helpers and a single backend statistics contract before UI changes.

- [x] T003 Add source-scoped count and latency aggregation helpers in `storage/storage.go`
- [x] T004 Refactor free-pool status calculation to use one backend scope in `pool/manager.go`
- [x] T005 Align overlapping stats endpoints to the same scope in `webui/server.go`

**Checkpoint**: Backend pool statistics expose one explicit free-pool contract for UI and runtime consumers.

---

## Phase 3: User Story 1 - Trust Pool Summary (Priority: P1) 🎯 MVP

**Goal**: Make the free-pool summary cards internally consistent in mixed free/custom environments.

**Independent Test**: Seed free and custom proxies, call the pool/stats endpoints, and verify total/protocol/custom counts reconcile without client-side subtraction.

### Tests for User Story 1

- [x] T006 [P] [US1] Add mixed free/custom pool status coverage in `pool/manager_test.go`
- [x] T007 [P] [US1] Add WebUI stats endpoint contract coverage in `webui/server_test.go`

### Implementation for User Story 1

- [x] T008 [US1] Implement corrected pool status aggregation in `storage/storage.go` and `pool/manager.go`
- [x] T009 [US1] Remove hidden dashboard subtraction and bind cards to explicit backend fields in `webui/dashboard.go`
- [x] T010 [US1] Keep `/api/stats` and `/api/pool/status` reconciled in `webui/server.go`

**Checkpoint**: The free-pool cards show consistent values in the dashboard and the API contract is test-covered.

---

## Phase 4: User Story 2 - Reconcile Summary And Tables (Priority: P2)

**Goal**: Audit and fix additional pool-related UI/API mismatches that can mislead operators when reading summary cards and detailed proxy tables together.

**Independent Test**: Compare free-pool cards, subscription cards, and protocol-filtered proxy rows after live traffic and verify they describe compatible scopes.

### Tests for User Story 2

- [x] T011 [P] [US2] Add regression coverage for audited pool-related statistics in `webui/server_test.go`

### Implementation for User Story 2

- [x] T012 [US2] Audit pool-related dashboard mappings and fix any additional scope mismatches in `webui/dashboard.go`, `webui/server.go`, `pool/manager.go`, and `storage/storage.go`
- [x] T013 [US2] Clarify historical usage versus availability semantics in the dashboard rendering logic in `webui/dashboard.go`

**Checkpoint**: Pool-related dashboard summaries and detailed views no longer contradict each other within the audited scope.

---

## Phase 5: User Story 3 - Detect Contract Drift Early (Priority: P3)

**Goal**: Make future frontend/backend drift in pool statistics easier to detect.

**Independent Test**: Break the contract intentionally in local edits and verify the new tests would fail on calculation or API drift.

### Implementation for User Story 3

- [x] T014 [US3] Preserve the audited statistics contract in `specs/001-stats-consistency-audit/contracts/pool-stats.md` and `specs/001-stats-consistency-audit/research.md`
- [x] T015 [US3] Document operator verification steps for the corrected contract in `specs/001-stats-consistency-audit/quickstart.md`

**Checkpoint**: The corrected contract is documented and regression checks cover the high-risk drift points.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validate the implementation end to end.

- [x] T016 Run package tests with `go test ./...`
- [x] T017 Run live endpoint verification for `/api/pool/status` and `/api/stats` against the local service

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Can be completed immediately.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks user-story implementation.
- **User Story 1 (Phase 3)**: Depends on Foundational completion.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because the same statistics contract is being extended and audited.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2.
- **Polish (Phase 6)**: Depends on all implementation work completing.

### Parallel Opportunities

- `T006` and `T007` can be written in parallel because they touch different test files.
- `T011` can run in parallel with documentation-only updates once User Story 1 backend behavior is stable.

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1.
3. Validate API and dashboard reconciliation before expanding the audit.

### Incremental Delivery

1. Fix the backend statistics scope.
2. Update dashboard bindings to match.
3. Audit and correct adjacent pool-related mismatches.
4. Lock the corrected behavior with tests and documentation.
