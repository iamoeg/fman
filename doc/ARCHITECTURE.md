# System Architecture

This document describes the architecture of `finmgmt`,
including structural patterns, design decisions, and implementation guidelines.

---

## Table of Contents

1. [Architectural Pattern](#architectural-pattern)
2. [Directory Structure](#directory-structure)
3. [Layer Responsibilities](#layer-responsibilities)
4. [Key Design Decisions](#key-design-decisions)
5. [Technical Guidelines](#technical-guidelines)
6. [Testing Strategy](#testing-strategy)

---

## Architectural Pattern

### Hexagonal Architecture (Go-Idiomatic)

We follow **Hexagonal Architecture** principles adapted for Go idioms,
not a direct translation of Java patterns.

**Core Principles:**

1. **Domain Layer:** Pure business logic, no external dependencies
2. **Application Layer:** Services that orchestrate domain objects
   and define their own interfaces
3. **Adapter Layer:** Concrete implementations of databases,
   external services, etc.

**Key Difference from Traditional Hexagonal:**

Instead of defining interfaces in a separate `ports/` package (Java style),
we follow **Go idioms**:

- Services define small, focused interfaces for what they need
- Adapters implicitly satisfy these interfaces
- "Accept interfaces, return structs"

### Example

```go
// application/payroll_service.go
type employeeRepository interface {
    FindByOrgID(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error)
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error)
}

type PayrollService struct {
    employees employeeRepository  // Small, focused interface
    calculator PayrollCalculator
}

// adapter/sqlite/employee_repo.go
type EmployeeRepository struct {
    db      *sql.DB
    queries *sqldb.Queries
}

func (r *EmployeeRepository) FindByID(...) (*domain.Employee, error) {
    // Implementation - implicitly satisfies employeeRepository interface
}
```

**Benefits:**

- Business logic is completely independent of UI/database
- Easy to test (mock only what you use)
- Can swap TUI for API without changing business logic
- Idiomatic Go - small interfaces defined by consumers
- Clear separation of concerns

**The dependency arrows point inward:** Adapters → Application → Domain

---

## Directory Structure

```text
finmgmt/
├── cmd/
│   ├── tui/              # TUI application entry point
│   └── api/              # Future: API server
├── db/
│   ├── migration/        # Database migrations (goose)
│   └── query/            # SQL queries for sqlc
├── internal/
│   ├── domain/           # Pure business entities (no dependencies)
│   ├── application/      # Services (define their own interfaces inline)
│   └── adapter/          # Concrete implementations
│       ├── sqlite/       # SQLite repository implementations
│       │   └── sqldb/    # sqlc-generated code
│       ├── payroll/      # Moroccan calculation engine
│       └── export/       # JSON/XML exporters
├── pkg/
│   ├── money/            # Money type (avoid float precision issues)
│   └── util/             # Shared utilities (enum helpers, etc.)
└── ui/
    └── tui/              # TUI-specific code
```

### Why This Structure?

**`db/` for SQL, `internal/` for Go:**

- SQL files (migrations, queries) aren't Go packages
- Easier to find and maintain
- Cleaner tool configuration (goose, sqlc)

**`adapter` not `infrastructure`:**

- Avoids confusion with DevOps infrastructure
- Correct hexagonal architecture terminology
- Clear intent: adapts external technologies to our interfaces

**No `ports/` package:**

- Go idiom: consumers define interfaces
- Services define small, focused interfaces for what they need
- Less boilerplate, easier to test
- Still maintains clean architecture (dependency direction is what matters)

**`pkg/` for reusable utilities:**

- Money type is used across all layers
- Utility functions (enum helpers) are shared
- Can be imported by other projects if needed

**Singular names:**

- `migration` not `migrations`
- `adapter` not `adapters`
- `model` not `models`
- Consistent convention

---

## Layer Responsibilities

### Domain Layer (`internal/domain/`)

**Purpose:** Core business entities and rules

**Allowed:**

- ✅ Pure Go structs
- ✅ Validation methods
- ✅ Business logic
- ✅ Enums (with validation)
- ✅ Helper methods (e.g., TotalDueToCNSS)
- ✅ Dependencies: uuid, time, pkg/money

**Not Allowed:**

- ❌ Database tags
- ❌ External dependencies (except uuid, time, pkg/money)
- ❌ Framework code
- ❌ I/O operations

**Example:**

```go
type Employee struct {
    ID             uuid.UUID
    OrgID          uuid.UUID
    FullName       string
    CINNum         string
    Gender         GenderEnum
    MaritalStatus  MaritalStatusEnum
    CompensationID uuid.UUID
    // ...
}

func (e *Employee) Validate() error {
    if e.FullName == "" {
        return errors.New("full_name is required")
    }
    if !e.Gender.IsSupported() {
        return fmt.Errorf("gender not supported: must be one of %v", SupportedGendersStr)
    }
    return nil
}

type GenderEnum string

const (
    GenderMale   GenderEnum = "MALE"
    GenderFemale GenderEnum = "FEMALE"
)

func (g GenderEnum) IsSupported() bool {
    _, ok := supportedGenders[g]
    return ok
}
```

### Application Layer (`internal/application/`)

**Purpose:** Orchestrate business logic, define service interfaces

**Responsibilities:**

- Define small, focused interfaces for dependencies
- Coordinate domain objects
- Implement use cases
- Set timestamps and generate UUIDs
- Transaction boundaries
- Domain validation before persistence

**Pattern:**

```go
// Define interface inline
type employeeRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error)
    FindByOrgID(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error)
    Create(ctx context.Context, emp *domain.Employee) error
}

type EmployeeService struct {
    employees employeeRepository
}

func (s *EmployeeService) CreateEmployee(
    ctx context.Context,
    emp *domain.Employee,
) error {
    // Generate UUID
    emp.ID = uuid.New()

    // Set timestamps
    now := time.Now().UTC()
    emp.CreatedAt = now
    emp.UpdatedAt = now

    // Validate
    if err := emp.Validate(); err != nil {
        return fmt.Errorf("invalid employee: %w", err)
    }

    // Persist
    if err := s.employees.Create(ctx, emp); err != nil {
        return fmt.Errorf("failed to create employee: %w", err)
    }

    return nil
}
```

### Adapter Layer (`internal/adapter/`)

**Purpose:** Implement technical details

**Responsibilities:**

- Implement repository interfaces (implicitly)
- Database operations
- External service integrations
- File I/O
- PDF generation

**Key Point:** Adapters don't declare "implements"
-- they just provide the methods that satisfy the interfaces
defined in application services.

**Example:**

```go
// adapter/sqlite/employee_repo.go
type EmployeeRepository struct {
    db      *sql.DB
    queries *sqldb.Queries
}

// This method implicitly satisfies the employeeRepository interface
// defined in application/employee_service.go
func (r *EmployeeRepository) FindByID(
    ctx context.Context,
    id uuid.UUID,
) (*domain.Employee, error) {
    row, err := r.queries.GetEmployee(ctx, id.String())
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get employee: %w", err)
    }

    // Convert sqlc type to domain type
    return rowToEmployee(row)
}

// Helper: Convert sqlc row to domain model
func rowToEmployee(row sqldb.Employee) (*domain.Employee, error) {
    id, err := uuid.Parse(row.ID)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID: %w", err)
    }

    birthDate, err := time.Parse(time.RFC3339, row.BirthDate)
    if err != nil {
        return nil, fmt.Errorf("invalid birth_date: %w", err)
    }

    return &domain.Employee{
        ID:        id,
        FullName:  row.FullName,
        BirthDate: birthDate,
        // ... convert all fields
    }, nil
}
```

### UI Layer (`ui/tui/`)

**Purpose:** User interface

**Responsibilities:**

- Display data
- Handle user input
- Call application services
- Manage UI state

**Not Allowed:**

- ❌ Business logic
- ❌ Direct repository access
- ❌ Database operations

---

## Key Design Decisions

### 1. Money as Integer Cents with Error Handling

**Decision:** Store all monetary values as integers (cents/smallest unit)
and return errors from operations

**Rationale:**

- Floating-point arithmetic is imprecise for financial calculations
- Example: `0.1 + 0.2 = 0.30000000000000004` in float64
- Integers guarantee exact calculations
- Operations can fail (overflow, division by zero, NaN/Inf)
- Explicit error handling prevents silent failures

**Implementation:**

```go
// pkg/money/money.go
type Money struct {
    cents int64
}

func FromCents(cents int64) Money {
    return Money{cents: cents}
}

func FromMAD(mad float64) (Money, error) {
    if math.IsNaN(mad) || math.IsInf(mad, 0) {
        return Money{}, ErrInvalidValue
    }

    madCents := mad * 100
    if madCents > float64(math.MaxInt64) || madCents < float64(math.MinInt64) {
        return Money{}, fmt.Errorf("%w: %f MAD is too large", ErrOverflow, mad)
    }

    return Money{cents: int64(math.Round(madCents))}, nil
}

func (m Money) Add(other Money) (Money, error) {
    // Check for overflow
    if other.cents > 0 && m.cents > math.MaxInt64-other.cents {
        return Money{}, fmt.Errorf("%w: %v + %v", ErrOverflow, m, other)
    }
    if other.cents < 0 && m.cents < math.MinInt64-other.cents {
        return Money{}, fmt.Errorf("%w: %v + %v", ErrOverflow, m, other)
    }

    return Money{cents: m.cents + other.cents}, nil
}

func (m Money) Divide(divisor float64) (Money, error) {
    if divisor == 0.0 {
        return Money{}, ErrDivByZero
    }
    if math.IsNaN(divisor) || math.IsInf(divisor, 0) {
        return Money{}, ErrInvalidValue
    }

    result := float64(m.cents) / divisor
    if result > float64(math.MaxInt64) || result < float64(math.MinInt64) {
        return Money{}, fmt.Errorf("%w: %v / %f", ErrOverflow, m, divisor)
    }

    return Money{cents: int64(math.Round(result))}, nil
}
```

**Benefits:**

- Exact precision for all financial calculations
- Protection against arithmetic errors
- Clear error handling
- Type safety

### 2. Calculated Fields in Payroll

**Decision:** Store all calculated values in `payroll_result` table,
not just base values

**Rationale:**

- Payroll is a legal document requiring historical accuracy
- Tax laws and calculation logic change over time
- Need to prove what was actually paid, not what would be calculated today
- Performance: Reports don't need to recalculate
- Compliance: Immutable audit trail

**Trade-off:** Some data redundancy, but financial/legal requirements justify it

### 3. Comprehensive Domain Validation

**Decision:** Validate all business rules in domain layer with detailed error messages

**Implementation:**

Every domain entity has:

- Main `Validate()` method that calls individual validators
- Individual `ValidateX()` methods for each field/rule
- Sentinel errors for each validation failure
- Clear, descriptive error messages

**Example:**

```go
func (e *Employee) Validate() error {
    if err := e.ValidateID(); err != nil {
        return err
    }
    if err := e.ValidateFullName(); err != nil {
        return err
    }
    if err := e.ValidateBirthDate(); err != nil {
        return err
    }
    // ... all validations
    return nil
}

func (e *Employee) ValidateBirthDate() error {
    now := time.Now().UTC()
    minBirthDate := now.AddDate(-MaxWorkLegalAge, 0, 0)
    maxBirthDate := now.AddDate(-MinWorkLegalAge, 0, 0)
    if e.BirthDate.Before(minBirthDate) || e.BirthDate.After(maxBirthDate) {
        return fmt.Errorf(
            "%w: employee's age must be between %v and %v years",
            ErrInvalidEmployeeBirthDate,
            MinWorkLegalAge,
            MaxWorkLegalAge,
        )
    }
    return nil
}
```

**Benefits:**

- Catch errors early (before database)
- Easy to test (no dependencies)
- Clear error messages for debugging
- Business rules documented in code

### 4. CNSS and AMO Separation

**Decision:** Keep CNSS and AMO contributions separate in calculations,
provide helpers for combined totals

**Rationale:**

- CNSS (social security) and AMO (health insurance) are legally distinct
- In practice, AMO is collected by CNSS
- Separation allows for:
  - Accurate reporting
  - Future changes in collection
  - Clear audit trail

**Implementation:**

```go
type PayrollResult struct {
    // CNSS (excludes AMO)
    TotalCNSSEmployeeContrib money.Money
    TotalCNSSEmployerContrib money.Money

    // AMO (separate)
    AMOEmployeeContrib money.Money
    AMOEmployerContrib money.Money
}

// Helper: Total actually paid to CNSS (includes AMO)
func (pr *PayrollResult) TotalDueToCNSS() (money.Money, error) {
    total, _ := pr.TotalCNSSEmployeeContrib.Add(pr.TotalCNSSEmployerContrib)
    total, _ = total.Add(pr.AMOEmployeeContrib)
    total, _ = total.Add(pr.AMOEmployerContrib)
    return total, nil
}
```

### 5. Soft Deletes

**Decision:** Never hard-delete financial data; use `deleted_at` timestamps

**Rationale:**

- Financial data must be retained for legal/audit purposes
- Ability to "un-delete" if mistake
- Maintains referential integrity
- Queries filter on `deleted_at IS NULL`

### 6. Foreign Key Strategies

**CASCADE** - Delete children when parent deleted:

- `employee.org_id` → organization
- `payroll_period.org_id` → organization
- `payroll_result.payroll_period_id` → payroll_period
- `payroll_result.employee_id` → employee

**RESTRICT** - Cannot delete if children exist:

- `employee.compensation_package_id` → employee_compensation_package
- `payroll_result.compensation_package_id` → employee_compensation_package

**Rationale:** Compensation packages are historical artifacts.
Once referenced by payroll, they're part of the permanent record.

### 7. Payroll Immutability

**Decision:** Once finalized, payroll results cannot be modified

**Implementation:**

- Status field: DRAFT → FINALIZED
- No UPDATE operations on finalized records
- If error found: DELETE entire period and regenerate

**Rationale:**

- Ensures data integrity
- Matches legal/accounting practices
- Simplifies audit trail

### 8. sqlc Over ORMs

**Decision:** Use sqlc for database access, not GORM or other ORMs

**Rationale:**

- **Full SQL control:** Write exact queries needed
- **Type safety:** Compile-time checks, no runtime surprises
- **No magic:** Generated code is readable and debuggable
- **Performance:** No hidden N+1 queries or lazy loading issues

**Trade-off:** More initial setup, but better long-term maintainability

### 9. Employee Serial Numbers

**Decision:** Generate in application code, not database auto-increment

**Rationale:**

- SQLite auto-increment is global, not per-organization
- Need per-organization numbering (Employee #1 in each org)
- Logic: `SELECT MAX(serial_num) FROM employee WHERE org_id = ? + 1`
- Database unique constraint catches race conditions

### 10. Enum Pattern with String Helper

**Decision:** Use map-based enums with pre-computed string representations

**Implementation:**

```go
type GenderEnum string

const (
    GenderMale   GenderEnum = "MALE"
    GenderFemale GenderEnum = "FEMALE"
)

var supportedGenders = map[GenderEnum]struct{}{
    GenderMale:   {},
    GenderFemale: {},
}

// Pre-computed string for error messages
var SupportedGendersStr = util.EnumMapToString(supportedGenders)

func (g GenderEnum) IsSupported() bool {
    _, ok := supportedGenders[g]
    return ok
}
```

**Benefits:**

- O(1) validation
- Clean error messages
- Easy to extend
- Type-safe

---

## Technical Guidelines

### SQLite Specifics

**Foreign Keys Must Be Enabled:**

```go
db, err := sql.Open("sqlite", dbPath)
_, err = db.Exec("PRAGMA foreign_keys = ON")  // CRITICAL
```

**No Native UUID or Timestamp Types:**

- UUIDs: Store as TEXT, convert with `uuid.String()` / `uuid.Parse()`
- Timestamps: Store as TEXT in RFC3339,
  convert with `time.Format()` / `time.Parse()`

**`:memory:` for Tests:**

```go
// In-memory database - perfect for fast, isolated tests
db, err := sql.Open("sqlite", ":memory:")
```

### sqlc Workflow

**Write SQL queries with special comments:**

```sql
-- name: GetOrganization :one
SELECT * FROM organization WHERE id = ? AND deleted_at IS NULL LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organization WHERE deleted_at IS NULL ORDER BY name;

-- name: CreateOrganization :exec
INSERT INTO organization (id, name, legal_form, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);
```

**Generate type-safe Go code:**

```bash
sqlc generate
```

**Use in repositories:**

```go
func (r *OrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
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

### Error Handling

**Define sentinel errors:**

```go
var (
    ErrNotFound  = errors.New("record not found")
    ErrDuplicate = errors.New("duplicate record")
)
```

**Wrap with context:**

```go
if err != nil {
    return fmt.Errorf("failed to create organization: %w", err)
}
```

**Check with `errors.Is()`:**

```go
if errors.Is(err, ErrNotFound) {
    // Handle not found
}
```

### Configuration

**XDG Base Directory Compliance:**

- Config: `~/.config/finmgmt/config.yaml`
- Data: `~/.local/share/finmgmt/data.db`

---

## Testing Strategy

### Domain Tests

Test validation and business rules without any infrastructure:

```go
func TestEmployee_Validate(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        emp     *domain.Employee
        wantErr error
    }{
        {
            name: "valid employee",
            emp: &domain.Employee{
                ID:       uuid.New(),
                FullName: "Ahmed Ali",
                // ... all required fields
            },
            wantErr: nil,
        },
        {
            name: "empty full name",
            emp: &domain.Employee{
                FullName: "",
            },
            wantErr: domain.ErrEmployeeFullNameRequired,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            err := tt.emp.Validate()

            if tt.wantErr == nil {
                require.NoError(t, err)
            } else {
                require.ErrorIs(t, err, tt.wantErr)
            }
        })
    }
}
```

### Repository Tests

Use `:memory:` SQLite for fast, isolated tests:

```go
func TestEmployeeRepository(t *testing.T) {
    db, _ := sql.Open("sqlite", ":memory:")
    defer db.Close()

    // Enable foreign keys
    db.Exec("PRAGMA foreign_keys = ON")

    // Run migrations
    goose.Up(db, "../../../../db/migration")

    // Test
    repo := NewEmployeeRepository(db)
    emp := &domain.Employee{...}
    err := repo.Create(context.Background(), emp)
    require.NoError(t, err)
}
```

### Service Tests

Mock only what you need:

```go
type mockEmployeeRepo struct {
    findByOrgIDFunc func(context.Context, uuid.UUID) ([]*domain.Employee, error)
}

func (m *mockEmployeeRepo) FindByOrgID(ctx context.Context, id uuid.UUID) ([]*domain.Employee, error) {
    return m.findByOrgIDFunc(ctx, id)
}

func TestPayrollService_GeneratePayroll(t *testing.T) {
    mock := &mockEmployeeRepo{
        findByOrgIDFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.Employee, error) {
            return []*domain.Employee{{...}}, nil
        },
    }

    service := NewPayrollService(mock)
    // Test service logic
}
```

### Test Patterns

**Table-Driven Tests:**

- All domain validation tests use table-driven pattern
- Easy to add new test cases
- Clear test names

**Parallel Execution:**

- All tests use `t.Parallel()`
- Fast test suite execution

**Realistic Data:**

- Use Moroccan names, amounts, dates
- Makes tests more meaningful

---

## Lessons Learned

### Architecture

1. **Go idioms > Cargo-culting Java patterns** -
   No separate `ports/` package needed
2. **Dependency direction matters, not directory structure** -
   Inward toward domain is what counts
3. **Small, focused interfaces** -
   Services define only what they need
4. **db/ for SQL, internal/ for Go** -
   Don't force SQL into Go package structure

### Domain Design

1. **Money type is fundamental** - Build it first, everything depends on it
2. **Validation in domain is powerful** - Catches errors early, easy to test
3. **Cross-field validation needs care** - BirthDate vs HireDate, Status vs FinalizedAt
4. **Enums with helpers improve UX** - Pre-computed strings for error messages
5. **Helper methods on entities** - TotalDueToCNSS() makes business logic clearer

### Database

1. **Calculated fields in payroll ARE correct** -
   Historical accuracy trumps normalization
2. **ON DELETE RESTRICT for historical artifacts** -
   Preserve audit trail
3. **Soft deletes everywhere** -
   Financial data shouldn't disappear
4. **PRAGMA foreign_keys = ON** -
   Must enable on every SQLite connection

### Go Practices

1. **Money as integer cents** - Never float64 for financial calculations
2. **Error returns from Money operations** - Prevents silent failures
3. **Use context.Context everywhere** - Enables timeouts and cancellation
4. **sqlc over ORMs** - Type safety without magic
5. **`:memory:` for tests** - Fast, isolated, auto-cleanup
6. **Table-driven tests with t.Parallel()** - Best practice for Go testing

### Testing

1. **Test domain first** - Pure logic, no dependencies, easy to test
2. **Comprehensive test coverage builds confidence** - 200+ test scenarios
3. **Realistic test data matters** - Moroccan context makes tests meaningful
4. **Benchmarks are valuable** - Know your performance characteristics
