# Implementation Guide

Phase-by-phase implementation plan for `finmgmt`.

---

## Table of Contents

1. [Phase 1A — Foundation](#phase-1a--foundation-)
2. [Phase 1B — Domain Models & Tests](#phase-1b--domain-models--tests-)
3. [Phase 1C — Repositories](#phase-1c--repositories-)
4. [Phase 1D — Application Services](#phase-1d--application-services-)
5. [Phase 1E — Payroll Engine](#phase-1e--payroll-engine-)
6. [Phase 1F — TUI Implementation](#phase-1f--tui-implementation)
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

- All rates are named constants — no magic numbers in logic
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

## Phase 1F — TUI Implementation

**Status:** In progress

### Goals

- Wire services into TUI screens
- Employee list and detail views
- Payroll period workflow (create, generate, finalize)
- Form input with validation feedback
- Navigation between screens

---

## Phase 1G — Export & Polish

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
- Tax rates and brackets change yearly — see `DOMAIN.md` for the update workflow
- Professional expense rate switches at 78,000 MAD annual gross, evaluated monthly using `gross × 12`

### TUI Layer

- No business logic in the UI layer — all operations go through application services
- Translate service sentinel errors (e.g., `ErrPayrollPeriodAlreadyFinalized`) to user-friendly messages at this boundary
- TUI state is ephemeral; canonical state lives in the database

### Adding a New Year's Calculator

1. Create `internal/adapter/payroll/<year>/` alongside the existing package
2. Update constants (rates, brackets) — the `DOMAIN.md` update workflow covers which constants to change
3. Wire the new calculator in `cmd/tui/main.go`
4. `PayrollService` requires no changes
