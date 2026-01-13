# Domain Knowledge: Moroccan Payroll Context

## CNSS (Caisse Nationale de Sécurité Sociale)

**Social Security System** - Mandatory contributions

**Components:**

1. **Social Allowance** (Allocations Familiales)
   - Employee: TBD%
   - Employer: TBD%

2. **Job Loss Compensation** (IPE - Indemnité pour Perte d'Emploi)
   - Employee: TBD%
   - Employer: TBD%

3. **Training Tax** (Taxe de Formation Professionnelle)
   - Employer only: TBD%

4. **Family Benefits**
   - Employer only: TBD%

**Total CNSS:**

- Employee: TBD%
- Employer: TBD%

## AMO (Assurance Maladie Obligatoire)

**Mandatory Health Insurance** - Separate from CNSS

**Contributions:**

- Employee: TBD%
- Employer: TBD%

## IR (Impôt sur le Revenu)

**Progressive Income Tax** - Applied to net taxable salary

**Annual Brackets (2024/2025)** - Divide by 12 for monthly:

```text
0 - 30,000 MAD:        0%   (Tax: 0 MAD)
30,001 - 50,000 MAD:   10%  (Tax: 2,000 MAD on this bracket)
50,001 - 60,000 MAD:   20%  (Tax: 2,000 MAD on this bracket)
60,001 - 80,000 MAD:   30%  (Tax: 6,000 MAD on this bracket)
80,001 - 180,000 MAD:  34%  (Tax: 34,000 MAD on this bracket)
180,001+ MAD:          38%  (Tax: 38% of amount over 180,000)
```

**Calculation Base:**

1. Start with gross salary
2. Subtract employee CNSS contributions
3. Subtract employee AMO contributions
4. Subtract professional expense deduction (20% of gross, capped)
5. Subtract family charge deductions (if applicable)
6. Apply progressive brackets to result

### Important Notes

**Rates may change:** These are reference rates.
Always verify against current Moroccan law before implementing.

**Professional Expenses:** Standard deduction of 20% of gross salary,
subject to caps.

**Family Allowances:** Based on number of dependents and children.

**Rounding:** Moroccan practice typically rounds to nearest dirham (100 cents).
