# Database Schema

Complete documentation of the database schema for `finmgmt`.
The canonical schema definition is in `db/migration/00001_initial_schema.sql`.
The canonical DBML diagram is in `doc/db-schema.dbml`.

---

## Table of Contents

1. [Schema Overview](#schema-overview)
2. [Tables](#tables)
3. [Relationships](#relationships)
4. [Foreign Key Strategies](#foreign-key-strategies)
5. [Indexes](#indexes)
6. [Data Types & Conventions](#data-types--conventions)
7. [Query Organization](#query-organization)

---

## Schema Overview

The database consists of 5 core tables plus an audit log:

```text
organization (1) ──< (N) employee
                 └──< (N) payroll_period
                 └──< (N) employee_compensation_package

employee_compensation_package (1) ──< (N) employee
                                   └──< (N) payroll_result

payroll_period (1) ──< (N) payroll_result (N) >── employee
```

**Key Principles:**

- Multi-tenant from the start (organization-scoped)
- Soft deletes on all main tables (`deleted_at`)
- Money stored as integer cents
- Historical accuracy — calculated fields stored, not recomputed
- Audit trail for compliance

---

## Tables

### `organization`

Represents a company in the system. Root of the multi-tenant hierarchy.

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
| `cnss_num`   | TEXT | UNIQUE            | CNSS registration number          |
| `bank_rib`   | TEXT | UNIQUE            | Bank account RIB                  |
| `created_at` | TEXT | NOT NULL          | ISO 8601 timestamp                |
| `updated_at` | TEXT | NOT NULL          | ISO 8601 timestamp                |
| `deleted_at` | TEXT |                   | Soft delete timestamp             |

---

### `employee_compensation_package`

Immutable compensation records. Treated as historical artifacts once referenced
by a payroll result — cannot be deleted while in use (`ON DELETE RESTRICT`).

A separate table (rather than inline fields on `employee`) allows tracking
compensation changes over time and preserves the exact package used for each
payroll calculation.

| Column              | Type    | Constraints                 | Description                   |
| ------------------- | ------- | --------------------------- | ----------------------------- |
| `id`                | TEXT    | PRIMARY KEY                 | UUID                          |
| `org_id`            | TEXT    | FK → organization, NOT NULL | Owning organization           |
| `name`              | TEXT    | NOT NULL                    | Human-readable package name   |
| `base_salary_cents` | INTEGER | NOT NULL, >= 0              | Monthly salary in cents (MAD) |
| `currency`          | TEXT    | NOT NULL, CHECK IN ('MAD')  | Currency code                 |
| `created_at`        | TEXT    | NOT NULL                    | ISO 8601 timestamp            |
| `updated_at`        | TEXT    | NOT NULL                    | ISO 8601 timestamp            |
| `deleted_at`        | TEXT    |                             | Soft delete timestamp         |

**Index:** `idx_comp_package_org_id` on `(org_id)` for efficient org-scoped lookups.

---

### `employee`

Employee records with demographic and payroll-relevant information.

| Column                    | Type    | Constraints                           | Description                                                              |
| ------------------------- | ------- | ------------------------------------- | ------------------------------------------------------------------------ |
| `id`                      | TEXT    | PRIMARY KEY                           | UUID                                                                     |
| `org_id`                  | TEXT    | FK → organization, NOT NULL           | Owning organization                                                      |
| `serial_num`              | INTEGER | NOT NULL, >= 1                        | Employee number within org                                               |
| `full_name`               | TEXT    | NOT NULL                              | Full legal name                                                          |
| `display_name`            | TEXT    |                                       | Preferred name                                                           |
| `address`                 | TEXT    |                                       | Home address                                                             |
| `email_address`           | TEXT    |                                       | Email                                                                    |
| `phone_number`            | TEXT    |                                       | Phone                                                                    |
| `birth_date`              | TEXT    | NOT NULL                              | Date of birth                                                            |
| `gender`                  | TEXT    | NOT NULL, CHECK IN ('MALE', 'FEMALE') | Gender                                                                   |
| `marital_status`          | TEXT    | NOT NULL, CHECK IN (...)              | Marital status                                                           |
| `num_dependents`          | INTEGER | NOT NULL, >= 0                        | Tax dependents (spouse + children) — used for IR family charge deduction |
| `num_children`            | INTEGER | NOT NULL, >= 0                        | Qualifying children — used for CNSS allocations familiales               |
| `cin_num`                 | TEXT    | NOT NULL, UNIQUE                      | Moroccan National ID (CIN)                                               |
| `cnss_num`                | TEXT    | UNIQUE                                | Social security number                                                   |
| `hire_date`               | TEXT    | NOT NULL                              | Date of hire                                                             |
| `position`                | TEXT    | NOT NULL                              | Job position/title                                                       |
| `compensation_package_id` | TEXT    | FK → compensation_package, NOT NULL   | Current compensation                                                     |
| `bank_rib`                | TEXT    |                                       | Employee's bank account                                                  |
| `created_at`              | TEXT    | NOT NULL                              | ISO 8601 timestamp                                                       |
| `updated_at`              | TEXT    | NOT NULL                              | ISO 8601 timestamp                                                       |
| `deleted_at`              | TEXT    |                                       | Soft delete timestamp                                                    |

**Constraints:**

- `UNIQUE(org_id, serial_num)` — employee numbers are unique per organization
- `gender` only has MALE/FEMALE because Moroccan official documents don't recognize other options
- Serial numbers are generated by the application (not database auto-increment) to support per-org numbering

---

### `payroll_period`

Represents a monthly payroll cycle for an organization.

| Column         | Type    | Constraints                               | Description           |
| -------------- | ------- | ----------------------------------------- | --------------------- |
| `id`           | TEXT    | PRIMARY KEY                               | UUID                  |
| `org_id`       | TEXT    | FK → organization, NOT NULL               | Organization          |
| `year`         | INTEGER | NOT NULL, 2020–2050                       | Year                  |
| `month`        | INTEGER | NOT NULL, 1–12                            | Month                 |
| `status`       | TEXT    | NOT NULL, CHECK IN ('DRAFT', 'FINALIZED') | Processing status     |
| `finalized_at` | TEXT    |                                           | When finalized        |
| `created_at`   | TEXT    | NOT NULL                                  | ISO 8601 timestamp    |
| `updated_at`   | TEXT    | NOT NULL                                  | ISO 8601 timestamp    |
| `deleted_at`   | TEXT    |                                           | Soft delete timestamp |

**Constraints:**

- `UNIQUE(org_id, year, month)` — one payroll period per org per month
- Status coherence: `(status = 'DRAFT' AND finalized_at IS NULL) OR (status = 'FINALIZED' AND finalized_at IS NOT NULL)`

**Workflow:** Create (DRAFT) → generate results → review → Finalize. Once finalized, period and its results are immutable. If an error is found, unfinalize, delete results, and regenerate.

---

### `payroll_result`

Complete, immutable snapshot of an employee's payroll for one period.
All calculated values are stored (not recomputed) for historical accuracy and legal compliance.
There is **no UPDATE query** for this table — corrections require deleting and recreating.

| Column                    | Type | Constraints                         | Description                 |
| ------------------------- | ---- | ----------------------------------- | --------------------------- |
| `id`                      | TEXT | PRIMARY KEY                         | UUID                        |
| `payroll_period_id`       | TEXT | FK → payroll_period, NOT NULL       | Which period                |
| `employee_id`             | TEXT | FK → employee, NOT NULL             | Which employee              |
| `compensation_package_id` | TEXT | FK → compensation_package, NOT NULL | Which compensation was used |
| `currency`                | TEXT | NOT NULL, DEFAULT 'MAD'             | Currency                    |

**Salary components (cents):**

| Column                           | Description                                                 |
| -------------------------------- | ----------------------------------------------------------- |
| `base_salary_cents`              | Base monthly salary                                         |
| `seniority_bonus_cents`          | Seniority bonus                                             |
| `gross_salary_cents`             | base + seniority                                            |
| `total_extra_bonus_cents`        | Other bonuses                                               |
| `gross_salary_grand_total_cents` | Total gross                                                 |
| `family_allowance_cents`         | CNSS allocations familiales (tax-exempt income to employee) |

**Employee deductions (cents):**

| Column                                         | Description                           |
| ---------------------------------------------- | ------------------------------------- |
| `social_allowance_employee_contrib_cents`      | CNSS social allowance (employee part) |
| `job_loss_compensation_employee_contrib_cents` | CNSS IPE (employee part)              |
| `total_cnss_employee_contrib_cents`            | Total CNSS employee                   |
| `amo_employee_contrib_cents`                   | AMO health insurance (employee part)  |

**Employer contributions (cents):**

| Column                                         | Description                           |
| ---------------------------------------------- | ------------------------------------- |
| `social_allowance_employer_contrib_cents`      | CNSS social allowance (employer part) |
| `job_loss_compensation_employer_contrib_cents` | CNSS IPE (employer part)              |
| `training_tax_employer_contrib_cents`          | CNSS training tax                     |
| `family_benefits_employer_contrib_cents`       | CNSS family benefits                  |
| `total_cnss_employer_contrib_cents`            | Total CNSS employer                   |
| `amo_employer_contrib_cents`                   | AMO health insurance (employer part)  |

**Tax calculation (cents):**

| Column                       | Description                 |
| ---------------------------- | --------------------------- |
| `total_exemptions_cents`     | Professional expenses, etc. |
| `taxable_gross_salary_cents` | Gross minus exemptions      |
| `taxable_net_salary_cents`   | After CNSS/AMO deductions   |
| `income_tax_cents`           | IR (progressive tax)        |

**Final amounts (cents):**

| Column                  | Description                        |
| ----------------------- | ---------------------------------- |
| `rounding_amount_cents` | Rounding adjustment (−100 to +100) |
| `net_to_pay_cents`      | Final amount paid to employee      |

**Metadata:** `created_at`, `updated_at`, `deleted_at`

**Constraints:** `UNIQUE(payroll_period_id, employee_id)` — one result per employee per period.

---

### `audit_log`

Append-only change tracking for compliance and debugging.
Entries are created automatically inside other repositories' transactions — never directly.

| Column       | Type | Constraints              | Description                                  |
| ------------ | ---- | ------------------------ | -------------------------------------------- |
| `id`         | TEXT | PRIMARY KEY              | UUID                                         |
| `table_name` | TEXT | NOT NULL                 | Which table was modified                     |
| `record_id`  | TEXT | NOT NULL                 | Which record (UUID)                          |
| `action`     | TEXT | NOT NULL, CHECK IN (...) | CREATE, UPDATE, DELETE, RESTORE, HARD_DELETE |
| `before`     | TEXT | json_valid()             | JSON snapshot before (null for CREATE)       |
| `after`      | TEXT | NOT NULL, json_valid()   | JSON snapshot after                          |
| `timestamp`  | TEXT | NOT NULL                 | When the change occurred                     |

---

## Relationships

- **Organization is the root.** One org has many employees and many payroll periods. Deleting an org cascades to both.
- **Compensation packages are shared.** One package can be used by multiple employees and referenced by multiple payroll results. They cannot be deleted while referenced (`RESTRICT`).
- **Payroll structure:**

```text
Organization
    ↓ (1:N)
PayrollPeriod (e.g., December 2025)
    ↓ (1:N)
PayrollResult (one per employee)
    ├→ Employee (who)
    └→ CompensationPackage (what compensation was used)
```

---

## Foreign Key Strategies

### CASCADE — delete children when parent is deleted

Used when child records are meaningless without the parent.

| Child Table      | Parent Table     | Rationale                                      |
| ---------------- | ---------------- | ---------------------------------------------- |
| `employee`       | `organization`   | Employees belong to an org                     |
| `payroll_period` | `organization`   | Periods belong to an org                       |
| `payroll_result` | `payroll_period` | Results are part of a period; delete together  |
| `payroll_result` | `employee`       | If employee is deleted, their payroll goes too |

Note: soft deletes mean CASCADE rarely triggers in practice. Hard deletes are for data purging only.

### RESTRICT — cannot delete if children exist

Used when the parent is a historical artifact that must be preserved.

| Child Table      | Parent Table           | Rationale                                                     |
| ---------------- | ---------------------- | ------------------------------------------------------------- |
| `employee`       | `compensation_package` | Can't delete a package if employees are using it              |
| `payroll_result` | `compensation_package` | Historical record — must preserve the exact compensation used |

Once a compensation package is referenced by a payroll result, it is a permanent artifact.

---

## Indexes

| Index Name                           | Table            | Columns                 | Purpose                               |
| ------------------------------------ | ---------------- | ----------------------- | ------------------------------------- |
| `idx_employee_org_id`                | `employee`       | `org_id`                | Find employees by organization        |
| `idx_payroll_period_org_id`          | `payroll_period` | `org_id`                | Find periods by organization          |
| `idx_payroll_result_period_id`       | `payroll_result` | `payroll_period_id`     | Find results for a period             |
| `idx_payroll_result_employee_id`     | `payroll_result` | `employee_id`           | Find payroll history for employee     |
| `idx_audit_log_table_name_record_id` | `audit_log`      | `table_name, record_id` | Query audit trail for specific record |
| `idx_audit_log_timestamp`            | `audit_log`      | `timestamp DESC`        | Query recent changes                  |

SQLite automatically indexes PRIMARY KEY and UNIQUE columns.

---

## Data Types & Conventions

### UUIDs

Stored as TEXT in RFC 4122 format (`550e8400-e29b-41d4-a716-446655440000`).
Generated in application code using `google/uuid`. Never generated by the database.

### Timestamps

Stored as TEXT in ISO 8601 / RFC 3339 format (`2025-01-09T10:30:45Z`).
Always generated and stored in UTC. Human-readable and compatible with SQLite date functions.

### Money

Stored as INTEGER in cents (Moroccan Dirhams × 100). Example: 10,234.56 MAD → `1023456`.
Avoids floating-point precision errors. Use the `pkg/money` type in application code.

### Enums

Stored as TEXT with CHECK constraints (SQLite has no native ENUM type).
Example: `gender TEXT CHECK(gender IN ('MALE', 'FEMALE'))`.

### Soft Deletes

`deleted_at` column (TEXT, nullable). Active records have `deleted_at IS NULL`.
All standard queries filter on this. Use `IncludingDeleted` query variants when access
to archived records is needed.

---

## Query Organization

All database interactions use sqlc-generated code from SQL files in `db/query/`.

```text
db/query/
├── organization.sql
├── employee.sql
├── employee_compensation_package.sql
├── payroll_period.sql
├── payroll_result.sql
└── audit_log.sql
```

**Naming conventions:**

- `Get{Entity}` / `Get{Entity}By{Criteria}` — single record
- `List{Entities}` / `List{Entities}By{Criteria}` — multiple records
- `Get{Entity}IncludingDeleted` / `List{Entities}IncludingDeleted` — includes soft-deleted
- `Create{Entity}` — insert, returns created record (`:one` with `RETURNING *`)
- `Update{Entity}` — generic update, returns updated record
- `Delete{Entity}` / `Restore{Entity}` / `HardDelete{Entity}` — soft delete, restore, permanent delete
- `{Verb}{Entity}` — explicit workflow operations (e.g., `FinalizePayrollPeriod`)
- `Count{Entities}...` — aggregate queries for usage guards
