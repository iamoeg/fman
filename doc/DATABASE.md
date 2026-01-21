# Database Schema

Complete documentation of the database schema for `finmgmt`.

---

## Table of Contents

1. [Schema Overview](#schema-overview)
2. [Tables](#tables)
3. [Relationships](#relationships)
4. [Foreign Key Strategies](#foreign-key-strategies)
5. [Indexes](#indexes)
6. [Data Types & Conventions](#data-types--conventions)
7. [Query Patterns](#query-patterns)

---

## Schema Overview

The database consists of 5 core tables implementing a multi-tenant payroll system:

```text
organization (1) ──< (N) employee
                 └──< (N) payroll_period

employee_compensation_package (1) ──< (N) employee
                                   └──< (N) payroll_result

payroll_period (1) ──< (N) payroll_result (N) >── employee
```

**Key Principles:**

- Multi-tenant from the start (organization-based)
- Soft deletes on all main tables (`deleted_at`)
- Money stored as integer cents
- Historical accuracy (calculated fields stored, not recomputed)
- Audit trail for compliance

---

## Tables

### 1. `organization`

Represents a company/organization in the system.

**Purpose:** Multi-tenant support - each org is independent

**Fields:**

| Column       | Type | Constraints       | Description                       |
| ------------ | ---- | ----------------- | --------------------------------- |
| `id`         | TEXT | PRIMARY KEY       | UUID                              |
| `name`       | TEXT | NOT NULL          | Organization name                 |
| `address`    | TEXT |                   | Physical address                  |
| `activity`   | TEXT |                   | Business activity description     |
| `legal_form` | TEXT | CHECK IN ('SARL') | Legal structure                   |
| `ice_num`    | TEXT | UNIQUE            | ICE number (Moroccan business ID) |
| `if_num`     | TEXT | UNIQUE            | IF number (tax ID)                |
| `rc_num`     | TEXT | UNIQUE            | RC number (commerce registry)     |
| `cnss_num`   | TEXT | UNIQUE            | CNSS number (social security)     |
| `bank_rib`   | TEXT | UNIQUE            | Bank account RIB                  |
| `created_at` | TEXT | NOT NULL          | ISO 8601 timestamp                |
| `updated_at` | TEXT | NOT NULL          | ISO 8601 timestamp                |
| `deleted_at` | TEXT |                   | Soft delete timestamp             |

**Notes:**

- All Moroccan business identifiers are unique
- Currently only supports SARL legal form (can be extended)

---

### 2. `employee_compensation_package`

Immutable compensation records for historical accuracy.

**Purpose:** Track compensation changes over time,
preserve historical payroll calculations

**Fields:**

| Column              | Type    | Constraints                | Description                   |
| ------------------- | ------- | -------------------------- | ----------------------------- |
| `id`                | TEXT    | PRIMARY KEY                | UUID                          |
| `base_salary_cents` | INTEGER | NOT NULL, >= 0             | Monthly salary in cents (MAD) |
| `currency`          | TEXT    | NOT NULL, CHECK IN ('MAD') | Currency code                 |
| `created_at`        | TEXT    | NOT NULL                   | ISO 8601 timestamp            |
| `updated_at`        | TEXT    | NOT NULL                   | ISO 8601 timestamp            |
| `deleted_at`        | TEXT    |                            | Soft delete timestamp         |

**Design Decision:** These are historical artifacts.
Once referenced by a `payroll_result`, they should NEVER be deleted
(enforced by `ON DELETE RESTRICT`).

**Why separate table:**

- Compensation can change over time
- Need to track which compensation was used for each payroll calculation
- Future: Can add bonuses, allowances, etc.

---

### 3. `employee`

Employee records with demographic and payroll-relevant information.

**Purpose:** Core employee data for payroll and management

**Fields:**

| Column                    | Type    | Constraints                           | Description                |
| ------------------------- | ------- | ------------------------------------- | -------------------------- |
| `id`                      | TEXT    | PRIMARY KEY                           | UUID                       |
| `org_id`                  | TEXT    | FK → organization, NOT NULL           | Owning organization        |
| `serial_num`              | INTEGER | NOT NULL, >= 1                        | Employee number within org |
| `full_name`               | TEXT    | NOT NULL                              | Full legal name            |
| `display_name`            | TEXT    |                                       | Preferred name             |
| `address`                 | TEXT    |                                       | Home address               |
| `email_address`           | TEXT    |                                       | Email                      |
| `phone_number`            | TEXT    |                                       | Phone                      |
| `birth_date`              | TEXT    | NOT NULL                              | Date of birth              |
| `gender`                  | TEXT    | NOT NULL, CHECK IN ('MALE', 'FEMALE') | Gender                     |
| `marital_status`          | TEXT    | NOT NULL, CHECK IN (...)              | Marital status             |
| `num_dependents`          | INTEGER | NOT NULL, >= 0                        | Number of dependents       |
| `num_kids`                | INTEGER | NOT NULL, >= 0                        | Number of children         |
| `cin_num`                 | TEXT    | NOT NULL, UNIQUE                      | Moroccan National ID       |
| `cnss_num`                | TEXT    | UNIQUE                                | Social security number     |
| `hire_date`               | TEXT    | NOT NULL                              | Date of hire               |
| `position`                | TEXT    | NOT NULL                              | Job position/title         |
| `compensation_package_id` | TEXT    | FK → compensation_package, NOT NULL   | Current compensation       |
| `bank_rib`                | TEXT    |                                       | Employee's bank account    |
| `created_at`              | TEXT    | NOT NULL                              | ISO 8601 timestamp         |
| `updated_at`              | TEXT    | NOT NULL                              | ISO 8601 timestamp         |
| `deleted_at`              | TEXT    |                                       | Soft delete timestamp      |

**Constraints:**

- `UNIQUE(org_id, serial_num)` - Employee numbers unique per organization
- `ON DELETE RESTRICT` on compensation_package_id
  -- Can't delete package if employees use it

**Moroccan-Specific Fields:**

- `cin_num`: Carte d'Identité Nationale (National ID) - Required
- `cnss_num`: Social security registration number
- `marital_status`, `num_dependents`, `num_kids`: Affect tax calculations

**Notes:**

- Gender field only has MALE/FEMALE because Moroccan official documents
  don't recognize other options
- Employee numbers generated by application (not database auto-increment)

---

### 4. `payroll_period`

Represents a monthly payroll cycle for an organization.

**Purpose:** Container for a month's payroll, tracks processing status

**Fields:**

| Column         | Type    | Constraints                               | Description           |
| -------------- | ------- | ----------------------------------------- | --------------------- |
| `id`           | TEXT    | PRIMARY KEY                               | UUID                  |
| `org_id`       | TEXT    | FK → organization, NOT NULL               | Organization          |
| `year`         | INTEGER | NOT NULL, 2020-2050                       | Year                  |
| `month`        | INTEGER | NOT NULL, 1-12                            | Month                 |
| `status`       | TEXT    | NOT NULL, CHECK IN ('DRAFT', 'FINALIZED') | Processing status     |
| `finalized_at` | TEXT    |                                           | When finalized        |
| `created_at`   | TEXT    | NOT NULL                                  | ISO 8601 timestamp    |
| `updated_at`   | TEXT    | NOT NULL                                  | ISO 8601 timestamp    |
| `deleted_at`   | TEXT    |                                           | Soft delete timestamp |

**Constraints:**

- `UNIQUE(org_id, year, month)` - One payroll period per org per month
- Status coherence: `(status = 'DRAFT' AND finalized_at IS NULL)
OR (status = 'FINALIZED' AND finalized_at IS NOT NULL)`

**Workflow:**

1. Create period with status='DRAFT'
2. Generate payroll_results for all employees
3. Review and edit if needed
4. Finalize: set status='FINALIZED', set finalized_at
5. Once finalized, period and results are immutable

---

### 5. `payroll_result`

Immutable calculated payroll records - historical snapshot of payroll calculation.

**Purpose:** Complete record of an employee's payroll for a specific period

**Fields:**

| Column                    | Type | Constraints                         | Description                 |
| ------------------------- | ---- | ----------------------------------- | --------------------------- |
| `id`                      | TEXT | PRIMARY KEY                         | UUID                        |
| `payroll_period_id`       | TEXT | FK → payroll_period, NOT NULL       | Which period                |
| `employee_id`             | TEXT | FK → employee, NOT NULL             | Which employee              |
| `compensation_package_id` | TEXT | FK → compensation_package, NOT NULL | Which compensation was used |
| `currency`                | TEXT | NOT NULL, DEFAULT 'MAD'             | Currency                    |

**Salary Components (all in cents):**

| Column                           | Description         |
| -------------------------------- | ------------------- |
| `base_salary_cents`              | Base monthly salary |
| `seniority_bonus_cents`          | Seniority bonus     |
| `gross_salary_cents`             | base + seniority    |
| `total_extra_bonus_cents`        | Other bonuses       |
| `gross_salary_grand_total_cents` | Total gross         |

**Deductions - Employee (all in cents):**

| Column                                         | Description                           |
| ---------------------------------------------- | ------------------------------------- |
| `social_allowance_employee_contrib_cents`      | CNSS social allowance (employee part) |
| `job_loss_compensation_employee_contrib_cents` | CNSS IPE (employee part)              |
| `total_cnss_employee_contrib_cents`            | Total CNSS employee                   |
| `amo_employee_contrib_cents`                   | Health insurance (employee part)      |

**Employer Contributions (all in cents):**

| Column                                         | Description                           |
| ---------------------------------------------- | ------------------------------------- |
| `social_allowance_employer_contrib_cents`      | CNSS social allowance (employer part) |
| `job_loss_compensation_employer_contrib_cents` | CNSS IPE (employer part)              |
| `training_tax_employer_contrib_cents`          | CNSS training tax                     |
| `family_benefits_employer_contrib_cents`       | CNSS family benefits                  |
| `total_cnss_employer_contrib_cents`            | Total CNSS employer                   |
| `amo_employer_contrib_cents`                   | Health insurance (employer part)      |

**Tax Calculation (all in cents):**

| Column                       | Description                 |
| ---------------------------- | --------------------------- |
| `total_exemptions_cents`     | Professional expenses, etc. |
| `taxable_gross_salary_cents` | Gross minus exemptions      |
| `taxable_net_salary_cents`   | After CNSS/AMO deductions   |
| `income_tax_cents`           | IR (progressive tax)        |

**Final Amounts (all in cents):**

| Column                  | Description                        |
| ----------------------- | ---------------------------------- |
| `rounding_amount_cents` | Rounding adjustment (-100 to +100) |
| `net_to_pay_cents`      | Final amount paid to employee      |

**Metadata:**

| Column       | Type | Description     |
| ------------ | ---- | --------------- |
| `created_at` | TEXT | When calculated |
| `updated_at` | TEXT | Last modified   |
| `deleted_at` | TEXT | Soft delete     |

**Constraints:**

- `UNIQUE(payroll_period_id, employee_id)` - One result per employee per period

**Critical Design Decision:**
All calculated fields are stored, not recomputed. This is intentional:

- Historical accuracy: Shows what was actually calculated/paid
- Legal compliance: Tax laws change over time
- Performance: No need to recalculate for reports
- Audit trail: Permanent record

Once finalized, these records are IMMUTABLE.
If there's an error, delete the entire period and regenerate.

---

### 6. `audit_log`

Change tracking for compliance and debugging.

**Purpose:** Track all changes to records for audit and debugging

**Fields:**

| Column       | Type | Constraints              | Description                            |
| ------------ | ---- | ------------------------ | -------------------------------------- |
| `id`         | TEXT | PRIMARY KEY              | UUID                                   |
| `table_name` | TEXT | NOT NULL                 | Which table was modified               |
| `record_id`  | TEXT | NOT NULL                 | Which record (UUID)                    |
| `action`     | TEXT | NOT NULL, CHECK IN (...) | CREATE, UPDATE, SOFT_DELETE, DELETE    |
| `before`     | TEXT | json_valid()             | JSON snapshot before (null for CREATE) |
| `after`      | TEXT | NOT NULL, json_valid()   | JSON snapshot after                    |
| `timestamp`  | TEXT | NOT NULL                 | When the change occurred               |

**Indexes:**

- Composite on `(table_name, record_id)` - Query specific record history
- Single on `timestamp DESC` - Query recent changes

**Notes:**

- `before` and `after` are complete JSON representations of the record
- Provides full audit trail for compliance
- Can be used to understand data changes over time

---

## Relationships

### Organization is the Root

- One organization has many employees
- One organization has many payroll periods
- Deleting an organization cascades to its employees and periods

### Compensation Packages are Shared

- One compensation package can be used by many employees
  (e.g., "Junior Developer Package")
- One compensation package can be referenced by many payroll results
- Compensation packages CANNOT be deleted if referenced (RESTRICT)

### Payroll Structure

```text
Organization
    ↓ (1:N)
PayrollPeriod (December 2025)
    ↓ (1:N)
PayrollResult (one per employee)
    ├→ Employee (who)
    └→ CompensationPackage (what compensation was used)
```

---

## Foreign Key Strategies

### CASCADE - Delete children when parent deleted

**When to use:** Parent-child relationships where children
are meaningless without the parent

| Child Table      | Parent Table     | Rationale                                          |
| ---------------- | ---------------- | -------------------------------------------------- |
| `employee`       | `organization`   | Employees belong to an org; meaningless without it |
| `payroll_period` | `organization`   | Periods belong to an org; meaningless without it   |
| `payroll_result` | `payroll_period` | Results are part of a period; delete together      |
| `payroll_result` | `employee`       | If employee deleted, their payroll goes too        |

**Note:** We use soft deletes (`deleted_at`), so CASCADE rarely triggers.
Hard delete only happens for data purging (GDPR, test data).

### RESTRICT - Cannot delete if children exist

**When to use:** Historical artifacts that must be preserved

| Child Table      | Parent Table           | Rationale                                                 |
| ---------------- | ---------------------- | --------------------------------------------------------- |
| `employee`       | `compensation_package` | Can't delete package if employees use it                  |
| `payroll_result` | `compensation_package` | Historical record - must preserve exact compensation used |

**Critical:** Compensation packages referenced by payroll results
are **permanent historical artifacts**.
They cannot be deleted because they're part of the audit trail.

---

## Indexes

All indexes are created to support common query patterns:

| Index Name                           | Table            | Columns                 | Purpose                               |
| ------------------------------------ | ---------------- | ----------------------- | ------------------------------------- |
| `idx_employee_org_id`                | `employee`       | `org_id`                | Find employees by organization        |
| `idx_payroll_period_org_id`          | `payroll_period` | `org_id`                | Find periods by organization          |
| `idx_payroll_result_period_id`       | `payroll_result` | `payroll_period_id`     | Find results for a period             |
| `idx_payroll_result_employee_id`     | `payroll_result` | `employee_id`           | Find payroll history for employee     |
| `idx_audit_log_table_name_record_id` | `audit_log`      | `table_name, record_id` | Query audit trail for specific record |
| `idx_audit_log_timestamp`            | `audit_log`      | `timestamp DESC`        | Query recent changes                  |

**Note:** SQLite automatically indexes PRIMARY KEY and UNIQUE columns.

---

## Data Types & Conventions

### UUIDs

- **Storage:** TEXT (RFC 4122 format: `550e8400-e29b-41d4-a716-446655440000`)
- **Generation:** In application code using `google/uuid`
- **Conversion:** `uuid.String()` to store, `uuid.Parse()` to read

### Timestamps

- **Storage:** TEXT in ISO 8601 / RFC 3339 format (`2025-01-09T10:30:45Z`)
- **Generation:** In application code: `time.Now().UTC().Format(time.RFC3339)`
- **Conversion:** `time.Parse(time.RFC3339, storedValue)`
- **Why not INTEGER:** Human-readable, works with SQLite date functions

### Money

- **Storage:** INTEGER in cents (Moroccan Dirhams × 100)
- **Example:** 10,234.56 MAD → stored as 1023456
- **Why:** Avoid floating-point precision errors
- **Application:** Use custom `Money` type in `pkg/money/`

### Enums

- **Storage:** TEXT with CHECK constraints
- **Example:** `gender TEXT CHECK(gender IN ('MALE', 'FEMALE'))`
- **Why:** SQLite has no native ENUM type

### Soft Deletes

- **Pattern:** `deleted_at` column (TEXT, nullable)
- **Active records:** `deleted_at IS NULL`
- **Deleted records:** `deleted_at IS NOT NULL` (timestamp of deletion)
- **Queries:** Always filter on `deleted_at IS NULL` unless querying deleted records

---

## Query Patterns

### Overview

All database interactions use sqlc-generated code from SQL query files in `db/query/`.
Queries follow consistent patterns for predictability and maintainability.

### Standard CRUD Pattern

Every entity follows this pattern:

```sql
-- Get single record (active only)
-- name: GetEntity :one
SELECT * FROM entity WHERE id = ? AND deleted_at IS NULL;

-- Get single record (including deleted)
-- name: GetEntityIncludingDeleted :one
SELECT * FROM entity WHERE id = ?;

-- List all records (active only)
-- name: ListEntities :many
SELECT * FROM entity WHERE deleted_at IS NULL ORDER BY ...;

-- List all records (including deleted)
-- name: ListEntitiesIncludingDeleted :many
SELECT * FROM entity ORDER BY ...;

-- Create new record
-- name: CreateEntity :one
INSERT INTO entity(...) VALUES (...) RETURNING *;

-- Update existing record
-- name: UpdateEntity :one
UPDATE entity SET ... WHERE id = ? AND deleted_at IS NULL RETURNING *;

-- Soft delete
-- name: DeleteEntity :exec
UPDATE entity SET deleted_at = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL;

-- Restore deleted record
-- name: RestoreEntity :one
UPDATE entity SET deleted_at = NULL, updated_at = ?
WHERE id = ? AND deleted_at IS NOT NULL RETURNING *;

-- Hard delete (permanent)
-- name: HardDeleteEntity :exec
DELETE FROM entity WHERE id = ?;
```

### Query Annotations

- `:one` - Returns single row or error (sql.ErrNoRows if not found)
- `:many` - Returns slice of rows (empty slice if none found)
- `:exec` - Executes query, returns only error

### Filtering Strategy

**Design Decision:** Queries are "primitives" without built-in business logic filtering.

**Rationale:**

- Users need access to archived (soft-deleted) data
- Repository layer controls which query variant to use
- Provides maximum flexibility

**Pattern:**

```go
// Repository methods choose appropriate query
func (r *Repo) FindByID(ctx, id) (*Entity, error) {
    // Use filtered query
    return r.queries.GetEntity(ctx, id)
}

func (r *Repo) FindByIDIncludingDeleted(ctx, id) (*Entity, error) {
    // Use unfiltered query
    return r.queries.GetEntityIncludingDeleted(ctx, id)
}
```

### Immutability Patterns

**Employee Serial Numbers:**

- Generated by application, not database
- Never updated after creation
- Query: `GetNextSerialNumber` provides next available number per org

**Payroll Period Workflow:**

- Status changes use explicit queries: `FinalizePayrollPeriod`, `UnfinalizePayrollPeriod`
- Prevents accidental status changes
- Enforces CHECK constraint: finalized_at must be set when status = 'FINALIZED'

**Payroll Results:**

- No UPDATE query - results are immutable once created
- If changes needed: DELETE and recreate
- Enforces historical accuracy requirement

**Compensation Packages:**

- UPDATE query exists but should be guarded at repository level
- Repository checks if package is in use before allowing update
- Query: `CountEmployeesUsingCompensationPackage` and `CountPayrollResultsUsingCompensationPackage`

### Audit Trail Pattern

All repository mutations should create audit log entries:

```go
func (r *Repo) Create(ctx context.Context, entity *Entity) error {
    tx, _ := r.db.BeginTx(ctx, nil)
    defer tx.Rollback()

    qtx := r.queries.WithTx(tx)

    // Create the entity
    qtx.CreateEntity(ctx, ...)

    // Audit the creation
    after, _ := json.Marshal(entity)
    qtx.CreateAuditLog(ctx, AuditLogParams{
        ID: uuid.New().String(),
        TableName: "entity",
        RecordID: entity.ID.String(),
        Action: "CREATE",
        Before: sql.NullString{Valid: false},
        After: string(after),
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    })

    return tx.Commit()
}
```

**Audit Log Queries:**

- `ListAuditLogsForRecord` - Complete history for one record
- `ListAuditLogsRecent` - Recent changes across all tables
- `ListAuditLogsByTable` - All changes to specific table
- `ListAuditLogsByAction` - All changes of specific type (CREATE, UPDATE, etc.)

### Specialized Queries

**Employee:**

- `GetEmployeeByOrgAndSerialNum` - Lookup by employee number
- `ListEmployeesByOrganization` - All employees in org
- `GetNextSerialNumber` - Generate next employee number
- `CountEmployeesUsingCompensationPackage` - Guard for package updates

**Payroll Period:**

- `GetPayrollPeriodByOrgYearMonth` - Lookup by period identifier
- `ListPayrollPeriodsByOrganization` - All periods for org
- `ListDraftPayrollPeriods` - Find unfinalized periods
- `FinalizePayrollPeriod` - Workflow: DRAFT → FINALIZED
- `UnfinalizePayrollPeriod` - Workflow: FINALIZED → DRAFT (error correction)

**Payroll Result:**

- `ListPayrollResultsByPayrollPeriod` - All results for one period
- `ListPayrollResultsByEmployee` - Payroll history for employee
- `CountPayrollResultsUsingCompensationPackage` - Guard for package updates

### Query File Organization

```text
db/query/
├── organization.sql                   # 8 queries
├── employee.sql                       # 12 queries
├── employee_compensation_package.sql  # 8 queries
├── payroll_period.sql                 # 15 queries
├── payroll_result.sql                 # 11 queries
└── audit_log.sql                      # 5 queries
Total: 59 queries
```
