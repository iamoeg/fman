# Implementation Guide

Phase-by-phase implementation guide for the financial management application.

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 1A - Foundation](#phase-1a---foundation-completed)
3. [Phase 1B - Domain Models & Tests](#phase-1b---domain-models--tests-completed)
4. [Phase 1C - Repositories & Application Services](#phase-1c---repositories--application-services-in-progress)
5. [Phase 1D - Payroll Engine](#phase-1d---payroll-engine)
6. [Phase 1E - TUI Implementation](#phase-1e---tui-implementation)
7. [Phase 1F - Export & Polish](#phase-1f---export--polish)
8. [Things to Consider](#things-to-consider)

---

## Overview

Development follows an iterative, phased approach:

- Start simple, prove concepts
- Build vertically (full stack for each feature)
- Test as you go
- Payroll is the priority (Phase 1 focus)

**Current Phase:** 1C - Repositories & Application Services

---

## Phase 1A - Foundation ✅ COMPLETED

**Duration:** Week 1  
**Status:** ✅ Complete

### Goals

- Project structure setup
- SQLite database with migrations
- Basic Bubble Tea app skeleton
- Main menu and navigation

### Delivered

- [x] Project initialized with Go modules
- [x] Directory structure created (hexagonal architecture, Go-idiomatic)
- [x] Database schema designed (5 core tables)
- [x] SQL migration created and tested (`db/migration/00001_initial_schema.sql`)
- [x] goose migration tool integrated
- [x] Basic Bubble Tea TUI skeleton running

### Key Decisions

- **Database:** SQLite with TEXT UUIDs and timestamps
- **Migration tool:** goose
- **Money representation:** Integer cents
- **Soft deletes:** `deleted_at` on all tables
- **Foreign keys:** CASCADE for aggregates, RESTRICT for historical artifacts
- **XDG compliance:** Config and data follow XDG spec
- **Configuration:** YAML files (not environment variables)
- **No `ports/` package:** Go idiom - services define interfaces inline
- **Directory names:** All singular (`migration`, `adapter`, `model`)

---

## Phase 1B - Domain Models & Tests ✅ COMPLETED

**Duration:** Week 2  
**Status:** ✅ Complete

### Goals

- Create domain models (pure business logic)
- Implement comprehensive validation
- Create Money type for financial calculations
- Write extensive tests

### Delivered

#### 1. Money Type (`pkg/money/`) ✅

Complete implementation of Money type with:

- Integer cents storage (no floating-point precision errors)
- Safe arithmetic operations with overflow checking
- Error handling for division by zero, NaN, Inf
- Currency support (MAD)
- Comprehensive methods: Add, Subtract, Multiply, Divide
- Comparison methods: Equals, LessThan, GreaterThan, IsZero, IsPositive, IsNegative
- String formatting

**Files:**

- `pkg/money/money.go` - Core Money implementation
- `pkg/money/currency.go` - Currency type and validation

#### 2. Domain Models (`internal/domain/`) ✅

**Organization (`organization.go`):**

- Core organization entity
- Legal form enum (SARL)
- Validation for required fields
- Moroccan business identifier fields (ICE, IF, RC, CNSS, RIB)

**Employee (`employee.go`):**

- Complete employee entity
- Gender enum (MALE, FEMALE)
- Marital status enum (SINGLE, MARRIED, SEPARATED, DIVORCED, WIDOWED)
- Age validation (16-80 years)
- Hire date validation (within last year + not in future)
- Cross-field validation (hired at minimum age 16)
- CIN and CNSS number fields
- Employee serial number (per-organization numbering)

**EmployeeCompensationPackage (`employee.go`):**

- Immutable compensation records
- SMIG (minimum wage) validation
- Currency validation
- Historical artifact preservation

**PayrollPeriod (`payroll.go`):**

- Monthly payroll cycle container
- Status enum (DRAFT, FINALIZED)
- State consistency validation
- Year/month validation
- Finalization timestamp logic

**PayrollResult (`payroll.go`):**

- Complete payroll calculation record
- All salary components (base, seniority, bonuses)
- CNSS contributions (employee + employer)
- AMO contributions (employee + employer)
- Tax calculations (exemptions, taxable amounts, IR)
- Mathematical consistency validation
- Rounding amount constraints (±100 cents)
- Helper methods: TotalDueToCNSS(), TotalEmployeeDeductions()

#### 3. Comprehensive Tests ✅

**Organization Tests (`organization_test.go`):**

- 30+ test scenarios
- Valid/invalid cases for all fields
- Enum validation tests
- Benchmark tests
- Uses table-driven test pattern

**Employee Tests (`employee_test.go`):**

- 100+ test scenarios
- Full validation coverage
- Age boundary testing
- Date cross-validation
- Unicode name support (Arabic, French)
- CompensationPackage validation
- Benchmark tests

**Payroll Tests (`payroll_test.go`):**

- 70+ test scenarios
- PayrollPeriod state consistency testing
- PayrollResult mathematical consistency
- All formulas validated
- Helper method testing
- Benchmark tests

### Key Achievements

✅ **Robust Money Type:**

- No floating-point precision issues
- Overflow protection
- Type-safe operations
- Comprehensive error handling

✅ **Complete Domain Validation:**

- All business rules enforced
- Cross-field validations
- Moroccan-specific requirements
- Clear, descriptive error messages

✅ **Excellent Test Coverage:**

- Table-driven tests
- Parallel execution
- Edge case coverage
- Realistic test data
- Benchmark tests for performance

✅ **Production-Ready Code:**

- Well-documented
- Idiomatic Go
- Clear separation of concerns
- Maintainable and extensible

### Lessons Learned

1. **Money Type First:** Building the Money type first was crucial - everything depends on it
2. **Validation Complexity:** Cross-field validation (e.g., hire date vs birth date) requires careful thought
3. **Moroccan Context:** Domain models need to reflect Moroccan legal/business requirements (CIN, CNSS, SMIG)
4. **Mathematical Consistency:** PayrollResult validation is complex but critical for correctness
5. **CNSS vs AMO:** Keeping them separate in totals while combining for payments is the right design
6. **Test Data Quality:** Using realistic Moroccan data (names, amounts) improves test quality
7. **Documentation Matters:** Comprehensive docstrings make the code much more maintainable

---

## Phase 1C - Repositories & Application Services 🔄 IN PROGRESS

**Duration:** Weeks 3-4  
**Status:** 🔄 In Progress

### Progress Overview

#### ✅ Completed: SQL Query Definitions

All SQL query files have been created, reviewed, and are ready for sqlc code generation.

**Files Created:**

- `db/query/organization.sql` - Organization CRUD operations
- `db/query/employee.sql` - Employee CRUD operations
- `db/query/employee_compensation_package.sql` - Compensation package operations
- `db/query/payroll_period.sql` - Payroll period operations
- `db/query/payroll_result.sql` - Payroll result operations
- `db/query/audit_log.sql` - Audit trail operations

**Query Pattern Established:**

All entity query files follow a consistent pattern:

1. **Get Operations:**
   - `Get{Entity}` - Single record by ID, active only
   - `Get{Entity}IncludingDeleted` - Single record by ID, includes soft-deleted

2. **List Operations:**
   - `List{Entities}` - All records, active only
   - `List{Entities}IncludingDeleted` - All records, includes soft-deleted
   - `List{Entities}By{Criteria}` - Filtered lists with variations

3. **Mutation Operations:**
   - `Create{Entity}` - Insert new record (returns created record)
   - `Update{Entity}` - Update existing record (returns updated record)
   - `Delete{Entity}` - Soft delete (sets deleted_at)
   - `Restore{Entity}` - Undelete (clears deleted_at, returns restored record)
   - `HardDelete{Entity}` - Permanent deletion (use with extreme caution)

4. **Specialized Operations:**
   - Entity-specific queries (e.g., `GetNextSerialNumber` for employees)
   - Relationship queries (e.g., `CountEmployeesUsingCompensationPackage`)

**Key Design Decisions Made:**

1. **Primitive Queries with Repository-Level Filtering:**
   - SQL queries are "primitives" - they don't filter deleted_at by default
   - Repository layer decides whether to use filtered or unfiltered queries
   - Supports user requirement: view/restore archived data

2. **Immutable Core Fields:**
   - `org_id`, `serial_num` excluded from employee UPDATE
   - `org_id`, `year`, `month` excluded from payroll_period UPDATE
   - These fields define identity and should never change

3. **Explicit Workflow Queries:**
   - `FinalizePayrollPeriod` / `UnfinalizePayrollPeriod` instead of generic UPDATE
   - Enforces business workflow at query level
   - Prevents accidental status changes

4. **Audit Trail Queries:**
   - `ListAuditLogsForRecord` - View complete history for a record
   - `ListAuditLogsRecent` - View recent changes system-wide
   - `ListAuditLogsByTable` / `ListAuditLogsByAction` - Filtered views
   - `CreateAuditLog` - Repository layer inserts audit records in transactions

5. **Consistent Naming Convention:**
   - All queries use `-- name: QueryName :type` format
   - `:one` returns single record or error
   - `:many` returns slice of records
   - `:exec` executes without returning data

**Query Statistics:**

| Entity                        | Queries | GET | LIST | CREATE | UPDATE | DELETE | SPECIAL |
| ----------------------------- | ------- | --- | ---- | ------ | ------ | ------ | ------- |
| Organization                  | 8       | 2   | 2    | 1      | 1      | 3      | 0       |
| Employee                      | 12      | 3   | 4    | 1      | 1      | 3      | 2       |
| Employee Compensation Package | 8       | 2   | 2    | 1      | 1      | 3      | 0       |
| Payroll Period                | 15      | 4   | 6    | 1      | 2      | 3      | 0       |
| Payroll Result                | 11      | 2   | 6    | 1      | 0      | 3      | 1       |
| Audit Log                     | 5       | 0   | 4    | 1      | 0      | 0      | 0       |
| **TOTAL**                     | **59**  | 13  | 24   | 6      | 5      | 15     | 3       |

**Lessons Learned:**

1. **Review is Essential:** Multiple rounds of review caught:
   - Copy-paste errors (wrong table names, wrong WHERE clauses)
   - Missing fields in INSERT statements
   - Typos in field names
   - Missing RETURNING clauses
   - Inconsistent filtering logic

2. **Primitives vs Safety:** Chose primitive queries (no built-in filtering) over safe-by-default queries
   - Provides flexibility for archive/restore features
   - Requires discipline in repository layer
   - Document which queries to use when

3. **Consistency Matters:** Establishing patterns early (naming, structure, annotations) makes maintenance easier

4. **Special Cases Need Special Queries:**
   - Employee serial number generation
   - Payroll period finalization workflow
   - Compensation package usage checking
   - All handled with dedicated queries

5. **Field Order in INSERT Matters:** Keep consistent with schema definition to avoid confusion

#### 📋 Next: Repository Implementation

**Tasks:**

1. Implement repository layer (`internal/adapter/sqlite/`)
2. Implement application services (`internal/application/`)
3. Implement audit logging helper
4. Write repository tests (using `:memory:`)
5. Write service tests (using mocks)
6. Configuration system (XDG paths)

**Target:** Week 3-4

---

## Phase 1D - Payroll Engine

**Duration:** Weeks 5-6  
**Status:** 📋 Planned

### Goals

- Implement Moroccan payroll calculator
- Generate monthly payroll periods
- Create payslip PDF generation
- Build payroll TUI screens

### High-Level Tasks

1. Research and document exact Moroccan payroll calculation rules
2. Implement payroll calculator (`internal/adapter/payroll/morocco/`)
3. Implement PayrollService
4. Implement PDF generation adapter (payslips)
5. Build TUI screens for payroll workflow
6. Comprehensive testing of calculations against known examples

**Details TBD when Phase 1C is complete.**

---

## Phase 1E - TUI Implementation

**Duration:** Week 7  
**Status:** 📋 Planned

### Goals

- Build Bubble Tea TUI screens
- Implement navigation
- Form input handling
- Data display (tables, details)

**Details TBD.**

---

## Phase 1F - Export & Polish

**Duration:** Week 8  
**Status:** 📋 Planned

### Goals

- JSON/XML export
- Data backup/restore
- Error handling polish
- Documentation
- End-to-end testing

**Details TBD.**

---

## Things to Consider

### Error Handling

- Define sentinel errors early (`ErrNotFound`, `ErrDuplicate`)
- Use `errors.Is()` and `errors.As()` for checking
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- TUI will translate technical errors to user-friendly messages

### UUID Generation

- Always generate in application code (service layer)
- Never in database
- Use `uuid.New()` from `google/uuid`

### Timestamp Handling

- Always store UTC: `time.Now().UTC()`
- Format as RFC3339: `time.Format(time.RFC3339)`
- Parse when reading: `time.Parse(time.RFC3339, s)`

### Employee Serial Numbers

- Generate in application service
- Logic: `SELECT MAX(serial_num) FROM employee WHERE org_id = ? + 1`
- Handle race conditions with UNIQUE constraint

### Database Connections

- **CRITICAL:** Enable foreign keys: `PRAGMA foreign_keys = ON`
- Do this immediately after opening connection
- Applies to both production and tests

### Testing Philosophy

- Test domain validation logic (no database needed)
- Test repository operations (use `:memory:`)
- Test service orchestration (use mocks)
- Integration tests for critical paths

### Configuration Management

- Use XDG Base Directory specification
- Config: `~/.config/finmgmt/config.yaml`
- Data: `~/.local/share/finmgmt/data.db`
- Never hardcode paths

### Moroccan Payroll Specifics

- CNSS and AMO are separate (but AMO collected by CNSS)
- Tax rates and brackets change yearly
- Professional expense deductions have caps
- Family allowances based on dependents
- Rounding to nearest dirham (100 cents)

---

## Progress Summary

| Phase                 | Status      | Duration  | Completion |
| --------------------- | ----------- | --------- | ---------- |
| 1A - Foundation       | ✅ Complete | Week 1    | 100%       |
| 1B - Domain & Tests   | ✅ Complete | Week 2    | 100%       |
| 1C - Repos & Services | ⏳ Next     | Weeks 3-4 | 0%         |
| 1D - Payroll Engine   | 📋 Planned  | Weeks 5-6 | 0%         |
| 1E - TUI              | 📋 Planned  | Week 7    | 0%         |
| 1F - Export & Polish  | 📋 Planned  | Week 8    | 0%         |

**Overall Progress:** Phase 1B Complete (25%)
