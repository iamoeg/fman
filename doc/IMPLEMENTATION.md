# Implementation Guide

Phase-by-phase implementation plan for `finmgmt`.

---

## Table of Contents

1. [Phase 1A — Foundation](#phase-1a--foundation-)
2. [Phase 1B — Domain Models & Tests](#phase-1b--domain-models--tests-)
3. [Phase 1C — Repositories](#phase-1c--repositories-)
4. [Phase 1D — Application Services](#phase-1d--application-services-)
5. [Phase 1E — Payroll Engine](#phase-1e--payroll-engine-)
6. [Phase 1F — TUI Implementation](#phase-1f--tui-implementation-)
7. [Phase 1G — Export & Polish](#phase-1g--export--polish)
8. [Things to Consider](#things-to-consider)

---

## Phase 1A — Foundation ✅

Project scaffolding, database setup, and basic TUI skeleton.

**Key decisions made here:**

- SQLite with TEXT UUIDs and RFC3339 timestamps
- goose for migrations; migrations embedded via `//go:embed` in `db/migration/embed.go` for distribution portability
- Money stored as integer cents
- Soft deletes (`deleted_at`) on all tables
- Foreign keys: CASCADE for aggregates, RESTRICT for historical artifacts
- XDG Base Directory compliance for config and data paths
- YAML config files (not environment variables)
- No `ports/` package — services define interfaces inline (Go idiom)
- All directory names singular (`migration`, `adapter`, `query`)

---

## Phase 1B — Domain Models & Tests ✅

Pure domain entities with validation, the `Money` type, and comprehensive tests.

**Key components:**

- `pkg/money/` — integer-cents Money type with overflow-safe arithmetic
- `internal/domain/organization.go` — Organization entity, LegalForm enum
- `internal/domain/employee.go` — Employee entity with age/hire date cross-validation; EmployeeCompensationPackage
- `internal/domain/payroll.go` — PayrollPeriod (DRAFT/FINALIZED workflow),
  PayrollResult with mathematical consistency validation and helpers (`TotalDueToCNSS`, `TotalEmployeeDeductions`)

---

## Phase 1C — Repositories ✅

Full SQLite repository layer: SQL queries, repository implementations, audit logging.

**Key components:**

- `db/query/` — 59 SQL queries across 6 entity types; primitive queries
  without soft-delete filtering (repository layer handles that)
- `internal/adapter/sqlite/` — 6 repositories: Organization, Employee,
  EmployeeCompensationPackage, PayrollPeriod, PayrollResult, AuditLog
- Utility helpers: `createAuditLog()`, `stringToNullString()`, conversion helpers
- AuditLog repository is read-only; audit entries are created inside other repositories' transactions

**Notable patterns established here:** transaction + audit log pattern,
primitive queries + repository-level filtering, immutable field protection
at SQL level, explicit workflow queries (`FinalizePayrollPeriod`).

---

## Phase 1D — Application Services ✅

Business logic orchestration layer and configuration system.

**Key components:**

- `pkg/config/` — XDG-compliant config loading/saving with YAML; default paths `~/.config/finmgmt/config.yaml` and `~/.local/share/finmgmt/data.db`
- `internal/application/organization_service.go`
- `internal/application/employee_service.go` — per-organization serial number generation
- `internal/application/compensation_package_service.go` — usage guards (cannot modify if referenced by employees or payroll results)
- `internal/application/payroll_service.go` — multi-repository coordination, DRAFT → FINALIZED workflow, batch result generation; contained a stub calculator replaced in Phase 1E

All services tested with mock repositories (no database needed).

---

## Phase 1E — Payroll Engine ✅

Moroccan payroll calculator adapter and integration into PayrollService.

**Key components:**

- `doc/DOMAIN.md` — complete specification of 2026 Moroccan payroll rules,
  validated against real payslips; two fully-worked examples serve as regression anchors
- `internal/adapter/payroll/calculator.go` — pure, stateless calculation engine

**Calculator implementation notes:**

- All year-specific values (rates, ceilings, brackets) live in `yearRates` struct entries in `ratesByYear` — no magic numbers in logic
- `completedYears()` uses truncated arithmetic (4y 11m = 4 completed years)
- Professional expense rate evaluated monthly using `gross × 12` as annual proxy; avoids needing year-to-date state
- `capAt()` helper keeps ceiling logic explicit and reusable
- Only net-to-pay is rounded; all intermediate values retain full cent precision

**Tests in `calculator_test.go`:**

- Per-step unit tests for every helper function (seniority rate boundaries, CNSS ceiling cases, IR brackets, etc.)
- Two integration tests derived directly from `DOMAIN.md` worked examples;
  every field in `PayrollResult` is asserted — effective regression guard when rates change

**Integration into PayrollService:**

- `payrollCalculator` interface defined in `payroll_service.go` (consumer-owned, Go-idiomatic)
- `PayrollService` has zero import of the calculator package; wiring happens in `cmd/tui/main.go`
- Stub method removed entirely once real calculator was wired in

---

## Phase 1F — TUI Implementation ✅

Four sections fully wired to application services, all with CRUD and validation.

**Architecture:**

- `sectionModel` interface: `Init`, `Update`, `View(w,h)`, `ShortHelp`, `IsOverlay`
- Root model routes key messages to the focused pane (sidebar or main)
- `IsOverlay() = true` suppresses all global keys (`q`, `tab`, `esc`) — section owns all input
- Footer renders `ShortHelp()` of the active pane + global bindings when not in overlay/filter mode
- All service calls are async; sections handle result messages and reload the list

**Sections:**

- **Organizations** — CRUD, set-active-org, active org persisted to config and reflected in sidebar
- **Compensation Packages** — CRUD + rename; org-scoped; immutable once referenced by a payroll result
- **Employees** — CRUD; 16-field form with scrollable viewport, cycling fields for enum values, org-scoped serial numbers
- **Payroll** — Period CRUD, generate results, finalize/unfinalize; drills into per-employee results view

**Notable patterns:**

- Form overlay: `lipgloss.Place(w, h, Center, Center, box)` centered over a dimmed list background
- `IsOverlay()` also returns `true` when the list filter is active — prevents `q`/`esc`/`tab` from firing global actions while the user is typing a search
- `activeOrgLoadedMsg{name, orgID}` — root model updates the sidebar AND forwards to all org-scoped sections so they reload for the new org
- One status row reserved at the bottom of each list view for error and success messages

---

## Phase 1G — Domain Hardening ✅

Year-keyed rate tables in the calculator and relaxed hire date validation.

### Calculator: year-keyed rate tables

- Introduced `yearRates` struct holding all legislation-specific values:
  CNSS rates, AMO rates, CNSS ceiling, professional expense thresholds,
  family allowance tiers, SMIG, IR brackets
- Replaced the flat `const` block with a `ratesByYear map[int]yearRates` registry;
  2026 is the sole entry
- `Calculate()` fails fast with `ErrUnsupportedPayrollYear` if no entry exists for `period.Year`
- All private helpers now accept `r yearRates` and read `r.Xyz` — algorithm is unchanged
- Seniority tiers stay package-level (structural, not year-specific)
- Adding a new year: one map entry in `ratesByYear`, no other code changes

### Employee: relaxed hire date constraint

- Removed `MaxHireYearsInPast` constant and the corresponding past-date check from `ValidateHireDate()`
- `ValidateHireDate()` now only rejects future hire dates
- Past hire dates are valid regardless of how far back;
  year correctness is enforced at the calculator level via `ErrUnsupportedPayrollYear`

---

## Phase 1H — Export & Polish

**Status:** Planned

### Goals

- Payslip PDF generation
- JSON/XML data export
- Backup and restore
- End-to-end testing
- Documentation polish

---

## Things to Consider

### Moroccan Payroll

- CNSS and AMO are legally distinct — keep them separate in `PayrollResult`; combine only via helpers
- Tax rates and brackets change yearly — add a new `yearRates` entry to `ratesByYear` in the calculator; see `DOMAIN.md` for the update workflow
- Professional expense rate switches at 78,000 MAD annual gross, evaluated monthly using `gross × 12`

### TUI Layer

- No business logic in the UI layer — all operations go through application services
- Translate service sentinel errors (e.g., `ErrPayrollPeriodAlreadyFinalized`) to user-friendly messages at this boundary
- TUI state is ephemeral; canonical state lives in the database

### Adding a New Year's Calculator

1. Update `doc/DOMAIN.md` with the new rates and brackets
2. Add a new entry to `ratesByYear` in `internal/adapter/payroll/calculator.go` with the updated `yearRates` values
3. Add integration tests in `calculator_test.go` with the new year's worked examples
4. `PayrollService`, `cmd/tui/main.go`, and the `payrollCalculator` interface require no changes
