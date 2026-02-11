# Implementation Guide

Phase-by-phase implementation guide for the financial management application.

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 1A - Foundation](#phase-1a---foundation-completed)
3. [Phase 1B - Domain Models & Tests](#phase-1b---domain-models--tests-completed)
4. [Phase 1C - Repositories](#phase-1c---repositories-completed)
5. [Phase 1D - Application Services](#phase-1d---application-services)
6. [Phase 1E - Payroll Engine](#phase-1e---payroll-engine)
7. [Phase 1F - TUI Implementation](#phase-1f---tui-implementation)
8. [Phase 1G - Export & Polish](#phase-1g---export--polish)
9. [Things to Consider](#things-to-consider)

---

## Overview

Development follows an iterative, phased approach:

- Start simple, prove concepts
- Build vertically (full stack for each feature)
- Test as you go
- Payroll is the priority (Phase 1 focus)

**Current Phase:** 1D - Application Services

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

## Phase 1C - Repositories ✅ COMPLETED

**Duration:** Weeks 3-4
**Status:** ✅ Complete

### Overview

Complete implementation of all repository layer components including SQL queries,
repository implementations, comprehensive testing, and utility functions.

### Delivered

#### 1. SQL Query Definitions ✅

All SQL query files created, reviewed, and sqlc code generated:

**Files Created:**

- `db/query/organization.sql` - Organization CRUD operations (8 queries)
- `db/query/employee.sql` - Employee CRUD operations (12 queries)
- `db/query/employee_compensation_package.sql` - Compensation package operations (8 queries)
- `db/query/payroll_period.sql` - Payroll period operations (15 queries)
- `db/query/payroll_result.sql` - Payroll result operations (11 queries)
- `db/query/audit_log.sql` - Audit trail operations (5 queries)

**Total:** 59 SQL queries across 6 entity types

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

#### 2. Repository Implementations ✅

**All 6 repositories fully implemented:**

1. **OrganizationRepository** (`organization_repo.go`)
   - Base pattern for all other repositories
   - Simple entity with no complex dependencies
   - ~40 test cases

2. **EmployeeRepository** (`employee_repo.go`)
   - Per-organization serial number generation
   - Multi-organization isolation
   - Immutable fields (org_id, serial_num)
   - Complex foreign key handling (CASCADE + RESTRICT)
   - ~50 test cases with Arabic name support

3. **EmployeeCompensationPackageRepository** (`compensation_package_repo.go`)
   - Historical artifact protection - cannot modify if in use
   - Usage guard pattern with `checkNotInUse()` helper
   - Checks both employees and payroll results before Update/Delete
   - Money precision preservation
   - ~45 test cases

4. **PayrollPeriodRepository** (`payroll_period_repo.go`)
   - Explicit workflow methods: `Finalize()` / `Unfinalize()`
   - Status transitions enforced at SQL level
   - UNIQUE constraint on (org_id, year, month)
   - Draft period filtering
   - ~60 test cases with comprehensive workflow testing

5. **PayrollResultRepository** (`payroll_result_repo.go`)
   - **No Update method** - immutable by design
   - 20+ money fields with exact precision
   - Most complex domain model
   - UNIQUE constraint on (payroll_period_id, employee_id)
   - Complete audit trail even for immutable records
   - ~40 test cases

6. **AuditLogRepository** (`audit_log_repo.go`)
   - Read-only repository - no Create/Update/Delete methods
   - Query methods: FindForRecord, FindRecent, FindByTable, FindByAction
   - Automatic creation via `createAuditLog()` helper
   - Infrastructure concern, not domain entity
   - ~25 test cases

#### 3. Utility Functions ✅

**Audit Logging Helper (`util.go`):**

- `createAuditLog()` - Creates audit trail for all mutations
- `DBActionEnum` - Type-safe action types (CREATE, UPDATE, DELETE, RESTORE, HARD_DELETE)
- Handles before/after JSON snapshots
- Special handling for `nil` values (HARD_DELETE uses "null" JSON)

**Conversion Helpers:**

- `stringToNullString()` / `nullStringToString()` - NULL handling
- `rowToEntity()` - Database row to domain entity
- `entityToCreateParams()` - Domain entity to SQL INSERT params
- `entityToUpdateParams()` - Domain entity to SQL UPDATE params

### Key Features Implemented

- ✅ Full CRUD operations with soft deletes
- ✅ Transaction support (BeginTx, WithTx, Rollback/Commit)
- ✅ Comprehensive audit logging for all mutations
- ✅ Money type conversions (integer cents ↔ domain Money)
- ✅ Foreign key relationship handling
- ✅ Query flexibility (with/without soft-deleted records)
- ✅ ~260+ test cases across all repositories
- ✅ Concurrency testing
- ✅ Edge case coverage
- ✅ Arabic name and Unicode support

### Design Patterns Established

#### 1. Transaction Pattern

```go
func (r *Repo) Mutate(ctx context.Context, entity *domain.Entity) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // No-op if already committed

    qtx := r.queries.WithTx(tx)

    // Perform mutation
    result, err := qtx.MutateEntity(ctx, params)
    if err != nil {
        return fmt.Errorf("failed to mutate: %w", err)
    }

    // Create audit log
    err = createAuditLog(ctx, qtx, "table_name", entity.ID, action, before, after)
    if err != nil {
        return fmt.Errorf("failed to create audit log: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

#### 2. Audit Logging Pattern

- **CREATE:** before=nil, after=created record
- **UPDATE:** before=old record, after=new record
- **DELETE:** before=active record, after=deleted record
- **RESTORE:** before=deleted record, after=restored record
- **HARD_DELETE:** before=deleted record, after="null" (JSON)

#### 3. Primitive Queries + Repository Filtering

- SQL queries don't filter `deleted_at` by default
- Repository methods choose filtered vs unfiltered queries
- Supports archive/restore features
- Requires discipline in implementation

#### 4. Immutable Field Protection

- Excluded from UPDATE queries at SQL level
- Examples: serial_num, org_id, (year, month)
- Tests verify fields cannot be changed

#### 5. Explicit Workflow Queries

- `FinalizePayrollPeriod` / `UnfinalizePayrollPeriod`
- Enforces business rules at database level
- Prevents accidental status changes

#### 6. Historical Artifact Protection

- Compensation packages in use cannot be modified
- `checkNotInUse()` helper before Update/Delete/HardDelete
- Protects audit trail integrity

#### 7. Money Field Conversions

- Domain: `money.Money` (type-safe operations)
- Database: `int64` (exact precision, no float errors)
- Conversion: `money.Cents()` → DB, `money.FromCents()` ← DB

### Testing Statistics

| Repository           | Test Cases | Lines of Test Code |
| -------------------- | ---------- | ------------------ |
| Organization         | ~40        | ~800               |
| Employee             | ~50        | ~1000              |
| Compensation Package | ~45        | ~900               |
| Payroll Period       | ~60        | ~1200              |
| Payroll Result       | ~40        | ~800               |
| Audit Log            | ~25        | ~500               |
| **TOTAL**            | **~260**   | **~5200**          |

### Key Bugs Found & Fixed

#### OrganizationRepository

1. HardDelete called wrong SQL function
2. Update used wrong "before" value in audit log
3. Delete missing error check
4. Missing transaction usage in Delete()
5. Audit log JSON validation for HARD_DELETE (needed "null" not "")
6. DBActionRestore missing from supported actions map

#### EmployeeRepository

1. FindByOrgAndSerialNum missing soft-delete filter
2. UpdatedAt using `emp.UpdatedAt` instead of `time.Now()`

#### Testing Issues

1. Migration failures (missing `goose.SetDialect("sqlite3")`)
2. UNIQUE constraint violations (needed atomic counters)
3. Test isolation (database sharing between subtests)
4. Timestamp comparison failures (timezone/precision)
5. Concurrency test race conditions with :memory: database

### Best Practices Established

✅ Always `defer tx.Rollback()` immediately after `BeginTx()`  
✅ Use qtx (transaction queries) not r.queries inside transactions  
✅ Check errors after every conversion (rowTo, toParams)  
✅ Create fresh database per test subtest for isolation  
✅ Use atomic counters for generating unique test data  
✅ Normalize timestamps (`.UTC().Truncate(time.Second)`) before comparing  
✅ Test with realistic data (Arabic names, special chars)  
✅ Document decisions while implementing, not after

### What's NOT Completed

**Application Services Layer** - Still need to implement:

1. OrganizationService - Business logic for organizations
2. EmployeeService - Employee CRUD + serial number generation
3. CompensationPackageService - Usage validation before mutations
4. PayrollService - Payroll generation, period finalization

**Configuration System** - Still need:

- XDG Base Directory compliance
- YAML configuration file parsing
- Database path management
- Config struct and loading

### Lessons Learned

#### What Worked Well

1. **Establishing Pattern First** - Organization repo as template saved time
2. **Comprehensive Testing** - Caught bugs immediately
3. **Realistic Test Data** - Moroccan context made tests meaningful
4. **Atomic Counters** - Solved UNIQUE constraint issues elegantly
5. **:memory: Databases** - Fast, isolated, auto-cleanup
6. **Documentation as You Go** - Summaries captured decisions while fresh

#### What Was Harder Than Expected

1. **Foreign Key Complexity** - Test setup more complex with dependencies
2. **Timestamp Handling** - UTC, Truncate, Sleep needed careful attention
3. **Copy-Paste Errors** - Review rounds essential to catch
4. **Transaction Patterns** - Took time to get rollback/commit right
5. **Concurrency Testing** - SQLite :memory: + goroutines = race conditions

### Time Investment

| Phase                | Estimated  | Actual     | Notes                            |
| -------------------- | ---------- | ---------- | -------------------------------- |
| Query Definitions    | 1 day      | 2 days     | Multiple review rounds needed    |
| Organization Repo    | 1 day      | 2 days     | Established patterns, found bugs |
| Employee Repo        | 1 day      | 1.5 days   | Complex but pattern established  |
| Compensation Package | 0.5 day    | 1 day      | Historical protection logic      |
| Payroll Period       | 1 day      | 1 day      | Workflow methods                 |
| Payroll Result       | 1 day      | 1 day      | Many fields, immutability        |
| Audit Log            | 0.5 day    | 0.5 day    | Simplest - read-only             |
| **TOTAL**            | **6 days** | **9 days** | Learning curve worth investment  |

---

## Phase 1D - Application Services

**Duration:** Weeks 5-6  
**Status:** 📋 Next

### Goals

- Implement application service layer
- Define service interfaces
- Orchestrate domain and repository operations
- Implement business logic coordination

### High-Level Tasks

1. **EmployeeService** - Employee management
   - Orchestrates employee repository
   - Generates serial numbers
   - Validates business rules
   - Returns domain errors

2. **OrganizationService** - Organization management
   - Simple passthrough to repository initially
   - May add business logic later

3. **CompensationPackageService** - Compensation management
   - Guards against modifying packages in use
   - Validates SMIG (minimum wage) compliance

4. **PayrollService** - Payroll processing
   - Most complex service
   - Coordinates multiple repositories
   - Integrates with payroll calculator (Phase 1E)
   - Manages period finalization workflow

5. **Configuration System**
   - XDG Base Directory compliance
   - YAML configuration loading
   - Database path management

6. **Service Tests**
   - Mock repositories
   - Test orchestration logic
   - Test error handling

**Target:** 4-5 days

---

## Phase 1E - Payroll Engine

**Duration:** Weeks 7-8  
**Status:** 📋 Planned

### Goals

- Research exact Moroccan payroll calculation rules
- Implement payroll calculator adapter
- Generate monthly payroll periods
- Create payslip PDF generation
- Build payroll TUI screens

### High-Level Tasks

1. Research and document exact Moroccan payroll calculation rules
2. Implement payroll calculator (`internal/adapter/payroll/morocco/`)
3. Integrate calculator with PayrollService
4. Implement PDF generation adapter (payslips)
5. Build TUI screens for payroll workflow
6. Comprehensive testing of calculations against known examples

**Details TBD when Phase 1D is complete.**

---

## Phase 1F - TUI Implementation

**Duration:** Week 9  
**Status:** 📋 Planned

### Goals

- Build Bubble Tea TUI screens
- Implement navigation
- Form input handling
- Data display (tables, details)

**Details TBD.**

---

## Phase 1G - Export & Polish

**Duration:** Week 10  
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

| Phase                | Status      | Duration  | Completion |
| -------------------- | ----------- | --------- | ---------- |
| 1A - Foundation      | ✅ Complete | Week 1    | 100%       |
| 1B - Domain & Tests  | ✅ Complete | Week 2    | 100%       |
| 1C - Repositories    | ✅ Complete | Weeks 3-4 | 100%       |
| 1D - App Services    | 📋 Next     | Weeks 5-6 | 0%         |
| 1E - Payroll Engine  | 📋 Planned  | Weeks 7-8 | 0%         |
| 1F - TUI             | 📋 Planned  | Week 9    | 0%         |
| 1G - Export & Polish | 📋 Planned  | Week 10   | 0%         |

**Overall Progress:** Phase 1C Complete (43%)
