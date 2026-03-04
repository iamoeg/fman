# Implementation Guide

Phase-by-phase implementation guide for the financial management application.

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 1A - Foundation](#phase-1a---foundation--completed)
3. [Phase 1B - Domain Models & Tests](#phase-1b---domain-models--tests--completed)
4. [Phase 1C - Repositories](#phase-1c---repositories--completed)
5. [Phase 1D - Application Services](#phase-1d---application-services--completed)
6. [Phase 1E - Payroll Engine](#phase-1e---payroll-engine--completed)
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

**Current Phase:** 1E - Payroll Engine ✅ Complete

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
- [x] goose migration tool integrated (migrations embedded via embed.FS for distribution portability)
- [x] Basic Bubble Tea TUI skeleton running

### Key Decisions

- **Database:** SQLite with TEXT UUIDs and timestamps
- **Migration tool:** goose
- **Migration embedding:** `//go:embed` via `db/migration/embed.go` for distribution portability
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

## Phase 1D - Application Services ✅ COMPLETED

**Duration:** Weeks 5-6  
**Status:** ✅ Complete

### Overview

Complete implementation of the application service layer,
providing business logic orchestration between the domain and repository layers.
All four core services implemented with comprehensive test coverage using mock repositories.

### Delivered

#### 1. Configuration System (`pkg/config/`)

**Purpose:** Manage application configuration with XDG Base Directory compliance

**Key Features:**

- XDG Base Directory specification compliance
- Environment variable overrides (`FINMGMT_*`)
- Automatic directory creation
- Config validation before save
- YAML configuration format
- CLI flag support (Cobra integration)

**Implementation Details:**

- Uses `adrg/xdg` package for proper XDG compliance
- Viper as local instance (non-global) for clean testing
- Default paths: `~/.config/finmgmt/config.yaml`, `~/.local/share/finmgmt/data.db`
- 11 test functions covering 20+ scenarios
- ~95% test coverage

**Files:**

- `pkg/config/config.go` (~300 lines)
- `pkg/config/config_test.go` (~350 lines)

**API:**

```go
// Core functions
Default() *Config
Load(configPath string) (*Config, error)
LoadOrCreate(configPath string) (*Config, error)
Save(configPath string) error
Validate() error
ResolveDatabasePath() (string, error)

// Helpers
ConfigDir() string
DataDir() string
```

#### 2. OrganizationService (`internal/application/organization_service.go`)

**Purpose:** Business logic for organization management

**Key Features:**

- UUID generation (flexible nil-check pattern)
- Timestamp management (CreatedAt, UpdatedAt)
- Domain validation before persistence
- Error translation (repository -> service errors)
- Duplicate detection for business identifiers
- Archive/restore support

**Implementation Details:**

- 9 public methods (Create, Update, Delete, Restore, HardDelete, Get, GetIncludingDeleted, List, ListIncludingDeleted)
- Small repository interface (8 methods)
- ~80 test cases with mock repository
- ~95% test coverage
- 3 benchmark tests

**Files:**

- `internal/application/organization_service.go` (~400 lines)
- `internal/application/organization_service_test.go` (~800 lines)

**Error Handling:**

```go
var (
    ErrOrganizationNotFound = errors.New("organization not found")
    ErrOrganizationExists   = errors.New("organization already exists")
)
```

#### 3. EmployeeService (`internal/application/employee_service.go`)

**Purpose:** Business logic for employee management with serial number generation

**Key Features:**

- Per-organization serial number generation
- Multi-tenant isolation
- Compensation package relationship management
- Hire date validation
- Age-based validation
- Archive/restore support

**Implementation Details:**

- 11 public methods (Create with serial number generation, Update, Delete, Restore, HardDelete, Get, GetIncludingDeleted, GetBySerialNum, List, ListByOrganization, ListIncludingDeleted)
- Atomic serial number generation (race-safe)
- Repository interface (10 methods)
- ~90 test cases with mock repository
- ~95% test coverage
- 3 benchmark tests

**Files:**

- `internal/application/employee_service.go` (~550 lines)
- `internal/application/employee_service_test.go` (~1100 lines)

**Serial Number Pattern:**

```go
func (s *EmployeeService) CreateEmployee(ctx, emp) error {
    // Get next serial number for organization
    serialNum, err := s.employees.GetNextSerialNumber(ctx, emp.OrgID)
    if err != nil {
        return fmt.Errorf("failed to get serial number: %w", err)
    }
    emp.SerialNum = serialNum

    // ... rest of creation logic
}
```

**Error Handling:**

```go
var (
    ErrEmployeeNotFound = errors.New("employee not found")
    ErrEmployeeExists   = errors.New("employee already exists")
)
```

#### 4. CompensationPackageService (`internal/application/compensation_package_service.go`)

**Purpose:** Business logic for compensation package management with historical protection

**Key Features:**

- Historical artifact protection (cannot modify if in use)
- Usage validation before Update/Delete
- SMIG (minimum wage) validation
- Money precision handling
- Archive/restore support

**Implementation Details:**

- 9 public methods (Create, Update with usage guards, Delete, Restore, HardDelete, Get, GetIncludingDeleted, List, ListIncludingDeleted)
- Usage guard pattern checks both employees and payroll results
- Repository interface (9 methods)
- ~85 test cases with mock repository
- ~95% test coverage
- 3 benchmark tests

**Files:**

- `internal/application/compensation_package_service.go` (~500 lines)
- `internal/application/compensation_package_service_test.go` (~1000 lines)

**Usage Guard Pattern:**

```go
func (s *CompensationPackageService) UpdateCompensationPackage(ctx, pkg) error {
    // Check if package is in use
    empCount, _ := s.packages.CountEmployeesUsing(ctx, pkg.ID)
    resultCount, _ := s.packages.CountPayrollResultsUsing(ctx, pkg.ID)

    if empCount > 0 || resultCount > 0 {
        return ErrCompensationPackageInUse
    }

    // Proceed with update...
}
```

**Error Handling:**

```go
var (
    ErrCompensationPackageNotFound = errors.New("compensation package not found")
    ErrCompensationPackageExists   = errors.New("compensation package already exists")
    ErrCompensationPackageInUse    = errors.New("compensation package is in use")
)
```

#### 5. PayrollService (`internal/application/payroll_service.go`)

**Purpose:** Orchestrate payroll lifecycle from period creation through result generation and finalization

**Key Features:**

- Multi-repository coordination (4 repositories)
- Workflow state management (DRAFT -> FINALIZED)
- Batch payroll generation for all employees
- Period finalization with validation
- Immutable results enforcement
- Stub calculation engine (Phase 1E integration point)

**Implementation Details:**

- 23 public methods across period and result management
- Coordinates 4 repositories (periods, results, employees, compensation packages)
- Repository interfaces (4 total, 30+ methods combined)
- ~50 test cases with mock repositories
- ~95% test coverage
- Most complex service in the application

**Files:**

- `internal/application/payroll_service.go` (~850 lines)
- `internal/application/payroll_service_test.go` (~1400 lines)

**Workflow Operations:**

```go
// Period Lifecycle
CreatePayrollPeriod(ctx, period) error        // Create DRAFT period
GeneratePayrollResults(ctx, periodID) error   // Generate results for all employees
FinalizePayrollPeriod(ctx, periodID) error    // Lock period (DRAFT -> FINALIZED)
UnfinalizePayrollPeriod(ctx, periodID) error  // Unlock for corrections

// Validation Rules
- Cannot delete finalized periods
- Cannot finalize empty periods
- Cannot generate results for finalized periods
- Results are immutable (no Update operation)
```

**Phase 1E Integration Point:**

```go
// Phase 1D: Stub implementation
func (s *PayrollService) calculatePayrollStub(
    period *domain.PayrollPeriod,
    emp *domain.Employee,
    pkg *domain.EmployeeCompensationPackage,
) (*domain.PayrollResult, error) {
    // Returns result with all zero values
    // Phase 1E will replace with real Moroccan calculator
}
```

**Error Handling:**

```go
var (
    // Payroll Period
    ErrPayrollPeriodNotFound         = errors.New("payroll period not found")
    ErrPayrollPeriodExists           = errors.New("payroll period already exists")
    ErrPayrollPeriodAlreadyFinalized = errors.New("payroll period is already finalized")
    ErrPayrollPeriodNotFinalized     = errors.New("payroll period is not finalized")
    ErrPayrollPeriodEmpty            = errors.New("payroll period has no results")
    ErrPayrollCalculationFailed      = errors.New("payroll calculation failed")

    // Payroll Result
    ErrPayrollResultNotFound = errors.New("payroll result not found")
    ErrPayrollResultExists   = errors.New("payroll result already exists")
)
```

### Testing Statistics

**Total Test Coverage Across All Services:**

| Service                    | Test Cases | Lines of Test Code | Coverage |
| -------------------------- | ---------- | ------------------ | -------- |
| Config                     | ~11        | ~350               | ~95%     |
| OrganizationService        | ~80        | ~800               | ~95%     |
| EmployeeService            | ~90        | ~1100              | ~95%     |
| CompensationPackageService | ~85        | ~1000              | ~95%     |
| PayrollService             | ~50        | ~1400              | ~95%     |
| **TOTAL**                  | **~316**   | **~4650**          | **~95%** |

### Key Achievements

**Complete Service Layer Implementation:**

- All 4 core services fully implemented
- Configuration system with XDG compliance
- Clean separation of concerns
- Consistent patterns across all services

**Comprehensive Testing:**

- 300+ test cases using mock repositories
- Fast execution (no database setup)
- Isolated tests (service logic only)
- Edge case coverage

**Production-Ready Error Handling:**

- Service-level sentinel errors
- Proper error translation from repository layer
- Clear, actionable error messages
- Wrapped errors with context

**Flexible Design Patterns:**

- UUID generation (nil-check pattern)
- Timestamp management (CreatedAt, UpdatedAt)
- Archive support (IncludingDeleted variants)
- Usage guards (CompensationPackage)
- Workflow state management (PayrollPeriod)

**Clean Phase 1E Integration:**

- Stub calculation engine in place
- Single method to replace for Phase 1E
- All orchestration logic complete
- Moroccan calculator will plug in cleanly

### Design Patterns Established

#### 1. Repository Interface Pattern

Services define small, focused interfaces for what they need:

```go
// In OrganizationService
type organizationRepository interface {
    Create(ctx, org) error
    Update(ctx, org) error
    Delete(ctx, id) error
    Restore(ctx, id) error
    HardDelete(ctx, id) error
    FindByID(ctx, id) (*domain.Organization, error)
    FindByIDIncludingDeleted(ctx, id) (*domain.Organization, error)
    FindAll(ctx) ([]*domain.Organization, error)
}

// SQLite repository implicitly satisfies this interface
```

**Benefits:**

- Service only depends on methods it uses
- Easy to mock (fewer methods)
- Clear dependencies
- Can have multiple services use same repo differently

#### 2. Error Translation Pattern

```go
func (s *Service) Operation(ctx, params) error {
    result, err := s.repo.RepositoryMethod(ctx, params)
    if err != nil {
        // Translate repository errors to service errors
        if errors.Is(err, sqlite.ErrRecordNotFound) {
            return ErrEntityNotFound
        }
        if errors.Is(err, sqlite.ErrDuplicateRecord) {
            return ErrEntityExists
        }
        // Wrap other errors with context
        return fmt.Errorf("failed to do operation: %w", err)
    }
    return result, nil
}
```

**Benefits:**

- UI layer doesn't know about SQLite
- Database can be swapped
- Business-appropriate error messages
- Errors carry context

#### 3. UUID Generation Pattern

```go
func (s *Service) Create(ctx, entity) error {
    // Flexible: Use provided UUID or generate new one
    if entity.ID == uuid.Nil {
        entity.ID = uuid.New()
    }

    // Set timestamps
    now := time.Now().UTC()
    entity.CreatedAt = now
    entity.UpdatedAt = now

    // Validate and persist...
}
```

**Benefits:**

- Caller can access generated ID without separate return value
- Useful for testing (can provide deterministic UUIDs)
- Follows common Go database patterns

#### 4. Mock Repository Pattern

```go
type mockEmployeeRepository struct {
    createFunc           func(context.Context, *domain.Employee) error
    findByIDFunc         func(context.Context, uuid.UUID) (*domain.Employee, error)
    getNextSerialNumFunc func(context.Context, uuid.UUID) (int, error)
    // ... other methods
}

func (m *mockEmployeeRepository) Create(ctx, emp) error {
    if m.createFunc != nil {
        return m.createFunc(ctx, emp)
    }
    return nil  // Default behavior
}

// Usage in test
mockRepo := &mockEmployeeRepository{
    findByIDFunc: func(ctx, id) (*domain.Employee, error) {
        return testEmployee, nil
    },
}
service := NewEmployeeService(mockRepo, mockCompRepo)
```

**Benefits:**

- Fast tests (no database)
- Easy edge case testing
- Service logic tested in isolation
- Can simulate any repository behavior

#### 5. Usage Guard Pattern

```go
func (s *CompensationPackageService) Update(ctx, pkg) error {
    // Check if package is referenced
    empCount, _ := s.packages.CountEmployeesUsing(ctx, pkg.ID)
    resultCount, _ := s.packages.CountPayrollResultsUsing(ctx, pkg.ID)

    if empCount > 0 || resultCount > 0 {
        return ErrCompensationPackageInUse
    }

    // Safe to update - not in use
    return s.packages.Update(ctx, pkg)
}
```

**Benefits:**

- Protects historical data integrity
- Prevents audit trail corruption
- Business rule enforcement
- Clear error messages

### Lessons Learned

#### What Worked Well

1. **Mock Repository Testing** - Fast, focused, comprehensive
2. **Error Translation** - Clean abstraction between layers
3. **UUID Nil-Check Pattern** - Flexible and testable
4. **Small Repository Interfaces** - Easy to mock and maintain
5. **Consistent Patterns** - Each service follows same structure
6. **Documentation While Building** - Captured decisions when fresh

#### Challenges Encountered

1. **Mock Repository Verbosity** - Many methods to implement, but worth it
2. **Error Translation Completeness** - Need to handle all repository errors
3. **Test Organization** - Kept tests in separate package (`application_test`) for black-box testing
4. **PayrollService Complexity** - 4 repository coordination required careful orchestration

#### Time Investment

| Component              | Estimated    | Actual     | Notes                          |
| ---------------------- | ------------ | ---------- | ------------------------------ |
| Config System          | 1 day        | 1 day      | adrg/xdg simplified things     |
| OrganizationService    | 1 day        | 1.5 days   | Established service patterns   |
| EmployeeService        | 1.5 days     | 2 days     | Serial number generation logic |
| CompensationPkgService | 1 day        | 1 day      | Usage guard pattern            |
| PayrollService         | 2 days       | 2.5 days   | Most complex, 4 repos          |
| **TOTAL**              | **6.5 days** | **8 days** | Learning curve, worth it       |

### Integration Checklist

When integrating into main application:

- [x] Config system (`pkg/config/`)
- [x] All service implementations (`internal/application/`)
- [x] All service tests (`internal/application/*_test.go`)
- [x] Error definitions and translations
- [x] Mock repository implementations
- [ ] Wire config loading into `cmd/tui/main.go`
- [ ] Add Cobra CLI flags (`--config`)
- [ ] Initialize services in application startup
- [ ] Connect services to TUI layer (Phase 1F)

---

## Phase 1E - Payroll Engine ✅ COMPLETED

**Duration:** Weeks 7-8  
**Status:** ✅ Complete

### Goals

- ✅ Research and document exact Moroccan payroll calculation rules
- ✅ Implement payroll calculator adapter
- ✅ Integrate calculator with PayrollService
- ~Payslip PDF generation~ (deferred to Phase 1G)
- Payroll TUI screens (Phase 1F)

### Delivered

#### 1. Moroccan Payroll Domain Documentation (`DOMAIN.md`) ✅

Complete specification of all 2026 Moroccan payroll calculation rules,
validated against real payslips. Covers:

- CNSS components: Social Allowance (Prestations Sociales),
  Job Loss Compensation (Indemnité de Perte d'Emploi - IPE),
  Training Tax, Family Benefits (Allocations Familiales)
- Health Insurance (Assurance Maladie Obligatoire - AMO) contributions
- Income Tax (Impôt sur le Revenu - IR) brackets with annualization method
- Professional expense deduction (rate switch at 78,000 MAD annual gross)
- Family charge deduction (40 MAD/month per dependent, max 6)
- Seniority bonus tiers (0%–25%)
- SMIG minimum wage enforcement
- Rounding rules (nearest dirham)
- Two fully worked examples validated against real payslips

#### 2. Moroccan Payroll Calculator (`internal/adapter/payroll/`) ✅

**File:** `calculator.go`

A pure calculation engine with no I/O dependencies. Implements the `payrollCalculator` interface defined in `PayrollService`.

**Calculation order (matches DOMAIN.md):**

1. Base salary
2. Seniority bonus (from `emp.HireDate` and period date)
3. Gross salary
4. CNSS employee contributions (Prestations Sociales + IPE, capped at 6,000 MAD)
5. AMO employee contribution (no ceiling)
6. Professional expense deduction (monthly evaluation, 35% or 20% rate, 2,500 MAD cap)
7. Family charge deduction (from `emp.NumDependents`)
8. Net taxable salary
9. IR (annualized progressive brackets, divided by 12)
10. Net to pay (rounded to nearest dirham)
11. Employer contributions (CNSS + AMO)

**Key implementation details:**

- All rates defined as named constants — zero magic numbers in logic
- `completedYears()` uses truncated arithmetic (4 years 11 months = 4)
- Professional expense rate evaluated monthly using `gross × 12` as annual proxy
- Only net to pay is rounded; all intermediate values retain full cent precision
- `capAt()` helper keeps ceiling logic explicit and reusable

```go
// Public API — satisfies payrollCalculator interface in PayrollService
type Calculator struct{}

func New() *Calculator

func (c *Calculator) Calculate(
    ctx context.Context,
    period *domain.PayrollPeriod,
    emp *domain.Employee,
    pkg *domain.EmployeeCompensationPackage,
) (*domain.PayrollResult, error)
```

#### 3. Calculator Tests (`calculator_test.go`) ✅

Two test categories:

**Per-step unit tests** — each helper function tested in isolation:

- `TestCompletedYears` — boundary and anniversary edge cases
- `TestSeniorityRate` — all tier boundaries
- `TestCalculateSeniorityBonus`
- `TestCalculateCNSSEmployee` — below/at/above ceiling cases
- `TestCalculateCNSSEmployer`
- `TestCalculateAMOEmployee`
- `TestCalculateProfessionalExpenseDeduction` — rate switch and cap cases
- `TestCalculateFamilyChargeDeduction` — cap at 6 dependents
- `TestCalculateIR` — each bracket
- `TestRoundToNearestDirham`

**Full integration tests** — two complete worked examples from DOMAIN.md:

- `TestCalculate_WorkedExample1`: 10,000 MAD base, 6 years seniority, 2 dependents → net 9,514 MAD
- `TestCalculate_WorkedExample2`: 20,000 MAD base, 3 years seniority, 0 dependents → net 15,963 MAD

Every field in `PayrollResult` is asserted in the integration tests, making them effective regression guards when rates change.

#### 4. PayrollService Integration ✅

**Changes to `internal/application/payroll_service.go`:**

- Added `payrollCalculator` interface (consumer-defined, Go-idiomatic)
- Added `calculator payrollCalculator` field to `PayrollService`
- Updated `NewPayrollService` to accept a `payrollCalculator` as fifth argument
- Replaced `calculatePayrollStub` with `s.calculator.Calculate` in `GeneratePayrollResults`
- Removed the stub method entirely

**Changes to `internal/application/payroll_service_test.go`:**

- Added `mockPayrollCalculator` with a sensible default (returns a minimal valid result)
- Added `&mockPayrollCalculator{}` as fifth argument to all 24 `NewPayrollService` call sites
- Existing orchestration tests unchanged — they test workflow logic, not calculation correctness

### Key Design Decisions

**Calculator as adapter, not domain:** The calculator lives in `internal/adapter/payroll/`
because it is a concrete implementation of a rate-specific algorithm.
When 2027 rates are published, a new package can be added alongside without touching existing code.

**Interface defined by the consumer:** `payrollCalculator` is defined in `payroll_service.go`,
not in the calculator package. This keeps dependency arrows pointing inward
and makes the service trivially testable with a mock.

**No validation in calculator:** The calculator trusts its inputs are valid domain objects.
Validation happens in the domain layer (`emp.Validate()`, `pkg.Validate()`)
before the service calls the calculator.

**Stub removed entirely:** Rather than leaving dead code,
the stub was deleted once the real calculator was wired in.
The `mockPayrollCalculator` in tests serves the same purpose for orchestration testing.

### Lessons Learned

1. **Validate worked examples before coding** — catching the double-counted CNSS component early saved significant debugging time
2. **Per-step unit tests first** — made the integration test failures immediately traceable to the exact calculation step
3. **Test assertions expose documentation errors** — the rounding discrepancy in DOMAIN.md was found by a failing test, not a manual review
4. **Monthly professional expense evaluation** — using `gross × 12` as the annual proxy avoids needing year-to-date state in the calculator, keeping it stateless and easy to test

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
| 1D - App Services    | ✅ Complete | Weeks 5-6 | 100%       |
| 1E - Payroll Engine  | ✅ Complete | Weeks 7-8 | 100%       |
| 1F - TUI             | 📋 Next     | Week 9    | 0%         |
| 1G - Export & Polish | 📋 Planned  | Week 10   | 0%         |

**Overall Progress:** Phase 1E Complete (71%)
