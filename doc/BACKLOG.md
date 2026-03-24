# Backlog

## Phase 1H — Finish Line

1. [ ] (High) PDF payslip export: `unidoc/unipdf` already in stack. Also consider CSV export (one row per employee)
       and a plain-text payslip matching the detail view layout. Trigger from the period list with a dedicated key.
2. [ ] (Medium) End-to-end tests: full flow: org → employee → payroll → finalize
3. [ ] (Low) Documentation polish: README + user-facing docs

## Domain & Data Model

4. [x] (Medium) Currency tracking in `Money` struct: decided NOT to embed currency in `Money`.
       System is single-currency (MAD) by design; adding it complicates arithmetic with no real benefit.
       `Currency` type exists in for boundary validation. See `ARCHITECTURE.md` for full rationale.
5. [ ] (Medium) Clarify domain vocabulary (FR/EN mapping): document canonical English names
       for all FR legal terms (e.g. allocations familiales, ancienneté, IPE, AMO).
       Prevent naming drift between DB columns, domain structs, and TUI labels.
6. [x] (High) Clarify `NumKids` vs `NumDependents` in IR calculation: renamed `NumKids` → `NumChildren`;
       documented distinction (`NumDependents` = IR family charge deduction, `NumChildren` = CNSS family allowance);
       implemented missing family allowance calculation in the payroll engine.
7. [x] (Low) Enum table pattern for constrained text columns: replaced all inline `CHECK(col IN (...))`
       constraints with reference tables (`gender`, `marital_status`, `legal_form`, `currency`,
       `payroll_period_status`, `audit_action`). Each has a single `code TEXT PRIMARY KEY` and is
       seeded in the migration. Enum columns now carry bare FK references. Adding a new value = one
       INSERT migration, no schema edit.
18. [x] (High) Constrain payroll calculations to supported years: moved all year-specific rates/brackets
       into a `yearRates` struct and a `ratesByYear` registry. `Calculate()` returns `ErrUnsupportedPayrollYear`
       when no entry exists for `period.Year`. Adding a new year = adding one map entry to the registry.
19. [x] (High) Relax hire date constraint: removed `MaxHireYearsInPast` constant and the past-date check
       from `ValidateHireDate()`. Now only rejects future hire dates. Past dates are valid; unsupported
       payroll years are caught by the calculator's `ErrUnsupportedPayrollYear` (see #18).

## Features

8. [ ] (Medium) Pay simulator for new hires: given a base salary and employee profile,
       show estimated net pay, taxes, and contributions before creating the employee. Read-only, no DB writes.
9. [ ] (Medium) Filter in payroll section: filter payroll periods by year or status (DRAFT/FINALIZED).
10. [ ] (Low) Restore soft-deleted items: UI to list and restore soft-deleted orgs/employees.
20. [ ] (High) Bonus/overtime input: `PayrollResult.TotalOtherBonus` is always 0 today.
        Add a per-employee per-period override (overlay on results list) before generating payroll.
        Requires storing overrides before calculation.
25. [ ] (Medium) First-launch experience: when no config exists, guide the user through initial setup
        (e.g. display name, config path, default org). Avoids dropping a blank screen on first run.
23. [ ] (Low) Payroll result: seniority details: the seniority bonus rate (5–25%) is applied silently.
        Show calculated seniority years and applicable tier in the payroll detail view.
24. [ ] (Low) Employee history view: `ListPayrollResultsByEmployee` exists in the service layer
        but is never called from the TUI. Add a history tab/overlay showing payslips across months.

## TUI Polish

11. [ ] (High) Fix form vertical dimensions: forms overflow or misalign vertically in some terminal sizes.
12. [ ] (Medium) Unified design system: audit and standardize colors, borders,
        spacing, and status row styles across all sections.
13. [ ] (Medium) Consistent keymaps: pick one back key (`esc` or `backspace`) and apply uniformly. Audit all sections for divergence.
21. [ ] (Medium) Empty states & onboarding hints: show contextual hints when lists are empty
        (e.g. "Create a package before adding employees", "Add employees first" for payroll).
        Warn before generating payroll with 0 employees.
22. [ ] (Medium) Active org switching: switching active org requires navigating to Orgs section.
        Consider a dedicated switch-org prompt accessible from any section, or clearer visual indicator + help text.
26. [ ] (Medium) Error message review: audit all user-facing errors for clarity and consistency.
        Prefer Go-idiomatic lowercase, sentence-style messages. Avoid raw sentinel error strings leaking into the UI.

## Infrastructure & Tooling

14. [x] (High) Fix config file path via CLI argument: introduced Cobra as the CLI framework;
        `--config` flag wired to `config.LoadOrCreate`. Startup logic moved to `runTUI()`.
15. [ ] (Medium) Configure `golangci-lint`: add `.golangci.yml` and lint the codebase.
16. [ ] (Medium) Set up CI/CD pipeline: build, test, and lint on push.
17. [ ] (Low) Rename the project: decide on final name before publishing.
