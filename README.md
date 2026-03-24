# `finmgmt`

Internal financial management system for Moroccan companies,
focusing on payroll compliance and business operations.

## Project Status

**Current Phase:** Phase 1F ✅ Complete | Phase 1G ⏳ Next

- ✅ Foundation, database schema, TUI skeleton (1A)
- ✅ Domain models & tests (1B)
- ✅ SQLite repositories (1C)
- ✅ Application services (1D)
- ✅ Moroccan payroll engine — CNSS, AMO, IR (1E)
- ✅ Full TUI implementation — all CRUD sections (1F)
- ⏳ Domain hardening (1G)
- ⏳ Export (payslip PDF), end-to-end tests, documentation polish (1H)

## Overview

Multi-tenant financial management system with phased development:

- **Phase 1:** Payroll management (Moroccan compliance) - TUI
- **Phase 2:** Invoicing, budgeting, reporting
- **Phase 3:** Web UI + API + Multi-user support

### Why This Project?

This is an internal tool for a solo founder managing their company in Morocco.
The primary need is **accurate payroll calculation**
complying with Moroccan tax law (CNSS, AMO, IR).

### Why TUI First?

- Faster iteration during development
- No browser/network overhead
- Forces clean architecture that will make the web version easier later
- Can iterate by simply recompiling and running

## Core Features (Phase 1)

### Payroll Management (Priority)

- Monthly payroll generation for salaried employees
- Moroccan tax compliance:
  - **CNSS** (Social Security): Employee & employer contributions
  - **AMO** (Health Insurance): Mandatory coverage
  - **IR** (Income Tax): Progressive tax brackets
- PDF payslip generation
- Audit trail for compliance

### Organization & Employee Management

- Multi-organization support
- Employee records with Moroccan-specific fields (CIN, CNSS number)
- Compensation package tracking with history
- Soft-delete for data retention

### Future (Phase 2+)

- Expense tracking & recurring payments
- Invoicing
- Budgeting and forecasting
- Reporting and analytics
- Web UI with API backend
- Multi-user access with RBAC

## Technology Stack

- **Language:** Go
- **TUI Framework:** Bubble Tea (Elm-inspired architecture)
- **Database:** SQLite (local) → PostgreSQL (future web version)
- **Query Builder:** sqlc (type-safe SQL)
- **Migrations:** goose
- **PDF Generation:** unidoc/unipdf
- **CLI:** Cobra

## Architecture Principles

- **Hexagonal Architecture** (Go-idiomatic version)
- **Domain-driven design** - Business logic independent of technical details
- **Interface-agnostic** - TUI now, API later, same business logic
- **Money as integers** - Financial accuracy (no floats)
- **Audit everything** - Compliance and debugging

## Project Structure

```text
finmgmt/
├── cmd/
│   └── tui/              # TUI application entry point
├── db/
│   ├── migration/        # Database migrations
│   └── query/            # SQL queries for sqlc
├── internal/
│   ├── domain/           # Business entities
│   ├── application/      # Services (with inline interfaces)
│   └── adapter/          # Implementations (SQLite, PDF, etc.)
├── pkg/
│   └── money/            # Money type
├── ui/
│   └── tui/              # TUI interface
└── doc/                 # Documentation
```

## Quick Start

### Prerequisites

```bash
# Install Go 1.21+
# Install sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install goose
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### Setup

```bash
# Clone and setup
git clone <repo>
cd finmgmt
go mod download

# Run migrations
goose -dir db/migration sqlite3 ~/.local/share/finmgmt/data.db up

# Generate sqlc code
sqlc generate

# Run TUI
go run ./cmd/tui/
```

## Configuration

Configuration follows XDG Base Directory specification:

- **Config:** `~/.config/finmgmt/config.yaml`
- **Data:** `~/.local/share/finmgmt/data.db`

A custom config path can be passed at startup:

```bash
go run ./cmd/tui/ --config /path/to/config.yaml
```

## Documentation

- **[ARCHITECTURE.md](./doc/ARCHITECTURE.md)** - System architecture, patterns,
  and design decisions
- **[DATABASE.md](./doc/DATABASE.md)** - Complete database schema documentation
- **[IMPLEMENTATION.md](./doc/IMPLEMENTATION.md)** - Phase-by-phase implementation guide

## Development Principles

1. **Start simple, iterate fast** - MVP first, features later
2. **Test what matters** - Domain logic and critical paths
3. **Payroll is immutable** - Once finalized, it's a historical record
4. **Go idioms > Design patterns** - Write idiomatic Go, not Java in Go
5. **Be frank about mistakes** - Catch them early, fix them fast

## Compliance

This system is designed to comply with Moroccan tax and labor law:

- Social Security (CNSS) calculations
- Mandatory Health Insurance (AMO)
- Progressive Income Tax (IR)
- Proper audit trails for tax authorities

**Note:** Tax rates and regulations may change.
Always verify calculations against current Moroccan law.

## License

[To be determined]

## Contributing

This is currently a solo project.
Contributions are not being accepted at this time.
