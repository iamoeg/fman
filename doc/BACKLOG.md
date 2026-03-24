# Backlog

## Execution Order

Architectural changes first to minimize rework, then features on a stable domain,
then polish and tooling.

1. ~**#4** — Decide Money/currency question. Most cross-cutting change;
   closes it early even if the answer is "no".~
2. ~**#6** — Clarify NumKids vs NumDependents. Affects calculator correctness and potentially DB schema.~
3. **#18 + #19** — Constrain calculations to supported year, relax hire date.
   Do together; #19 is safe only once #18 is in place.
4. **#1** — PDF payslip export. Builds on top of a now-stable domain.
5. **#14, #13, #11** — Easy wins: CLI config flag, consistent keymaps, form dimensions.
6. **#2** — End-to-end tests. Written last, once behavior is locked.

---

## Phase 1G — Finish Line

1. [ ] (High) PDF payslip export: `unidoc/unipdf` already in stack
2. [ ] (Medium) End-to-end tests: full flow: org → employee → payroll → finalize
3. [ ] (Low) Documentation polish: README + user-facing docs

## Domain & Data Model

4. [x] (Medium) Currency tracking in `Money` struct: decided NOT to embed currency in `Money`.
       System is single-currency (MAD) by design; adding it complicates arithmetic with no real benefit.
       `Currency` type exists in for boundary validation. See `ARCHITECTURE.md` for full rationale.
5. [ ] (Medium) Clarify domain vocabulary (FR/EN mapping): document canonical English names
       for all FR legal terms (e.g. allocations familiales, ancienneté, IPE, AMO).
       Prevent naming drift between DB columns, domain structs, and TUI labels.
6. [x] (High) Clarify `NumKids` vs `NumDependents` in IR calculation: renamed `NumKids` → `NumChildren`; documented distinction (`NumDependents` = IR family charge deduction, `NumChildren` = CNSS allocations familiales); implemented missing allocations familiales calculation in the payroll engine. See DOMAIN.md §CNSS Allocations Familiales.
7. [ ] (Low) Enum table pattern for constrained text columns: replace inline `CHECK(col IN (...))`
       constraints in schema (gender, marital_status, legal_form, status) with reference tables.
       Improves referential integrity and makes adding new values a migration rather than a schema edit + code change.
8. [ ] (High) Constrain payroll calculations to supported years: the calculator silently applies current-year tax tables to any period year.
       Must validate that the payroll period year matches the supported year (2025 for now)
       and return a clear error otherwise. Relax when multi-year engine support is added.
9. [ ] (High) Relax hire date constraint (`MaxHireYearsInPast`): currently `MaxHireYearsInPast = 1`,
       blocking employees hired more than a year ago. This was tied to payroll calculation limits,
       but since we're constraining calculations to the current year (see #18),
       the hire date can safely be any date in the past. Remove or greatly increase this limit.

## Features

8. [ ] (Medium) Pay simulator for new hires: given a base salary and employee profile,
       show estimated net pay, taxes, and contributions before creating the employee. Read-only, no DB writes.
9. [ ] (Medium) Filter in payroll section: filter payroll periods by year or status (DRAFT/FINALIZED).
10. [ ] (Low) Restore soft-deleted items: UI to list and restore soft-deleted orgs/employees.

## TUI Polish

11. [ ] (High) Fix form vertical dimensions: forms overflow or misalign vertically in some terminal sizes.
12. [ ] (Medium) Unified design system: audit and standardize colors, borders,
        spacing, and status row styles across all sections.
13. [ ] (Medium) Consistent keymaps: pick one back key (`esc` or `backspace`) and apply uniformly. Audit all sections for divergence.

## Infrastructure & Tooling

14. [ ] (High) Fix config file path via CLI argument: `main.go` calls `config.LoadOrCreate("")`,
        ignoring any CLI-provided path. Wire up a `--config` flag.
15. [ ] (Medium) Configure `golangci-lint`: add `.golangci.yml` and lint the codebase.
16. [ ] (Medium) Set up CI/CD pipeline: build, test, and lint on push.
17. [ ] (Low) Rename the project: decide on final name before publishing.
