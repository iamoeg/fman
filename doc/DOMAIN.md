# Domain Knowledge: Moroccan Payroll Context

This document defines the exact rules, rates, and calculation order used by
the Moroccan payroll engine (`internal/adapter/payroll/`).
All rates are verified against real payslips and reflect 2026 legislation.

> **Important:** When any rate or bracket changes, update this document first,
> then update the corresponding constants in the calculator. Never hardcode
> values directly in calculation logic.

---

## Table of Contents

1. [Calculation Order](#calculation-order)
2. [SMIG (Minimum Wage)](#smig-minimum-wage)
3. [Seniority Bonus](#seniority-bonus-prime-dancienneté)
4. [CNSS Contributions](#cnss-contributions)
5. [AMO Contributions](#amo-contributions)
6. [IR (Income Tax)](#ir-impôt-sur-le-revenu)
7. [CNSS Family Allowance (Allocations Familiales)](#cnss-family-allowance-allocations-familiales)
8. [Rounding](#rounding)
9. [Net to Pay Formula](#net-to-pay-formula)

---

## Calculation Order

The following order must be respected exactly. Each step depends on the result
of the previous one.

```text
1.  Base salary                        (from compensation package)
2.  Seniority bonus                    (% of base, tiered by years of service)
3.  Gross salary                       (base + seniority bonus)
4.  CNSS employee deductions           (applied to gross, IPE is capped)
5.  AMO employee deduction             (applied to gross, no ceiling)
6.  Professional expense deduction     (applied to gross, capped)
7.  Family charge deduction            (fixed per tax dependent, capped at 6) — IR input only
8.  Net taxable salary                 (gross − CNSS employee − AMO employee − professional expenses − family charges)
9.  IR (income tax)                    (progressive brackets applied to annual net taxable, divided by 12)
10. Family allowance                   (CNSS allocations familiales, based on num_children, tax-exempt)
11. Net to pay                         (gross − CNSS employee − AMO employee − IR + family allowance ± rounding)
12. Employer contributions             (calculated separately, do not affect net to pay)
```

---

## SMIG (Minimum Wage)

| Parameter    | Value        |
| ------------ | ------------ |
| Monthly SMIG | 3,422.00 MAD |

**Validation rule:** Base salary only must be ≥ SMIG. The seniority bonus
does not count toward the SMIG floor.

---

## Seniority Bonus (Prime d'Ancienneté)

The seniority bonus is a percentage of the **base salary** and is determined
by the employee's years of service (calculated from hire date).

| Seniority (years) | Rate |
| ----------------- | ---: |
| 0 – 2             |   0% |
| 2 – 5             |   5% |
| 5 – 12            |  10% |
| 12 – 20           |  15% |
| 20 – 25           |  20% |
| 25+               |  25% |

**Bracket logic:** The lower bound is inclusive, the upper bound is exclusive.
An employee with exactly 5 years of service falls in the 5–12 bracket (10%).

**Formula:**

```text
seniority_bonus = base_salary × rate
gross_salary    = base_salary + seniority_bonus
```

---

## CNSS Contributions

CNSS (Caisse Nationale de Sécurité Sociale) is composed of four components.
Each component has its own rate and ceiling rules.

### 1. Prestations Familiales

|                 | Employee | Employer |
| --------------- | -------: | -------: |
| Rate            |       0% |    6.40% |
| Monthly ceiling |     None |     None |

No employee contribution. Applied to full gross salary.

> **Note:** "Allocations Familiales" and "Prestations Familiales" refer to the
> same CNSS component. The payslip label is "ER - Prestations Familiales".
> There is no separate Social Allowance line — do not add one.

### 2. Prestations Sociales

|                 |     Employee |     Employer |
| --------------- | -----------: | -----------: |
| Rate            |        4.48% |        8.98% |
| Monthly ceiling | 6,000.00 MAD | 6,000.00 MAD |

Applied to `min(gross_salary, 6,000 MAD)`.

### 3. Job Loss Compensation (IPE — Indemnité pour Perte d'Emploi)

|                 |     Employee |     Employer |
| --------------- | -----------: | -----------: |
| Rate            |        0.19% |        0.38% |
| Monthly ceiling | 6,000.00 MAD | 6,000.00 MAD |

Applied to `min(gross_salary, 6,000 MAD)`.

### 4. Training Tax (Taxe de Formation Professionnelle)

|                 | Employee | Employer |
| --------------- | -------: | -------: |
| Rate            |       0% |    1.60% |
| Monthly ceiling |     None |     None |

No employee contribution. Applied to full gross salary.

### Mapping to PayrollResult Fields

The `PayrollResult` domain model uses different naming than the official CNSS
labels. This table is the authoritative mapping — do not add new fields.

| PayrollResult Field                  | CNSS Component                                                                                                                           |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `SocialAllowanceEmployeeContrib`     | Prestations Sociales (employee 4.48%)                                                                                                    |
| `SocialAllowanceEmployerContrib`     | Prestations Sociales (employer 8.98%)                                                                                                    |
| `JobLossCompensationEmployeeContrib` | IPE (employee 0.19%)                                                                                                                     |
| `JobLossCompensationEmployerContrib` | IPE (employer 0.38%)                                                                                                                     |
| `TrainingTaxEmployerContrib`         | Taxe de Formation Professionnelle (1.60%)                                                                                                |
| `FamilyBenefitsEmployerContrib`      | Prestations Familiales (employer 6.40%)                                                                                                  |
| `TotalCNSSEmployeeContrib`           | `SocialAllowanceEmployeeContrib` + `JobLossCompensationEmployeeContrib`                                                                  |
| `TotalCNSSEmployerContrib`           | `SocialAllowanceEmployerContrib` + `JobLossCompensationEmployerContrib` + `TrainingTaxEmployerContrib` + `FamilyBenefitsEmployerContrib` |

| Component                 | Employee Rate | Employer Rate | Ceiling         |
| ------------------------- | ------------: | ------------: | --------------- |
| Prestations Familiales    |            0% |         6.40% | None            |
| Prestations Sociales      |         4.48% |         8.98% | 6,000 MAD/month |
| IPE                       |         0.19% |         0.38% | 6,000 MAD/month |
| Training Tax              |            0% |         1.60% | None            |
| **Total (uncapped base)** |     **4.67%** |    **17.36%** | —               |

**Employee CNSS formula:**

```text
cnss_capped_base        = min(gross_salary, 6_000 MAD)
cnss_employee_contrib   = (cnss_capped_base × 4.48%)   // Prestations Sociales
                        + (cnss_capped_base × 0.19%)   // IPE
```

**Employer CNSS formula:**

```text
cnss_capped_base                    = min(gross_salary, 6_000 MAD)
prestations_familiales_contrib      = gross_salary × 6.40%
prestations_sociales_contrib        = cnss_capped_base × 8.98%
ipe_employer_contrib                = cnss_capped_base × 0.38%
training_tax_employer_contrib       = gross_salary × 1.60%
total_cnss_employer_contrib         = sum of all four above
```

---

## AMO Contributions

AMO (Assurance Maladie Obligatoire) is mandatory health insurance, collected
separately from CNSS but remitted together in practice.

|                 | Employee | Employer |
| --------------- | -------: | -------: |
| Rate            |    2.26% |    4.11% |
| Monthly ceiling |     None |     None |

Applied to full gross salary.

```text
amo_employee_contrib = gross_salary × 2.26%
amo_employer_contrib = gross_salary × 4.11%
```

---

## IR (Impôt sur le Revenu)

### Step 1 — Professional Expense Deduction

Two rates apply depending on annual gross salary:

| Condition                 |         Rate |     Monthly Cap |
| ------------------------- | -----------: | --------------: |
| Annual gross > 78,000 MAD | 20% of gross | 2,500 MAD/month |
| Annual gross ≤ 78,000 MAD | 35% of gross | 2,500 MAD/month |

The rate is evaluated **every month** using `gross_salary × 12` as a proxy
for annual gross. This means an employee whose gross changes mid-year may
switch rate brackets from one month to the next — this is intentional and
matches payslip behaviour.

```text
annual_gross                = gross_salary × 12
if annual_gross <= 78_000:
    rate = 35%
else:
    rate = 20%

professional_expense_deduction = min(gross_salary × rate, 2_500 MAD)
```

> **Open question:** Verify against a payslip for an employee with annual gross
> close to 78,000 MAD (i.e. monthly gross ~6,500 MAD) that the rate switches
> correctly at the boundary.

### Step 2 — Family Charge Deduction (IR input only)

This deduction reduces the **IR taxable base** only. It does not appear as a
line in the net-to-pay calculation. It is based on `num_dependents` (spouse +
qualifying children), not `num_children`.

| Parameter               | Value           |
| ----------------------- | --------------- |
| Deduction per dependent | 40.00 MAD/month |
| Maximum dependents      | 6               |

```text
capped_dependents       = min(num_dependents, 6)
family_charge_deduction = capped_dependents × 40 MAD
```

### Step 3 — Net Taxable Salary

```text
net_taxable_salary = gross_salary
                   − cnss_employee_contrib
                   − amo_employee_contrib
                   − professional_expense_deduction
                   − family_charge_deduction
```

### Step 4 — IR Brackets (2026)

IR is calculated on the **annualised** net taxable salary, then divided by 12
for the monthly amount.

```text
annual_taxable = net_taxable_salary × 12
```

| Annual Taxable Income | Rate | Fixed Deduction |
| --------------------- | ---: | --------------: |
| 0 – 40,000 MAD        |   0% |           0 MAD |
| 40,001 – 60,000 MAD   |  10% |       4,000 MAD |
| 60,001 – 80,000 MAD   |  20% |      10,000 MAD |
| 80,001 – 100,000 MAD  |  30% |      18,000 MAD |
| 100,001 – 180,000 MAD |  34% |      22,000 MAD |
| 180,001+ MAD          |  37% |      27,400 MAD |

**Formula:**

```text
annual_ir   = (annual_taxable × bracket_rate) − fixed_deduction
monthly_ir  = annual_ir / 12
```

If `annual_taxable` falls in the 0% bracket, IR is 0.

---

## CNSS Family Allowance (Allocations Familiales)

Allocations familiales are a **cash allowance paid directly to the employee**
based on the number of qualifying children (`num_children`). They are funded
by the employer's CNSS family benefits contribution
but appear as a separate income line on the payslip.

- **Tax-exempt** — does not affect the IR base
- **Not subject to CNSS social contributions** — does not affect the CNSS base
- **Increases net to pay directly**

| Children | Rate per child   |
| -------- | ---------------- |
| 1 – 3    | 300.00 MAD/month |
| 4 – 6    | 36.00 MAD/month  |
| > 6      | capped at 6      |

```text
low_tier  = min(num_children, 3)
high_tier = max(min(num_children, 6) − 3, 0)

family_allowance = (low_tier × 300) + (high_tier × 36)
```

**Important distinction:**

- `num_dependents` (spouse + children) → used for the IR family charge **deduction**
- `num_children` (qualifying children only) → used for the CNSS allocations **familiales**

---

## Rounding

| Rule                | Detail                                       |
| ------------------- | -------------------------------------------- |
| Target              | Net to pay only                              |
| Method              | Round to nearest dirham (nearest 100 cents)  |
| Intermediate values | Not rounded — full precision kept throughout |

```text
rounding_amount = round_to_nearest_dirham(net_to_pay_before_rounding)
                  − net_to_pay_before_rounding

net_to_pay      = net_to_pay_before_rounding + rounding_amount
```

The `rounding_amount` field in `PayrollResult` stores the adjustment applied
(can be negative, zero, or positive). Its absolute value should never exceed
50 cents (1 dirham / 2).

---

## Net to Pay Formula

```text
net_to_pay_before_rounding = gross_salary
                           − cnss_employee_contrib
                           − amo_employee_contrib
                           − monthly_ir
                           + family_allowance

net_to_pay = round_to_nearest_dirham(net_to_pay_before_rounding)
```

Note that professional expense deduction and family charge deduction affect
the IR base but do **not** appear as deduction lines in the net-to-pay formula.
They are IR inputs only. The family allowance, by contrast, is **added** to
net pay — it is tax-exempt income funded by the CNSS pool.

---

## Worked Example 1

Given:

- Base salary: 10,000 MAD
- Hire date: 6 years ago (seniority bracket: 5–12 → 10%)
- Dependents: 2
- Annual gross: 110,000 MAD (> 78,000 → professional expense rate: 20%)

```text
base_salary                     = 10,000.00 MAD
seniority_bonus                 = 10,000 × 10%           =  1,000.00 MAD
gross_salary                    = 10,000 + 1,000          = 11,000.00 MAD

cnss_capped_base                = min(11,000, 6,000)      =  6,000.00 MAD
cnss_employee_contrib           = 6,000 × (4.48% + 0.19%) =    280.20 MAD

amo_employee_contrib            = 11,000 × 2.26%          =    248.60 MAD

professional_expense_deduction  = min(11,000 × 20%, 2,500)=  2,200.00 MAD
family_charge_deduction         = min(2, 6) × 40          =     80.00 MAD

net_taxable_salary              = 11,000 − 280.20 − 248.60
                                  − 2,200 − 80             =  8,191.20 MAD

annual_taxable                  = 8,191.20 × 12            = 98,294.40 MAD
  → bracket: 80,001–100,000 → rate 30%, deduction 18,000 MAD
annual_ir                       = (98,294.40 × 30%) − 18,000 =  11,488.32 MAD
monthly_ir                      = 11,488.32 / 12           =    957.36 MAD

family_allowance                = 0 children → 0 MAD

net_to_pay_before_rounding      = 11,000 − 280.20 − 248.60
                                  − 957.36 + 0             =  9,513.84 MAD
net_to_pay (rounded)            = 9,514.00 MAD
rounding_amount                 = +0.16 MAD (= +16 cents)

--- Employer side ---
prestations_familiales          = 11,000 × 6.40%           =    704.00 MAD
prestations_sociales_employer   = 6,000 × 8.98%            =    538.80 MAD
ipe_employer                    = 6,000 × 0.38%            =     22.80 MAD
training_tax_employer           = 11,000 × 1.60%           =    176.00 MAD
total_cnss_employer             = 704 + 538.80 + 22.80 + 176 = 1,441.60 MAD
amo_employer                    = 11,000 × 4.11%           =    452.10 MAD
```

---

## Worked Example 2

Given:

- Base salary: 20,000 MAD
- Seniority: 3 years (bracket: 2–5 → 5%)
- Dependents: 0
- Annual gross: 252,000 MAD (> 78,000 → professional expense rate: 20%)

```text
base_salary                     = 20,000.00 MAD
seniority_bonus                 = 20,000 × 5%            =  1,000.00 MAD
gross_salary                    = 20,000 + 1,000          = 21,000.00 MAD

cnss_capped_base                = min(21,000, 6,000)      =  6,000.00 MAD
cnss_employee_contrib           = 6,000 × (4.48% + 0.19%) =    280.20 MAD

amo_employee_contrib            = 21,000 × 2.26%          =    474.60 MAD

professional_expense_deduction  = min(21,000 × 20%, 2,500)=  2,500.00 MAD  ← cap reached
family_charge_deduction         = 0 × 40                  =      0.00 MAD

net_taxable_salary              = 21,000 − 280.20 − 474.60
                                  − 2,500 − 0             = 17,745.20 MAD

annual_taxable                  = 17,745.20 × 12          = 212,942.40 MAD
  → bracket: 180,001+ → rate 37%, deduction 27,400 MAD
annual_ir                       = (212,942.40 × 37%) − 27,400 = 51,388.69 MAD
monthly_ir                      = 51,388.69 / 12          =  4,282.39 MAD

family_allowance                = 0 children → 0 MAD

net_to_pay_before_rounding      = 21,000 − 280.20 − 474.60
                                  − 4,282.39 + 0          = 15,962.81 MAD
net_to_pay (rounded)            = 15,963.00 MAD
rounding_amount                 = +0.19 MAD (= +19 cents)

--- Employer side ---
prestations_familiales          = 21,000 × 6.40%          =  1,344.00 MAD
prestations_sociales_employer   = 6,000 × 8.98%           =    538.80 MAD
ipe_employer                    = 6,000 × 0.38%           =     22.80 MAD
training_tax_employer           = 21,000 × 1.60%          =    336.00 MAD
total_cnss_employer             = 1,344 + 538.80 + 22.80
                                  + 336                   =  2,241.60 MAD
amo_employer                    = 21,000 × 4.11%          =    863.10 MAD
```
