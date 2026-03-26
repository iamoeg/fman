# Backlog

1. [ ] (High) PDF payslip export: `unidoc/unipdf` already in stack. Also consider CSV export (one row per employee)
       and a plain-text payslip matching the detail view layout. Trigger from the period list with a dedicated key.
2. [ ] (Medium) End-to-end tests: full flow: org → employee → payroll → finalize
3. [ ] (Low) Documentation polish: README + user-facing docs
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
8. [x] (Medium) Pay simulator for new hires: dedicated "Simulator" sidebar section.
       Form inputs: base salary, hire date, marital status (SINGLE/MARRIED/SEPARATED/DIVORCED/WIDOWED),
       dependents, children. Defaults to current month/year. Result mirrors the payroll detail view layout.
       No DB writes; calculator instantiated directly with no service dependency.
9. [x] (Medium) Filter in payroll section: filter payroll periods by year or status (DRAFT/FINALIZED).
       Press `/` to activate. `FilterValue` includes year, month name, status, and YYYY-MM so any
       of "2025", "January", "DRAFT", "FINALIZED", or "2025-01" narrows the list.
10. [x] (Low) Restore soft-deleted items: press `D` in any section to toggle the deleted view
        (title shows `[DELETED]`). Deleted items show their `deleted_at` date. `r` restores
        immediately; `x` hard-deletes with a permanent-warning confirm prompt. Implemented for
        orgs, employees, and compensation packages. Payroll periods/results deferred (complex
        restore semantics).
11. [ ] (High) Fix form vertical dimensions: forms overflow or misalign vertically in some terminal sizes.
12. [ ] (Medium) Unified design system: audit and standardize colors, borders,
        spacing, and status row styles across all sections.
13. [x] (Medium) Consistent keymaps: two-key back model. `backspace` = within-section back
        (`sectionBackKey`, never intercepted by root model — no conflict with drill-down states
        that return `IsOverlay() = false`). `esc` = return to sidebar from any list view (root
        model intercepts when `!IsOverlay()`). Forms keep `esc` = cancel (`IsOverlay() = true`
        so root never sees it). Confirm dialogs use `n` / `backspace` to dismiss.
14. [x] (High) Fix config file path via CLI argument: introduced Cobra as the CLI framework;
        `--config` flag wired to `config.LoadOrCreate`. Startup logic moved to `runTUI()`.
15. [ ] (Medium) Configure `golangci-lint`: add `.golangci.yml` and lint the codebase.
16. [ ] (Medium) Set up CI/CD pipeline: build, test, and lint on push.
17. [ ] (Low) Rename the project: decide on final name before publishing.
18. [x] (High) Constrain payroll calculations to supported years: moved all year-specific rates/brackets
        into a `yearRates` struct and a `ratesByYear` registry. `Calculate()` returns `ErrUnsupportedPayrollYear`
        when no entry exists for `period.Year`. Adding a new year = adding one map entry to the registry.
19. [x] (High) Relax hire date constraint: removed `MaxHireYearsInPast` constant and the past-date check
        from `ValidateHireDate()`. Now only rejects future hire dates. Past dates are valid; unsupported
        payroll years are caught by the calculator's `ErrUnsupportedPayrollYear` (see #18).
20. [ ] (High) Bonus/overtime input: `PayrollResult.TotalOtherBonus` is always 0 today.
        Add a per-employee per-period override (overlay on results list) before generating payroll.
        Requires storing overrides before calculation.
21. [x] (Medium) Empty states & onboarding hints: each section shows a contextual hint in
        the status row when its list is empty (org-dependency hints for comp/employees/payroll).
        Generating payroll with 0 employees now shows an error instead of "Generated 0 employee(s)".
22. [x] (Medium) Active org switching: press `o` from any pane to open a global overlay listing all
        active organizations. Navigate with `k`/`j`, confirm with `enter`, dismiss with `esc`. Active org
        is pre-selected and marked with `•`. Suppressed when a section overlay is already open.
        Reuses `setActiveOrgCmd` / `activeOrgLoadedMsg` — propagation to all sections unchanged.
23. [x] (Low) Payroll result: seniority details: added `SeniorityYears` and `SeniorityRate` to
        `PayrollResult` (domain + DB). Calculator populates them at generation time. Detail view
        now renders `Seniority Bonus (Xyr · Y%)` inline so the applied tier is always visible.
24. [x] (Low) Employee history view: `ListPayrollResultsByEmployee` exists in the service layer
        but is never called from the TUI. Add a history tab/overlay showing payslips across months.
25. [ ] (Medium) First-launch experience: when no config exists, guide the user through initial setup
        (e.g. display name, config path, default org). Avoids dropping a blank screen on first run.
26. [x] (Medium) Error message review: audited all user-facing errors for clarity and consistency.
        Load errors, delete failures, and form validation no longer expose raw internal strings.
        `userFriendly*` default cases return a generic message instead of `err.Error()`.
        `ErrPayrollCalculationFailed` unwraps to actionable hints (SMIG / unsupported year / generic).
        `"Invalid salary amount: %w"` replaced with a fixed user-facing string.
