# `fman`

Internal financial management system for Moroccan companies,
focusing on payroll compliance and business operations.
Built for my own use running a small company in Morocco,
with accurate payroll calculation (CNSS, AMO, IR) as the core priority.

## Core Features

### Payroll Management

- Monthly payroll generation for salaried employees
- Moroccan tax compliance:
  - **CNSS** (Social Security): Employee & employer contributions
  - **AMO** (Health Insurance): Mandatory coverage
  - **IR** (Income Tax): Progressive tax brackets
- Audit trail for compliance

### Organization & Employee Management

- Multi-organization support
- Employee records with Moroccan-specific fields (CIN, CNSS number)
- Compensation package tracking with history
- Soft-delete for data retention

### Planned

- PDF payslip generation
- Expense tracking & recurring payments
- Invoicing
- Budgeting and forecasting
- Reporting and analytics
- Web UI with API backend
- Multi-user access with RBAC

## Technology Stack

- **Language:** Go
- **TUI Framework:** Bubble Tea
- **Database:** SQLite
- **Query Builder:** `sqlc`
- **Migrations:** `goose`
- **CLI:** Cobra

## Quick Start

### Binary (recommended)

Download the latest release for your platform from the
[releases page](https://github.com/iamoeg/fman/releases), put the binary on your
`$PATH`, and run:

```bash
fman
```

The database and config are created automatically on first run.

### From source

Requires [mise](https://mise.jdx.dev) to manage Go and dev tool versions.

```bash
git clone https://github.com/iamoeg/fman
cd fman
mise install
mise run run-tui
```

Common tasks:

```bash
mise run build      # build binary to bin/fman
mise run test       # run tests
mise run lint       # run golangci-lint
mise run ci         # full CI check suite (fmt, lint, test)
```

## Configuration

Configuration follows XDG Base Directory specification:

- **Config:** `~/.config/fman/config.yaml`
- **Data:** `~/.local/share/fman/data.db`

A custom config path can be passed at startup:

```bash
fman --config /path/to/config.yaml
```

## Documentation

- **[ARCHITECTURE.md](./doc/ARCHITECTURE.md)** — System architecture, patterns, and design decisions
- **[DATABASE.md](./doc/DATABASE.md)** — Complete database schema documentation
- **[IMPLEMENTATION.md](./doc/IMPLEMENTATION.md)** — Phase-by-phase implementation guide

## License

[To be determined]

## Contributing

This is currently a solo project.
Contributions are not being accepted at this time.
