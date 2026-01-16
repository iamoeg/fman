# Implementation Guide

Phase-by-phase implementation guide for the financial management application.

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 1A - Foundation](#phase-1a---foundation-completed)
3. [Phase 1B - Domain Models & Tests](#phase-1b---domain-models--tests-completed)
4. [Phase 1C - Repositories & Application Services](#phase-1c---repositories--application-services-next)
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

## Phase 1C - Repositories & Application Services ⏳ NEXT

**Duration:** Weeks 3-4  
**Status:** 📋 Planned

### Goals

- Set up sqlc for type-safe SQL
- Implement SQLite repositories
- Create application services with inline interfaces
- Write repository and service tests

### Tasks

#### 1. Set Up sqlc

**Install sqlc:**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**Create `sqlc.yaml`:**

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

- `organization.sql` - CRUD operations for organizations
- `employee.sql` - CRUD operations for employees
- `compensation_package.sql` - CRUD operations for compensation packages
- `payroll_period.sql` - CRUD operations for payroll periods
- `payroll_result.sql` - CRUD operations for payroll results

**Generate code:**

```bash
sqlc generate
```

#### 2. Implement Repositories (`internal/adapter/sqlite/`)

**Files to create:**

- `organization_repo.go` - Organization repository
- `employee_repo.go` - Employee repository
- `compensation_package_repo.go` - Compensation package repository
- `payroll_period_repo.go` - Payroll period repository
- `payroll_result_repo.go` - Payroll result repository

**Each repository should:**

- Use sqlc-generated code
- Convert between sqlc types and domain types
- Handle UUIDs and timestamps (SQLite stores as TEXT)
- Handle nullable fields with `sql.NullString`, etc.
- Define sentinel errors (ErrNotFound, ErrDuplicate)
- Wrap errors with context

**Key Patterns:**

```go
type OrganizationRepository struct {
    db      *sql.DB
    queries *sqldb.Queries
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
    err := r.queries.CreateOrganization(ctx, sqldb.CreateOrganizationParams{
        ID:        org.ID.String(),
        Name:      org.Name,
        // ... convert all fields
        CreatedAt: org.CreatedAt.Format(time.RFC3339),
        UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
    })

    if err != nil {
        if isUniqueConstraintError(err) {
            return fmt.Errorf("%w: %v", ErrDuplicate, err)
        }
        return fmt.Errorf("failed to create organization: %w", err)
    }

    return nil
}

func (r *OrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
    row, err := r.queries.GetOrganization(ctx, id.String())
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get organization: %w", err)
    }

    return rowToOrganization(row)
}
```

#### 3. Implement Application Services (`internal/application/`)

**Files to create:**

- `organization_service.go` - Organization management
- `employee_service.go` - Employee management
- `payroll_service.go` - Payroll generation and management

**Each service should:**

- Define small, focused interfaces for dependencies
- Orchestrate domain objects
- Set timestamps (CreatedAt, UpdatedAt)
- Generate UUIDs
- Handle transaction boundaries
- Validate domain objects before persistence

**Key Pattern:**

```go
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

func (s *OrganizationService) CreateOrganization(ctx context.Context, org *domain.Organization) error {
    // Set timestamps
    now := time.Now().UTC()
    org.CreatedAt = now
    org.UpdatedAt = now

    // Generate UUID
    org.ID = uuid.New()

    // Validate
    if err := org.Validate(); err != nil {
        return fmt.Errorf("invalid organization: %w", err)
    }

    // Persist
    if err := s.repo.Create(ctx, org); err != nil {
        return fmt.Errorf("failed to create organization: %w", err)
    }

    return nil
}
```

#### 4. Write Tests

**Repository Tests (using `:memory:`):**

```go
func TestOrganizationRepository_Create(t *testing.T) {
    db, _ := sql.Open("sqlite", ":memory:")
    defer db.Close()

    db.Exec("PRAGMA foreign_keys = ON")
    goose.Up(db, "../../../db/migration")

    repo := NewOrganizationRepository(db)

    org := &domain.Organization{
        ID:        uuid.New(),
        Name:      "Test Company",
        LegalForm: domain.LegalFormSARL,
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
    }

    err := repo.Create(context.Background(), org)
    require.NoError(t, err)

    found, err := repo.FindByID(context.Background(), org.ID)
    require.NoError(t, err)
    require.Equal(t, org.Name, found.Name)
}
```

**Service Tests (using mocks):**

```go
type mockOrgRepo struct {
    createFunc func(context.Context, *domain.Organization) error
}

func (m *mockOrgRepo) Create(ctx context.Context, org *domain.Organization) error {
    if m.createFunc != nil {
        return m.createFunc(ctx, org)
    }
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

    service := NewOrganizationService(mock)

    org := &domain.Organization{Name: "Test Company", LegalForm: domain.LegalFormSARL}
    err := service.CreateOrganization(context.Background(), org)

    require.NoError(t, err)
    require.NotEqual(t, uuid.Nil, capturedOrg.ID)
    require.False(t, capturedOrg.CreatedAt.IsZero())
}
```

### Deliverables

By end of Phase 1C:

- [ ] sqlc configured and generating code
- [ ] SQL query files for all entities
- [ ] SQLite repositories for all entities
- [ ] Application services for Organization and Employee
- [ ] Repository tests using `:memory:`
- [ ] Service tests using mocks
- [ ] Configuration system (XDG paths)

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
