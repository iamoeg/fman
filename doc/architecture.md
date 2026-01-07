# Financial Management Application - Architecture & Planning Document

**Project Type:** Internal Financial Management System
**Primary User:** Solo founder/manager (multi-user capability planned for future)
**Compliance:** Moroccan law (payroll, tax)

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Requirements](#requirements)
3. [Architecture](#architecture)
4. [Technology Stack](#technology-stack)
5. [Implementation Phases](#implementation-phases)
6. [Key Design Decisions](#key-design-decisions)
7. [Next Steps](#next-steps)

---

## Project Overview

Multi-tenant financial management system with phased development:

- **Phase 1:** Payroll management (Moroccan compliance) + Expense tracking (TUI)
- **Phase 2:** Invoicing, budgeting, reporting
- **Phase 3:** Web UI + API + Multi-user support

**Development Philosophy:**

- TUI-first for rapid iteration
- Interface-agnostic business logic
- Modular architecture for easy extension

---

## Requirements

### Core Functions (Phase 1)

1. **Organization Management**
   - Create and manage multiple organizations/companies
   - Single user initially, multi-tenant architecture from the start

2. **Employee Management**
   - CRUD operations for employees
   - Compensation configuration
   - Association with organizations

3. **Payroll Processing**
   - Monthly payroll generation
   - Moroccan tax compliance (CNSS, AMO, IR)
   - Payslip generation (PDF)
   - Salaried employees only (no contractors initially)

4. **Expense Tracking**
   - One-time expense recording
   - Recurring payment management
   - In-app notifications/reminders (TUI)
   - No email/SMS until web version

5. **Data Export**
   - JSON export
   - XML export

### Future Functions (Phase 2+)

- Invoicing
- Budgeting
- Reporting and analytics
- Multi-user access with RBAC
- Web UI with API backend
- Email/SMS notifications

### Non-Functional Requirements

- **Auditability:** Track state changes (who, what, when, before/after)
- **Compliance:** Moroccan payroll and tax law
- **Deployment:** Local binary (TUI), containerized (future web)
- **Data Export:** JSON and XML formats

---

## Architecture

### High-Level Structure

```text
financial-management/
├── cmd/
│   ├── tui/              # TUI application entry point
│   └── api/              # Future: API server
├── internal/
│   ├── domain/           # Pure business entities
│   │   ├── organization.go
│   │   ├── employee.go
│   │   ├── payroll.go
│   │   └── expense.go
│   ├── application/      # Business logic / Use cases
│   │   ├── organization_service.go
│   │   ├── payroll_service.go
│   │   └── expense_service.go
│   ├── infrastructure/   # External dependencies
│   │   ├── database/     # Repository implementations
│   │   ├── payroll/      # Moroccan calculation engine
│   │   └── export/       # JSON/XML exporters
│   └── ports/            # Interface definitions
│       ├── repositories.go
│       └── services.go
├── pkg/
│   ├── money/            # Money type (avoid float precision issues)
│   └── notifications/    # In-memory notification queue
└── ui/
    └── tui/              # TUI-specific code
        ├── app.go        # Main TUI app
        ├── models/       # TUI state models
        └── views/        # Screen components
```

### Architectural Pattern: Hexagonal Architecture (Ports & Adapters)

**Core Principles:**

1. **Domain Layer:** Pure business logic, no external dependencies
2. **Application Layer:** Use cases orchestrate domain objects
3. **Ports:** Interfaces defining how to interact with external systems
4. **Adapters:** Concrete implementations (database, UI, etc.)

**Benefits:**

- Business logic is completely independent of UI/database
- Easy to test
- Can swap TUI for API without changing business logic
- Clear separation of concerns

### Key Interfaces

```go
// Repositories (Ports)
type OrganizationRepository interface {
    Create(ctx context.Context, org *domain.Organization) error
    FindByID(ctx context.Context, id string) (*domain.Organization, error)
    List(ctx context.Context) ([]*domain.Organization, error)
    Update(ctx context.Context, org *domain.Organization) error
}

// Services (Application Layer)
type PayrollService struct {
    empRepo      ports.EmployeeRepository
    payrollRepo  ports.PayrollRepository
    calculator   PayrollCalculator
    pdfGenerator PDFGenerator
    notifier     Notifier
}
```

---

## Technology Stack

### TUI Version (Phase 1)

- **TUI Framework:** `bubbletea` (Elm-inspired, excellent state management)
- **TUI Components:** `bubbles` (pre-built tables, lists, inputs)
- **Styling:** `lipgloss` (terminal styling)
- **Database:** SQLite (local file, simple for single-user)
- **PDF Generation:** `unidoc/unipdf`
- **Migrations:** `golang-migrate` or `goose`

### Future Web Version (Phase 2+)

- **Web Framework:** `chi` or `echo`
- **Database:** PostgreSQL (migration from SQLite)
- **Authentication:** JWT tokens
- **Containerization:** Docker

### Rationale

- **Bubble Tea:** Clean architecture, handles complex TUI state well
- **SQLite → PostgreSQL:** Start simple, migrate when needed (same repository interfaces)
- **Hexagonal Architecture:** Makes TUI → Web migration seamless

---

## Implementation Phases

### Phase 1A - TUI Foundation (Week 1)

**Goals:**

- Project structure setup
- SQLite database with migrations
- Basic Bubble Tea app skeleton
- Main menu and navigation

**Deliverables:**

- Runnable TUI with navigation
- Database connection and migration system
- Basic screen routing

---

### Phase 1B - Organization & Employee Management (Week 2)

**Goals:**

- Organization CRUD operations
- Employee CRUD operations
- Data validation
- Basic audit logging

**Deliverables:**

- TUI screens for org/employee management
- Working database persistence
- Audit trail for changes

---

### Phase 1C - Payroll Engine (Weeks 3-4)

**Goals:**

- Moroccan payroll calculator
- Monthly payroll period generation
- Payslip PDF generation
- Payroll TUI screens

**Deliverables:**

- Accurate Moroccan tax calculations (CNSS, AMO, IR)
- PDF payslips
- Payroll generation workflow in TUI

---

### Phase 1D - Expense Tracking (Week 5)

**Goals:**

- Expense CRUD
- Recurring payment management
- In-app notification system
- Reminder generation

**Deliverables:**

- Expense tracking screens
- Notification queue and display
- Recurring payment scheduler

---

### Phase 1E - Export & Polish (Week 6)

**Goals:**

- JSON/XML export functionality
- Data backup/restore
- Error handling improvements
- Documentation

**Deliverables:**

- Complete TUI application
- Export functionality
- User documentation

---

## Key Design Decisions

### 1. Money Type

**Decision:** Create a custom `Money` type  
**Rationale:** Avoid floating-point precision errors in financial calculations  
**Implementation:** Store as smallest unit (e.g., cents) internally

### 2. Moroccan Payroll Calculator

**Decision:** Standalone, pluggable calculator  
**Rationale:**

- Encapsulates complex tax logic
- Easy to update when laws change
- Testable in isolation

**Key Calculations:**

- **CNSS:** Employee 4.48%, Employer 16.01%
- **AMO:** Employee 2.26%, Employer 2.26%
- **IR:** Progressive brackets (0%, 10%, 20%, 30%, 34%, 38%)
- **Professional expenses:** Standard deduction
- **Family allowances:** If applicable

### 3. Database Choice

**Decision:** SQLite for TUI, PostgreSQL for web  
**Rationale:**

- SQLite: Perfect for local, single-user
- Same repository interfaces enable easy migration
- PostgreSQL: Better for multi-user, JSONB support

### 4. Audit Trail

**Decision:** Simple event log table initially  
**Rationale:**

- Lightweight for Phase 1
- Can evolve to full event sourcing if needed
- Captures: who, what, when, before/after state

### 5. Notification System

**Decision:** In-memory queue for TUI  
**Rationale:**

- No need for persistence initially
- Simple channel-based system
- Can upgrade to database-backed for web version

---

## Next Steps

### Immediate Actions (Phase 1A)

1. **Initialize Go module**
   - Set up project structure
   - Add initial dependencies
2. **Database Setup**
   - Create SQLite schema
   - Set up migration system
   - Design initial tables (organizations, employees)

3. **Bubble Tea Skeleton**
   - Create main TUI app
   - Implement basic navigation
   - Set up state management

### Things to Consider

- **Error handling strategy:** How will you handle and display errors in the TUI?
- **Configuration:** Where will you store app config (DB path, etc.)?
- **Testing strategy:** Unit tests for business logic, integration tests for repositories?
- **Data location:** Where should the SQLite file live? (`~/.finmgmt/data.db`?)

---

## Open Questions

- [ ] Should audit logs be queryable through TUI, or just for compliance?
- [ ] Do you need to support multiple currencies, or just MAD (Moroccan Dirham)?
- [ ] How should data backups work? Manual export or automatic?
- [ ] What level of data validation is needed? (e.g., employee ID format, salary ranges)

---

## Resources

- **Bubble Tea Docs:** <https://github.com/charmbracelet/bubbletea>
- **Moroccan Tax Reference:** [To be added]
- **Go Project Layout:** <https://github.com/golang-standards/project-layout>

---

## Document History

- **2026-01-07:** Initial architecture and planning document created
