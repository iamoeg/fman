# Implementation Guide

Phase-by-phase implementation guide for the financial management application.

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 1A - Foundation](#phase-1a---foundation-completed)
3. [Phase 1B - Domain Models & Repositories](#phase-1b---domain-models--repositories-current)
4. [Phase 1C - Payroll Engine](#phase-1c---payroll-engine)
5. [Phase 1D - Expense Tracking](#phase-1d---expense-tracking)
6. [Phase 1E - Export & Polish](#phase-1e---export--polish)
7. [Things to Consider](#things-to-consider)

---

## Overview

Development follows an iterative, phased approach:

- Start simple, prove concepts
- Build vertically (full stack for each feature)
- Test as you go
- Payroll is the priority (Phase 1 focus)

**Current Phase:** 1B - Domain Models & Repositories

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

## Phase 1B - Domain Models & Repositories 🚧 CURRENT

**Duration:** Week 2  
**Status:** 🚧 In Progress

### Goals

- Create domain models (pure business logic)
- Implement application services with inline interfaces
- Set up sqlc for type-safe SQL
- Implement SQLite repositories
- Write tests

### Tasks

#### 1. Money Type (`pkg/money/`) ✅ COMPLETED

**Priority:** Do this FIRST - everything else depends on it

**Status:** ✅ Complete with comprehensive testing

The Money type has been implemented with:

- **Overflow/underflow protection** on all arithmetic operations
- **Error handling** - returns `(Money, error)` for operations that can fail
- **Three error types**: `ErrOverflow`, `ErrDivByZero`, `ErrInvalidValue`
- **Comparison methods**: `Equals()`, `LessThan()`, `GreaterThan()`
- **String formatting**: implements `fmt.Stringer` for display
- **Full godoc documentation** with usage examples

**Implementation highlights:**

```go
package money

import (
    "errors"
    "fmt"
    "math"
)

var (
    ErrOverflow     = errors.New("money: operation would overflow")
    ErrDivByZero    = errors.New("money: division by zero")
    ErrInvalidValue = errors.New("money: invalid value (NaN or Inf)")
)

type Money struct {
    cents int64
}

// Constructors with validation
func FromCents(cents int64) Money
func FromMAD(mad float64) (Money, error)

// Accessors
func (m Money) Cents() int64
func (m Money) ToMAD() float64
func (m Money) String() string

// Arithmetic with overflow detection
func (m Money) Add(other Money) (Money, error)
func (m Money) Subtract(other Money) (Money, error)
func (m Money) Multiply(factor float64) (Money, error)
func (m Money) Divide(divisor float64) (Money, error)

// Comparison
func (m Money) Equals(other Money) bool
func (m Money) LessThan(other Money) bool
func (m Money) GreaterThan(other Money) bool
func (m Money) IsZero() bool
func (m Money) IsPositive() bool
func (m Money) IsNegative() bool
```

**Test coverage: 146 test cases** including:

- Constructor tests with edge cases (NaN, Inf, overflow)
- Arithmetic tests with overflow/underflow detection
- Comparison operation tests
- Floating-point precision verification
- Realistic payroll calculation simulation
- Benchmark tests for performance

**Usage example:**

```go
// Safe arithmetic with error handling
salary, err := money.FromMAD(8500.00)
if err != nil {
    return err
}

bonus, _ := money.FromMAD(500.00)
total, err := salary.Add(bonus)
if err != nil {
    return fmt.Errorf("calculating total: %w", err)
}

// Comparison
threshold, _ := money.FromMAD(10000.00)
if total.LessThan(threshold) {
    fmt.Println("Below tax threshold")
}

// Display
fmt.Println(total)  // "9000.00 MAD"
```

**Key design decisions:**

- Returns errors instead of panicking for production safety
- All overflow checks verified against int64 boundaries
- Tested with realistic Moroccan payroll scenarios (CNSS, AMO, IR calculations)

#### 2. Domain Models (`internal/domain/`)

Create pure business entities (no database tags, no external dependencies
except uuid/time):

**Organization:**

```go
type Organization struct {
    ID        uuid.UUID
    Name      string
    Address   string
    Activity  string
    LegalForm string
    ICENum    string
    IFNum     string
    RCNum     string
    CNSSNum   string
    BankRIB   string
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt *time.Time
}

func (o *Organization) Validate() error {
    if o.Name == "" {
        return errors.New("name is required")
    }
    // Add other validations
    return nil
}
```

**Employee:**

```go
type Employee struct {
    ID                    uuid.UUID
    OrgID                 uuid.UUID
    SerialNum             int
    FullName              string
    DisplayName           string
    Address               string
    EmailAddress          string
    PhoneNumber           string
    BirthDate             time.Time
    Gender                Gender
    MaritalStatus         MaritalStatus
    NumDependents         int
    NumKids               int
    CINNum                string
    CNSSNum               string
    HireDate              time.Time
    Position              string
    CompensationPackageID uuid.UUID
    BankRIB               string
    CreatedAt             time.Time
    UpdatedAt             time.Time
    DeletedAt             *time.Time
}

type Gender string
const (
    GenderMale   Gender = "MALE"
    GenderFemale Gender = "FEMALE"
)

type MaritalStatus string
const (
    MaritalStatusSingle    MaritalStatus = "SINGLE"
    MaritalStatusMarried   MaritalStatus = "MARRIED"
    MaritalStatusSeparated MaritalStatus = "SEPARATED"
    MaritalStatusDivorced  MaritalStatus = "DIVORCED"
    MaritalStatusWidowed   MaritalStatus = "WIDOWED"
)

func (e *Employee) Validate() error {
    if e.FullName == "" {
        return errors.New("full_name is required")
    }
    if e.CINNum == "" {
        return errors.New("cin_num is required")
    }
    // Add validations for CIN format, etc.
    return nil
}
```

**CompensationPackage:**

```go
type CompensationPackage struct {
    ID         uuid.UUID
    BaseSalary money.Money  // ✅ Use Money type!
    Currency   string
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  *time.Time
}
```

**PayrollPeriod, PayrollResult:**
Similarly define these with money.Money for all monetary fields

**Important:**
All monetary fields should use `money.Money`, not `float64` or `int64` directly.

#### 3. Application Services (`internal/application/`)

Services define small, focused interfaces and orchestrate domain logic:

```go
// organization_service.go
package application

type organizationRepository interface {
    Create(ctx context.Context, org *domain.Organization) error
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
    List(ctx context.Context) ([]*domain.Organization, error)
    Update(ctx context.Context, org *domain.Organization) error
    SoftDelete(ctx context.Context, id uuid.UUID) error
}

type OrganizationService struct {
    repo organizationRepository
}

func NewOrganizationService(repo organizationRepository) *OrganizationService {
    return &OrganizationService{repo: repo}
}

func (s *OrganizationService) CreateOrganization(
    ctx context.Context,
    org *domain.Organization,
) error {
    // Validate
    if err := org.Validate(); err != nil {
        return fmt.Errorf("invalid organization: %w", err)
    }

    // Set timestamps
    now := time.Now().UTC()
    org.CreatedAt = now
    org.UpdatedAt = now

    // Generate UUID
    org.ID = uuid.New()

    // Persist
    if err := s.repo.Create(ctx, org); err != nil {
        return fmt.Errorf("failed to create organization: %w", err)
    }

    return nil
}

func (s *OrganizationService) GetOrganization(
    ctx context.Context,
    id uuid.UUID,
) (*domain.Organization, error) {
    org, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get organization: %w", err)
    }
    return org, nil
}

// ... other methods
```

**Benefits:**

- Service only depends on methods it needs
- Easy to mock for testing
- Clear, focused interface

#### 4. Set Up sqlc

**Install:**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**Create `sqlc.yaml` in project root:**

```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "db/query/"
    schema: "db/migration/"
    gen:
      go:
        package: "sqldb"
        out: "internal/adapter/sqlite/sqldb/"
        emit_json_tags: true
        emit_interface: true
        emit_prepared_queries: false
```

**Create query files in `db/query/`:**

`organization.sql`:

```sql
-- name: GetOrganization :one
SELECT * FROM organization
WHERE id = ? AND deleted_at IS NULL
LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organization
WHERE deleted_at IS NULL
ORDER BY name;

-- name: CreateOrganization :exec
INSERT INTO organization (
    id, name, address, activity, legal_form,
    ice_num, if_num, rc_num, cnss_num, bank_rib,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateOrganization :exec
UPDATE organization
SET name = ?, address = ?, activity = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL;

-- name: SoftDeleteOrganization :exec
UPDATE organization
SET deleted_at = ?
WHERE id = ? AND deleted_at IS NULL;
```

**Generate code:**

```bash
sqlc generate
```

This creates type-safe functions in `internal/adapter/sqlite/sqldb/`

#### 5. Implement Repositories (`internal/adapter/sqlite/`)

Repositories implicitly satisfy application service interfaces:

```go
// organization_repo.go
package sqlite

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
    "yourproject/internal/adapter/sqlite/sqldb"
    "yourproject/internal/domain"
)

var (
    ErrNotFound = errors.New("record not found")
    ErrDuplicate = errors.New("duplicate record")
)

type OrganizationRepository struct {
    db      *sql.DB
    queries *sqldb.Queries
}

func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
    return &OrganizationRepository{
        db:      db,
        queries: sqldb.New(db),
    }
}

// Implicitly satisfies organizationRepository interface
func (r *OrganizationRepository) Create(
    ctx context.Context,
    org *domain.Organization,
) error {
    err := r.queries.CreateOrganization(ctx, sqldb.CreateOrganizationParams{
        ID:        org.ID.String(),
        Name:      org.Name,
        Address:   sqlNullString(org.Address),
        Activity:  sqlNullString(org.Activity),
        LegalForm: sqlNullString(org.LegalForm),
        IceNum:    sqlNullString(org.ICENum),
        IfNum:     sqlNullString(org.IFNum),
        RcNum:     sqlNullString(org.RCNum),
        CnssNum:   sqlNullString(org.CNSSNum),
        BankRib:   sqlNullString(org.BankRIB),
        CreatedAt: org.CreatedAt.Format(time.RFC3339),
        UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
    })

    if err != nil {
        // Check for constraint violations
        if isUniqueConstraintError(err) {
            return fmt.Errorf("%w: %v", ErrDuplicate, err)
        }
        return fmt.Errorf("failed to create organization: %w", err)
    }

    return nil
}

func (r *OrganizationRepository) FindByID(
    ctx context.Context,
    id uuid.UUID,
) (*domain.Organization, error) {
    row, err := r.queries.GetOrganization(ctx, id.String())
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get organization: %w", err)
    }

    return rowToOrganization(row)
}

func (r *OrganizationRepository) List(
    ctx context.Context,
) ([]*domain.Organization, error) {
    rows, err := r.queries.ListOrganizations(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list organizations: %w", err)
    }

    orgs := make([]*domain.Organization, 0, len(rows))
    for _, row := range rows {
        org, err := rowToOrganization(row)
        if err != nil {
            return nil, err
        }
        orgs = append(orgs, org)
    }

    return orgs, nil
}

// Helper: Convert sqlc row to domain model
func rowToOrganization(row sqldb.Organization) (*domain.Organization, error) {
    id, err := uuid.Parse(row.ID)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID: %w", err)
    }

    createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
    if err != nil {
        return nil, fmt.Errorf("invalid created_at: %w", err)
    }

    updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("invalid updated_at: %w", err)
    }

    var deletedAt *time.Time
    if row.DeletedAt.Valid {
        t, err := time.Parse(time.RFC3339, row.DeletedAt.String)
        if err == nil {
            deletedAt = &t
        }
    }

    return &domain.Organization{
        ID:        id,
        Name:      row.Name,
        Address:   row.Address.String,
        Activity:  row.Activity.String,
        LegalForm: row.LegalForm.String,
        ICENum:    row.IceNum.String,
        IFNum:     row.IfNum.String,
        RCNum:     row.RcNum.String,
        CNSSNum:   row.CnssNum.String,
        BankRIB:   row.BankRib.String,
        CreatedAt: createdAt,
        UpdatedAt: updatedAt,
        DeletedAt: deletedAt,
    }, nil
}

// Helper: Convert Go string to sql.NullString
func sqlNullString(s string) sql.NullString {
    return sql.NullString{
        String: s,
        Valid:  s != "",
    }
}

// Helper: Check if error is unique constraint violation
func isUniqueConstraintError(err error) bool {
    // SQLite specific - check error message
    return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
```

**Key points:**

- Convert between sqlc types and domain types
- Handle nullable columns with `sql.NullString`, etc.
- Parse UUIDs and timestamps
- **Convert Money**:
  Store as `money.Cents()` in DB, reconstruct with `money.FromCents()`
- Wrap errors with context
- Define sentinel errors (`ErrNotFound`, `ErrDuplicate`)

**Money type database conversion:**

```go
// Storing Money in database
params := sqldb.CreateCompensationPackageParams{
    BaseSalaryCents: baseSalary.Cents(),  // Convert Money to int64
    // ...
}

// Reading Money from database
baseSalary := money.FromCents(row.BaseSalaryCents)  // Convert int64 to Money
```

#### 6. Testing

**Repository tests with `:memory:`:**

```go
// organization_repo_test.go
package sqlite_test

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/pressly/goose/v3"
    "github.com/stretchr/testify/require"

    "yourproject/internal/adapter/sqlite"
    "yourproject/internal/domain"
)

func TestOrganizationRepository_Create(t *testing.T) {
    // Create in-memory database
    db, err := sql.Open("sqlite", ":memory:")
    require.NoError(t, err)
    defer db.Close()

    // Enable foreign keys
    _, err = db.Exec("PRAGMA foreign_keys = ON")
    require.NoError(t, err)

    // Run migrations
    err = goose.Up(db, "../../../db/migration")
    require.NoError(t, err)

    // Create repository
    repo := sqlite.NewOrganizationRepository(db)

    // Test
    org := &domain.Organization{
        ID:        uuid.New(),
        Name:      "Test Company",
        Address:   "123 Test St",
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
    }

    err = repo.Create(context.Background(), org)
    require.NoError(t, err)

    // Verify
    found, err := repo.FindByID(context.Background(), org.ID)
    require.NoError(t, err)
    require.Equal(t, org.Name, found.Name)
    require.Equal(t, org.Address, found.Address)
}

func TestOrganizationRepository_FindByID_NotFound(t *testing.T) {
    db, _ := sql.Open("sqlite", ":memory:")
    defer db.Close()

    db.Exec("PRAGMA foreign_keys = ON")
    goose.Up(db, "../../../db/migration")

    repo := sqlite.NewOrganizationRepository(db)

    _, err := repo.FindByID(context.Background(), uuid.New())
    require.ErrorIs(t, err, sqlite.ErrNotFound)
}
```

**Service tests with mocks:**

```go
// organization_service_test.go
package application_test

import (
    "context"
    "errors"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/require"

    "yourproject/internal/application"
    "yourproject/internal/domain"
)

type mockOrgRepo struct {
    createFunc func(context.Context, *domain.Organization) error
    findByIDFunc func(context.Context, uuid.UUID) (*domain.Organization, error)
}

func (m *mockOrgRepo) Create(ctx context.Context, org *domain.Organization) error {
    if m.createFunc != nil {
        return m.createFunc(ctx, org)
    }
    return nil
}

func (m *mockOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
    if m.findByIDFunc != nil {
        return m.findByIDFunc(ctx, id)
    }
    return nil, nil
}

func (m *mockOrgRepo) List(ctx context.Context) ([]*domain.Organization, error) {
    return nil, nil
}

func (m *mockOrgRepo) Update(ctx context.Context, org *domain.Organization) error {
    return nil
}

func (m *mockOrgRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
    return nil
}

func TestOrganizationService_CreateOrganization(t *testing.T) {
    var capturedOrg *domain.Organization

    mock := &mockOrgRepo{
        createFunc: func(ctx context.Context, org *domain.Organization) error {
            capturedOrg = org
            return nil
        },
    }

    service := application.NewOrganizationService(mock)

    org := &domain.Organization{
        Name: "Test Company",
    }

    err := service.CreateOrganization(context.Background(), org)
    require.NoError(t, err)

    // Verify service set ID and timestamps
    require.NotEqual(t, uuid.Nil, capturedOrg.ID)
    require.False(t, capturedOrg.CreatedAt.IsZero())
    require.False(t, capturedOrg.UpdatedAt.IsZero())
}

func TestOrganizationService_CreateOrganization_ValidationError(t *testing.T) {
    mock := &mockOrgRepo{}
    service := application.NewOrganizationService(mock)

    org := &domain.Organization{
        Name: "", // Invalid
    }

    err := service.CreateOrganization(context.Background(), org)
    require.Error(t, err)
    require.Contains(t, err.Error(), "name")
}
```

### Deliverables

By end of Phase 1B:

- [x] Money type implemented and tested
  - Full implementation with overflow detection
  - 146 comprehensive test cases
  - Benchmark tests
  - Verification scripts
- [ ] Domain models for Organization, Employee, CompensationPackage
- [ ] Application services with inline interfaces
- [ ] sqlc configured and generating code
- [ ] SQLite repositories for Organization, Employee
- [ ] Repository tests using `:memory:`
- [ ] Service tests using mocks
- [ ] Configuration system (XDG paths)

---

## Phase 1C - Payroll Engine

**Duration:** Weeks 3-4  
**Status:** ⏳ Not Started

### Goals

- Implement Moroccan payroll calculator
- Generate monthly payroll periods
- Create payslip PDF generation
- Build payroll TUI screens

### High-Level Tasks

1. Implement Moroccan payroll calculator (`internal/adapter/payroll/morocco/`)
2. Implement PayrollService
3. Implement PDF generation adapter
4. Build TUI screens for payroll workflow
5. Comprehensive testing of calculations

**Details TBD when Phase 1B is complete**

---

## Phase 1D - Expense Tracking

**Duration:** Week 5  
**Status:** ⏳ Not Started

**Note:** Currently deferred - payroll is priority

---

## Phase 1E - Export & Polish

**Duration:** Week 6  
**Status:** ⏳ Not Started

**Goals:**

- JSON/XML export
- Data backup/restore
- Error handling polish
- Documentation

---

## Things to Consider

### Error Handling

- Define sentinel errors early (`ErrNotFound`, `ErrDuplicate`)
- Use `errors.Is()` and `errors.As()` for checking
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- TUI will translate technical errors to user-friendly messages
- **Money operations return errors** - always check them in business logic

### UUID Generation

- Always generate in application code (service layer)
- Never in database
- Use `uuid.New()` from `google/uuid`

### Timestamp Handling

- Always store UTC: `time.Now().UTC()`
- Format as RFC3339: `time.Format(time.RFC3339)`
- Parse when reading: `time.Parse(time.RFC3339, s)`

### Money Type Usage

- **Always use `money.Money` for monetary values** in domain models
- Store in database as `int64` (cents) using `money.Cents()`
- Reconstruct from database using `money.FromCents()`
- Handle errors from arithmetic operations:

```go
total, err := salary.Add(bonus)
if err != nil {
    return fmt.Errorf("calculating total: %w", err)
}
```

- Use comparison methods for business logic:

```go
if salary.LessThan(threshold) {
    // Apply different tax rate
}
```

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

### Payroll Immutability

**Decision:** Once finalized, payroll results cannot be modified

**Implementation:**

- Status field: DRAFT → FINALIZED
- No UPDATE operations on finalized records
- If error found: DELETE entire period and regenerate

**Rationale:**

- Ensures data integrity
- Matches legal/accounting practices
- Simplifies audit trail
