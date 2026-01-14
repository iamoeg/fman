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
│   └── money/            # Money type (avoid float precision issues)
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
- ✅ Value objects (e.g., Money)

**Not Allowed:**

- ❌ Database tags
- ❌ External dependencies (except uuid, time)
- ❌ Framework code
- ❌ I/O operations

**Example:**

```go
type Employee struct {
    ID             uuid.UUID
    OrgID          uuid.UUID
    FullName       string
    CIN            string
    CompensationID uuid.UUID
    // ...
}

func (e *Employee) Validate() error {
    if e.FullName == "" {
        return errors.New("full_name is required")
    }
    if !isValidCIN(e.CIN) {
        return errors.New("invalid CIN format")
    }
    return nil
}
```

### Application Layer (`internal/application/`)

**Purpose:** Orchestrate business logic, define service interfaces

**Responsibilities:**

- Define small, focused interfaces for dependencies
- Coordinate domain objects
- Implement use cases
- Transaction boundaries

**Pattern:**

```go
// Define interface inline
type employeeRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error)
    FindByOrgID(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error)
}

type PayrollService struct {
    employees employeeRepository
}

func (s *PayrollService) GenerateMonthlyPayroll(
    ctx context.Context,
    orgID uuid.UUID,
    year int,
    month int,
) error {
    // Orchestrate domain objects
    employees, err := s.employees.FindByOrgID(ctx, orgID)
    // ... business logic
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
// defined in application/payroll_service.go
func (r *EmployeeRepository) FindByID(
    ctx context.Context,
    id uuid.UUID,
) (*domain.Employee, error) {
    row, err := r.queries.GetEmployee(ctx, id.String())
    // ... implementation
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

### 1. Money as Integer Cents

**Decision:** Store all monetary values as integers (cents/smallest unit)

**Rationale:**

- Floating-point arithmetic is imprecise for financial calculations
- Example: `0.1 + 0.2 = 0.30000000000000004` in float64
- Integers guarantee exact calculations

**Implementation:**

```go
// pkg/money/money.go
type Money struct {
    cents int64
}

func FromMAD(mad float64) (Money, error) {
    // Validates input and checks for overflow
    return Money{cents: int64(math.Round(mad * 100))}, nil
}

func (m Money) Add(other Money) (Money, error) {
    // Checks for overflow before adding
    return Money{cents: m.cents + other.cents}, nil
}
```

**Key Features:**

- **Overflow protection**: All arithmetic operations detect int64 overflow/underflow
- **Error handling**: Operations return `(Money, error)` for safety
- **Comparison methods**: `Equals()`, `LessThan()`, `GreaterThan()`
- **Display formatting**: `String()` method for user-friendly output
- **Comprehensive testing**: 146 test cases including edge cases and realistic payroll scenarios

**Status:** ✅ Fully implemented and tested

**Usage:**

```go
salary, err := money.FromMAD(8500.00)
if err != nil {
    return err
}

bonus, _ := money.FromMAD(500.00)
total, err := salary.Add(bonus)
if err != nil {
    return fmt.Errorf("calculating total: %w", err)
}

fmt.Println(total)  // "9000.00 MAD"
```

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

### 3. Soft Deletes

**Decision:** Never hard-delete financial data; use `deleted_at` timestamps

**Rationale:**

- Financial data must be retained for legal/audit purposes
- Ability to "un-delete" if mistake
- Maintains referential integrity
- Queries filter on `deleted_at IS NULL`

### 4. Foreign Key Strategies

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

### 5. Payroll Immutability

**Decision:** Once finalized, payroll results cannot be modified

**Implementation:**

- Status field: DRAFT → FINALIZED
- No UPDATE operations on finalized records
- If error found: DELETE entire period and regenerate

**Rationale:**

- Ensures data integrity
- Matches legal/accounting practices
- Simplifies audit trail

### 6. sqlc Over ORMs

**Decision:** Use sqlc for database access, not GORM or other ORMs

**Rationale:**

- **Full SQL control:** Write exact queries needed
- **Type safety:** Compile-time checks, no runtime surprises
- **No magic:** Generated code is readable and debuggable
- **Performance:** No hidden N+1 queries or lazy loading issues

**Trade-off:** More initial setup, but better long-term maintainability

### 7. Employee Serial Numbers

**Decision:** Generate in application code, not database auto-increment

**Rationale:**

- SQLite auto-increment is global, not per-organization
- Need per-organization numbering (Employee #1 in each org)
- Logic: `SELECT MAX(serial_num) FROM employee WHERE org_id = ? + 1`
- Database unique constraint catches race conditions

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
SELECT * FROM organization WHERE id = ? LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organization WHERE deleted_at IS NULL;
```

**Generate type-safe Go code:**

```bash
sqlc generate
```

**Use in repositories:**

```go
func (r *OrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
    row, err := r.queries.GetOrganization(ctx, id.String())
    // sqlc handles the scanning
}
```

### Error Handling

**Define sentinel errors:**

```go
var (
    ErrNotFound = errors.New("record not found")
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

### Domain Tests

Test validation and business rules without any infrastructure:

```go
func TestEmployee_Validate(t *testing.T) {
    emp := &domain.Employee{
        FullName: "",  // Invalid
    }

    err := emp.Validate()
    require.Error(t, err)
    require.Contains(t, err.Error(), "full_name")
}
```

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

1. **Money as integer cents with overflow protection** -
   Never float64 for financial calculations; always check for overflow
2. **Return errors, don't panic** -
   Financial operations should return `(result, error)` for graceful error handling
3. **Test edge cases thoroughly** -
   Boundary conditions in financial code are subtle (e.g., MaxInt64 boundaries)
4. **Use context.Context everywhere** -
   Enables timeouts and cancellation
5. **sqlc over ORMs** -
   Type safety without magic
6. **`:memory:` for tests** -
   Fast, isolated, auto-cleanup
7. **Comprehensive test coverage** -
   Financial code demands extensive testing
   (unit, integration, edge cases, benchmarks)

### Money Type Lessons

1. **Overflow is real** -
   Even with int64, financial calculations can overflow; always detect and handle
2. **Boundary testing is critical** -
   `0 - (MinInt64 + 1) = MaxInt64` exactly (no overflow),
   but `1 - MinInt64` overflows
3. **Verify with concrete math** -
   Don't assume overflow logic is correct; verify with actual int64 limits
4. **Error handling > silent failures** -
   Better to return an error than silently wrap around to negative
5. **Document edge cases** -
   Non-obvious boundary behaviors should be tested and documented
