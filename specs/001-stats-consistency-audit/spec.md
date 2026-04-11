# Feature Specification: Statistics Consistency Audit

**Feature Branch**: `001-stats-consistency-audit`  
**Created**: 2026-04-11  
**Status**: Draft  
**Input**: User description: "Fix frontend/backend statistics mismatch, audit other frontend/backend inconsistencies, and implement the necessary corrections"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Trust Pool Summary (Priority: P1)

As an administrator viewing the dashboard, I need the pool summary cards to use one clearly defined counting scope so I can trust what the totals and protocol counts mean without manually reconciling them.

**Why this priority**: The summary cards are the first operational signal shown in the WebUI. If they disagree internally, operators cannot trust the dashboard.

**Independent Test**: Open the dashboard with a mixed free/custom pool and verify that total, protocol, and custom counts reconcile according to the same documented scope.

**Acceptance Scenarios**:

1. **Given** the system has both free and custom proxies, **When** an administrator opens the dashboard, **Then** the total count and protocol counts shown in the summary are derived from the same counting scope.
2. **Given** the system reports custom proxy counts separately, **When** an administrator compares total and protocol cards, **Then** the relationship between them is internally consistent and does not require hidden subtraction rules.

---

### User Story 2 - Reconcile Summary And Tables (Priority: P2)

As an administrator investigating protocol usage, I need the summary area and the proxy tables to present compatible meanings so I can understand whether HTTP and SOCKS5 proxies are present, eligible, and actually being used.

**Why this priority**: Operators use the summary cards and detailed tables together. If the two views describe different scopes without making that clear, they can misdiagnose routing behavior.

**Independent Test**: Generate traffic through the mixed rotation port, then compare protocol summary values and proxy table contents to verify they describe consistent states and do not imply false routing failures.

**Acceptance Scenarios**:

1. **Given** a protocol has eligible proxies in the pool, **When** the administrator views both the summary and filtered proxy tables, **Then** the counts and labels do not contradict each other.
2. **Given** only a subset of proxies has recorded traffic, **When** the administrator inspects usage columns, **Then** the interface does not imply that an entire protocol is unused merely because some rows show zero usage.

---

### User Story 3 - Detect Contract Drift Early (Priority: P3)

As a maintainer changing the dashboard or API, I need a lightweight consistency audit for key statistics so that future UI/API changes do not silently drift apart again.

**Why this priority**: The current issue came from separate counting rules evolving independently. A basic audit reduces regressions during future changes.

**Independent Test**: Run the relevant automated checks and verify they fail if a future change causes summary data or labels to disagree with the defined contract.

**Acceptance Scenarios**:

1. **Given** the counting contract for pool statistics is defined, **When** a maintainer changes either the API or dashboard mapping, **Then** automated checks detect contract drift before release.
2. **Given** a new dashboard statistic is introduced, **When** it is added to the consistency audit coverage, **Then** its intended counting scope is explicit and testable.

---

### Edge Cases

- What happens when the pool contains only free proxies, with no custom proxies present?
- What happens when one protocol has zero eligible proxies but stale usage history remains on disabled rows?
- How does the dashboard behave when the pool status API responds successfully but some statistic fields are zero or omitted?
- How does the interface distinguish between "available in pool", "currently displayed rows", and "historical usage" without implying they are the same measurement?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST define a single counting scope for each dashboard summary metric and use that scope consistently across the backend response and frontend rendering.
- **FR-002**: The system MUST present pool total, protocol totals, and custom proxy totals in a way that can be reconciled by an administrator without hidden arithmetic or undocumented exclusions.
- **FR-003**: The system MUST ensure dashboard protocol summaries do not contradict the protocol-filtered proxy tables for the same eligibility scope.
- **FR-004**: The system MUST distinguish between pool availability counts and per-proxy historical usage counts wherever both are shown in the interface.
- **FR-005**: The system MUST provide automated verification for the statistics contract that covers at least the key mixed-pool summary metrics used by the dashboard.
- **FR-006**: The system MUST audit existing dashboard statistics tied to proxy counts or protocol distribution and correct any additional frontend/backend mismatches discovered within the audited scope.
- **FR-007**: The system MUST preserve existing operational modes for free-only, custom-only, and mixed pool behavior while correcting the displayed statistics.

### Key Entities *(include if feature involves data)*

- **Pool Summary Metric**: A dashboard value representing total eligible proxies, protocol-specific eligible proxies, capacity slots, or custom proxy counts, each with a defined counting scope.
- **Proxy Usage Record**: Historical per-proxy usage information displayed in detailed tables, distinct from current pool eligibility counts.
- **Statistics Contract**: The documented relationship between backend fields and frontend labels for pool-related dashboard statistics.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a mixed pool containing both free and custom proxies, an administrator can reconcile the dashboard summary values without manual database inspection.
- **SC-002**: Automated checks cover the key pool summary contract and fail if the dashboard reintroduces inconsistent counting scope for total, protocol, or custom counts.
- **SC-003**: During verification, mixed-port traffic can be generated and the resulting dashboard interpretation no longer leads to a false conclusion that HTTP routing is entirely unused when HTTP proxies have recorded usage.
- **SC-004**: The audited pool-related dashboard statistics within scope have no remaining identified frontend/backend counting mismatches at completion.

## Assumptions

- The current mixed pool behavior and upstream selection rules are functionally correct; this feature focuses on statistics correctness and consistency.
- The WebUI remains the primary operator interface for diagnosing pool composition and usage.
- Audited scope is limited to pool- and proxy-related statistics currently exposed by the dashboard and its related API responses.
- Existing non-statistical UI behavior and subscription management flows are out of scope unless they are directly affected by the statistics contract.
