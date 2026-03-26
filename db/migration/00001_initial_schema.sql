-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys = ON;

-- Reference tables (enum enforcement via FK instead of inline CHECK)
CREATE TABLE legal_form(code TEXT PRIMARY KEY);
INSERT INTO legal_form(code) VALUES ('SARL');

CREATE TABLE currency(code TEXT PRIMARY KEY);
INSERT INTO currency(code) VALUES ('MAD');

CREATE TABLE gender(code TEXT PRIMARY KEY);
INSERT INTO gender(code) VALUES ('MALE'), ('FEMALE');

CREATE TABLE marital_status(code TEXT PRIMARY KEY);
INSERT INTO marital_status(code) VALUES ('SINGLE'), ('MARRIED'), ('SEPARATED'), ('DIVORCED'), ('WIDOWED');

CREATE TABLE payroll_period_status(code TEXT PRIMARY KEY);
INSERT INTO payroll_period_status(code) VALUES ('DRAFT'), ('FINALIZED');

CREATE TABLE audit_action(code TEXT PRIMARY KEY);
INSERT INTO audit_action(code) VALUES ('CREATE'), ('UPDATE'), ('DELETE'), ('RESTORE'), ('HARD_DELETE');

CREATE TABLE organization(
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT,
    activity TEXT,
    legal_form TEXT REFERENCES legal_form(code),
    ice_num TEXT UNIQUE,
    if_num TEXT UNIQUE,
    rc_num TEXT UNIQUE,
    cnss_num TEXT UNIQUE,
    bank_rib TEXT UNIQUE,

    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT
);

CREATE TABLE employee_compensation_package(
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'MAD' REFERENCES currency(code),
    base_salary_cents INTEGER NOT NULL CHECK(base_salary_cents >= 0),

    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT
);

CREATE INDEX idx_comp_package_org_id ON employee_compensation_package(org_id);

CREATE TABLE employee(
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    serial_num INTEGER NOT NULL CHECK(serial_num >= 1),
    full_name TEXT NOT NULL,
    display_name TEXT,
    address TEXT,
    email_address TEXT,
    phone_number TEXT,
    birth_date TEXT NOT NULL,
    gender TEXT NOT NULL REFERENCES gender(code),
    marital_status TEXT NOT NULL REFERENCES marital_status(code),
    num_dependents INTEGER NOT NULL DEFAULT 0 CHECK(num_dependents >= 0),
    num_children INTEGER NOT NULL DEFAULT 0 CHECK(num_children >= 0),
    cin_num TEXT NOT NULL UNIQUE,
    cnss_num TEXT UNIQUE,
    hire_date TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    position TEXT NOT NULL,
    compensation_package_id TEXT NOT NULL REFERENCES employee_compensation_package(id) ON DELETE RESTRICT,
    bank_rib TEXT,

    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT,

    UNIQUE(org_id, serial_num)
);

CREATE INDEX idx_employee_org_id ON employee(org_id);

CREATE TABLE payroll_period(
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    year INTEGER NOT NULL CHECK(year BETWEEN 2020 AND 2050),
    month INTEGER NOT NULL CHECK(month BETWEEN 1 AND 12),
    status TEXT NOT NULL DEFAULT 'DRAFT' REFERENCES payroll_period_status(code),
    finalized_at TEXT,

    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT,

    UNIQUE(org_id, year, month),
    CHECK(
        (status = 'DRAFT' AND finalized_at IS NULL)
        OR
        (status = 'FINALIZED' AND finalized_at IS NOT NULL)
    )
);

CREATE INDEX idx_payroll_period_org_id ON payroll_period(org_id);

