# System Architecture

This document describes the architecture of `finmgmt`:
structural patterns, layer responsibilities, and the rationale behind key decisions.

---

## Table of Contents

1. [Architectural Pattern](#architectural-pattern)
2. [Layer Responsibilities](#layer-responsibilities)
3. [Key Design Decisions](#key-design-decisions)
4. [Common Pitfalls](#common-pitfalls)
5. [Lessons Learned](#lessons-learned)

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

**The dependency arrows point inward:** Adapters → Application → Domain

---

## Layer Responsibilities

### Domain Layer (`internal/domain/`)

**Purpose:** Core business entities and rules

**Allowed:**

- ✅ Pure Go structs
- ✅ Validation methods
- ✅ Business logic and enums
- ✅ Helper methods (e.g., `TotalDueToCNSS`)
- ✅ Dependencies: `uuid`, `time`, `pkg/money`

**Not Allowed:**

- ❌ Database tags or external dependencies
- ❌ Framework code or I/O operations

### Application Layer (`internal/application/`)

**Purpose:** Orchestrate business logic, define service interfaces

**Responsibilities:**

- Define small, focused interfaces for dependencies
- Coordinate domain objects and implement use cases
- Set timestamps and generate UUIDs
- Manage transaction boundaries
- Validate domain rules before persistence
- Translate repository errors to service-level errors

**Implemented Services:**

1. **OrganizationService** — CRUD, soft deletes, duplicate detection for business identifiers
2. **EmployeeService** — Per-organization serial number generation, multi-tenant isolation, compensation relationship
3. **CompensationPackageService** — Historical artifact protection, usage guards, SMIG validation
4. **PayrollService** — Multi-repository coordination, workflow state management (DRAFT → FINALIZED),
   batch payroll generation; delegates calculation to a `payrollCalculator` interface

### Adapter Layer (`internal/adapter/`)

**Purpose:** Implement technical details

**Responsibilities:**

- Implement repository interfaces (implicitly via structural typing)
- Database operations
- External service integrations and file I/O

**Key Point:** Adapters don't declare "implements" — they just provide the methods
that satisfy the interfaces defined by application services.

### UI Layer (`ui/tui/`)

**Purpose:** User interface

**Allowed:**

- ✅ Display data, handle user input, call application services, manage UI state

**Not Allowed:**

- ❌ Business logic, direct repository access, database operations

---

## Key Design Decisions

### 1. Money as Integer Cents

**Decision:** Store all monetary values as integers (cents/smallest unit)
and return errors from arithmetic operations.

**Rationale:** Floating-point arithmetic is imprecise for financial calculations
(`0.1 + 0.2 = 0.30000000000000004`). Integer cents guarantee exact results.
Operations that can fail (overflow, division by zero) return explicit errors
rather than silently producing wrong values.

### 1a. Currency Not Embedded in `Money`

**Decision:** The `Money` struct holds only `cents int64`. Currency is not tracked
per-value. A separate `Currency` type exists in `pkg/money/currency.go` for
validation at system boundaries (DB, config), but is not embedded in `Money`.

**Rationale:** The system is explicitly single-currency (MAD) at every layer —
the domain enforces MAD-only compensation packages, and the DB has constraints in place.
Embedding currency in every `Money` value would complicate all arithmetic
(every `Add`/`Subtract` would need a currency-equality check)
without enabling any real use case in a single-currency system.
The correctness risk of mixing currencies does not exist here.

**When to revisit:** If Phase 2+ introduces multi-currency invoicing or
foreign-currency expense tracking, add currency to the specific domain types
that need it (e.g. `Invoice`) rather than retrofitting `Money`. At that point,
a `CurrencyAmount` wrapper type at system boundaries may be appropriate.

### 2. Calculated Fields in Payroll Results

**Decision:** Store all calculated values in `payroll_result`, not just base values.

**Rationale:** Payroll is a legal document. Tax laws and calculation logic change over time.
Storing the result of each calculation proves what was actually paid, not what would be
calculated today. Also eliminates recomputation for reports.

**Trade-off:** Some data redundancy, but financial/legal requirements justify it.

### 3. Comprehensive Domain Validation

**Decision:** Validate all business rules in the domain layer with detailed error messages.

**Rationale:** Every domain entity has a `Validate()` method with per-field sentinel errors.
This catches errors before the database, is trivially testable (no dependencies),
and documents business rules in the code itself.

### 4. CNSS and AMO Separation

**Decision:** Keep CNSS and AMO contributions as distinct fields; provide helpers for combined totals.

**Rationale:** CNSS (social security) and AMO (health insurance) are legally distinct entities,
even though AMO is collected by CNSS in practice. Separation enables accurate reporting and
a clear audit trail. `TotalDueToCNSS()` is a helper that combines them where needed.

### 5. Soft Deletes

**Decision:** Never hard-delete financial data; use `deleted_at` timestamps instead.

**Rationale:** Financial data must be retained for legal and audit purposes.
Soft deletes allow recovery from mistakes and maintain referential integrity.
All queries filter on `deleted_at IS NULL` for active records.

### 6. Foreign Key Strategies

**CASCADE** — delete children when parent is deleted:

- `employee.org_id` → organization
- `payroll_period.org_id` → organization
- `payroll_result.payroll_period_id` → payroll_period
- `payroll_result.employee_id` → employee

**RESTRICT** — cannot delete if children exist:

- `employee.compensation_package_id` → employee_compensation_package
- `payroll_result.compensation_package_id` → employee_compensation_package

**Rationale:** Compensation packages are historical artifacts. Once referenced by
a payroll result, they are part of the permanent legal record and cannot be deleted.

### 7. Payroll Immutability

**Decision:** Once a payroll period is finalized, its results cannot be modified.

**Implementation:** Status transitions: DRAFT → FINALIZED. No UPDATE query exists
for `payroll_result`. If a correction is needed, delete the entire period and regenerate.

**Rationale:** Matches legal/accounting practices and simplifies the audit trail.

### 8. sqlc Over ORMs

**Decision:** Use sqlc for database access, not GORM or similar ORMs.

**Rationale:** Full SQL control, compile-time type safety, no hidden N+1 queries,
and generated code that is readable and debuggable. The trade-off is more initial
setup, which is justified by long-term maintainability.

### 9. Employee Serial Numbers

**Decision:** Generate serial numbers in application code, not via database auto-increment.

**Rationale:** SQLite auto-increment is global. We need per-organization numbering
(Employee #1 in each org). Logic: `MAX(serial_num) + 1` scoped to `org_id`.
The database unique constraint on `(org_id, serial_num)` catches any race conditions.

### 10. Enum Pattern

**Decision:** Use typed string constants with a map for O(1) validation and pre-computed
error strings.

**Rationale:** Avoids linear scans for validation, produces clean error messages,
and is easy to extend. All enums follow the same pattern for consistency.

### 11. Primitive SQL Queries, Filtering at Repository Level

**Decision:** SQL queries in `db/query/` are primitive (no built-in soft-delete filtering).
Repository methods choose which query variant to use.

**Rationale:** Some features (archive views, admin tools) legitimately need access to
soft-deleted records. The repository is the right abstraction layer for this filtering,
not SQL. A single query serves multiple repository methods.

### 12. Immutable Field Protection

**Decision:** Exclude certain fields from UPDATE queries entirely.

**Fields protected:**

- Employee: `org_id`, `serial_num` — define identity
- Payroll Period: `org_id`, `year`, `month` — define the period
- Payroll Result: no UPDATE query at all

**Rationale:** SQL-level protection is stronger than application-level guards.
Changing `serial_num` or moving a payroll period between orgs makes no business sense
and would corrupt the audit trail. If wrong: delete and recreate.

### 13. Explicit Workflow Queries

**Decision:** Dedicated queries for state transitions instead of generic UPDATEs.

**Example:** `FinalizePayrollPeriod` sets `status = 'FINALIZED'` and `finalized_at`
in a single query with `WHERE status = 'DRAFT'`. A generic `UPDATE` could accidentally
finalize an already-finalized period or clear `finalized_at`.

**Rationale:** Workflow queries enforce business rules at the SQL level, are
self-documenting, and generate type-safe sqlc functions.

### 14. Calculation Engine with Year-Keyed Rate Tables

**Decision:** Implement the payroll calculator as a single adapter package
(`internal/adapter/payroll/`) with an interface defined by `PayrollService`,
not by the calculator itself. Year-specific legislation (rates, ceilings, brackets)
is isolated in a `yearRates` struct stored in a `ratesByYear` registry map.
`Calculate()` looks up `period.Year` in the registry and returns
`ErrUnsupportedPayrollYear` if no entry exists.

**Rationale:** Moroccan payroll rates change yearly. Rather than creating a new
package per year (high overhead, wiring churn), all year-specific *data* lives
in one place — the `ratesByYear` map. Adding 2027 support means adding one map
entry; the calculation *algorithm* never changes. `PayrollService` never needs
to change when rates change.

The calculator is intentionally stateless and pure: inputs in, result out.
This makes it trivially testable and avoids any need for year-to-date accumulation state.

---

## Common Pitfalls

1. **Forgetting `defer tx.Rollback()`** — always defer immediately after `BeginTx`;
   it becomes a no-op after a successful commit.

2. **Using `r.queries` instead of `qtx` inside a transaction** — operations outside `qtx`
   are not part of the transaction and will not roll back on failure.

3. **Missing soft-delete filter** — `db/query/` queries are primitive; repositories
   must check `DeletedAt != nil` and return `ErrRecordNotFound` accordingly.

4. **Non-atomic test data counters** — use `atomic.AddInt64` for unique field values
   in parallel tests to avoid `UNIQUE` constraint violations.

5. **Timestamp comparison failures** — always normalize with `.UTC().Truncate(time.Second)`
   before comparing; storage precision and timezone can differ.

6. **SQLite foreign keys** — must explicitly enable with `PRAGMA foreign_keys = ON`
   on every connection; SQLite disables them by default.

---

## Lessons Learned

### Architecture

1. **Go idioms > cargo-culting Java patterns** — no separate `ports/` package needed
2. **Dependency direction matters, not directory structure** — inward toward domain is what counts
3. **Small, focused interfaces** — services define only what they need
4. **`db/` for SQL, `internal/` for Go** — don't force SQL into Go package structure

### Domain Design

1. **Money type is fundamental** — build it first; everything depends on it
2. **Validation in domain is powerful** — catches errors early, trivially testable
3. **Cross-field validation needs care** — e.g., `BirthDate` vs `HireDate`, `Status` vs `FinalizedAt`
4. **Helper methods on entities** — `TotalDueToCNSS()` makes business logic clearer

### Database

1. **Calculated fields in payroll are correct** — historical accuracy trumps normalization
2. **ON DELETE RESTRICT for historical artifacts** — preserve the audit trail
3. **Soft deletes everywhere** — financial data shouldn't disappear
4. **PRAGMA foreign_keys = ON** — must enable on every SQLite connection

### Database Queries

1. **Primitive queries provide flexibility** — let the repository layer handle filtering
2. **Immutability at SQL level is powerful** — exclude fields from UPDATE to prevent accidents
3. **Explicit workflow queries are safer** — `FinalizePayrollPeriod` vs a generic UPDATE
4. **Consistent patterns reduce cognitive load** — every entity follows the same query structure
5. **sqlc annotations must match intent** — `:one` needs `RETURNING *`, `:exec` doesn't

### Go Practices

1. **Integer cents, never float64** — for all financial calculations
2. **Error returns from Money operations** — prevents silent failures
3. **Use `context.Context` everywhere** — enables timeouts and cancellation
4. **sqlc over ORMs** — type safety without magic
5. **`:memory:` SQLite for tests** — fast, isolated, no cleanup needed

### Testing

1. **Test domain first** — pure logic, no dependencies, easy to test
2. **Realistic test data matters** — Moroccan names, amounts, and dates make tests meaningful
3. **Table-driven tests with `t.Parallel()`** — standard Go best practice