CREATE TABLE payroll_result(
    id TEXT PRIMARY KEY,
    payroll_period_id TEXT NOT NULL REFERENCES payroll_period(id) ON DELETE CASCADE,
    employee_id TEXT NOT NULL REFERENCES employee(id) ON DELETE CASCADE,
    compensation_package_id TEXT NOT NULL REFERENCES employee_compensation_package(id) ON DELETE RESTRICT,
    currency TEXT NOT NULL DEFAULT 'MAD' REFERENCES currency(code),
    base_salary_cents INTEGER NOT NULL CHECK(base_salary_cents >= 0),
    seniority_bonus_cents INTEGER NOT NULL DEFAULT 0 CHECK(seniority_bonus_cents >= 0),
    seniority_years INTEGER NOT NULL DEFAULT 0 CHECK(seniority_years >= 0),
    seniority_rate REAL NOT NULL DEFAULT 0.0 CHECK(seniority_rate >= 0.0 AND seniority_rate <= 1.0),
    gross_salary_cents INTEGER NOT NULL CHECK(gross_salary_cents >= 0),
    total_other_bonus_cents INTEGER NOT NULL DEFAULT 0 CHECK(total_other_bonus_cents >= 0),
    gross_salary_grand_total_cents INTEGER NOT NULL CHECK(gross_salary_grand_total_cents >= 0),
    family_allowance_cents INTEGER NOT NULL DEFAULT 0 CHECK(family_allowance_cents >= 0),
    total_exemptions_cents INTEGER NOT NULL DEFAULT 0 CHECK(total_exemptions_cents >= 0),
    taxable_gross_salary_cents INTEGER NOT NULL CHECK(taxable_gross_salary_cents >= 0),
    social_allowance_employee_contrib_cents INTEGER NOT NULL CHECK(social_allowance_employee_contrib_cents >= 0),
    social_allowance_employer_contrib_cents INTEGER NOT NULL CHECK(social_allowance_employer_contrib_cents >= 0),
    job_loss_compensation_employee_contrib_cents INTEGER NOT NULL CHECK(job_loss_compensation_employee_contrib_cents >= 0),
    job_loss_compensation_employer_contrib_cents INTEGER NOT NULL CHECK(job_loss_compensation_employer_contrib_cents >= 0),
    training_tax_employer_contrib_cents INTEGER NOT NULL CHECK(training_tax_employer_contrib_cents >= 0),
    family_benefits_employer_contrib_cents INTEGER NOT NULL CHECK(family_benefits_employer_contrib_cents >= 0),
    total_cnss_employee_contrib_cents INTEGER NOT NULL CHECK(total_cnss_employee_contrib_cents >= 0),
    total_cnss_employer_contrib_cents INTEGER NOT NULL CHECK(total_cnss_employer_contrib_cents >= 0),
    amo_employee_contrib_cents INTEGER NOT NULL CHECK(amo_employee_contrib_cents >= 0),
    amo_employer_contrib_cents INTEGER NOT NULL CHECK(amo_employer_contrib_cents >= 0),
    taxable_net_salary_cents INTEGER NOT NULL CHECK(taxable_net_salary_cents >= 0),
    income_tax_cents INTEGER NOT NULL CHECK(income_tax_cents >= 0),
    rounding_amount_cents INTEGER NOT NULL CHECK(rounding_amount_cents BETWEEN -100 AND 100),
    net_to_pay_cents INTEGER NOT NULL CHECK(net_to_pay_cents >= 0),

    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT,

    UNIQUE(payroll_period_id, employee_id)
);

CREATE INDEX idx_payroll_result_period_id ON payroll_result(payroll_period_id);

CREATE INDEX idx_payroll_result_employee_id ON payroll_result(employee_id);

CREATE TABLE audit_log(
    id TEXT PRIMARY KEY,
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    action TEXT NOT NULL REFERENCES audit_action(code),
    before TEXT CHECK(json_valid(before)),
    after TEXT NOT NULL CHECK(json_valid(after)),
    timestamp TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_log_table_name_record_id ON audit_log(table_name, record_id);

CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_audit_log_timestamp;

DROP INDEX idx_audit_log_table_name_record_id;

DROP TABLE audit_log;

DROP INDEX idx_payroll_result_employee_id;

DROP INDEX idx_payroll_result_period_id;

DROP TABLE payroll_result;

DROP INDEX idx_payroll_period_org_id;

DROP TABLE payroll_period;

DROP INDEX idx_employee_org_id;

DROP TABLE employee;

DROP INDEX idx_comp_package_org_id;

DROP TABLE employee_compensation_package;

DROP TABLE organization;

DROP TABLE audit_action;
DROP TABLE payroll_period_status;
DROP TABLE marital_status;
DROP TABLE gender;
DROP TABLE currency;
DROP TABLE legal_form;
-- +goose StatementEnd
