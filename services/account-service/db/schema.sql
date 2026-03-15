CREATE TABLE activity_codes (
    code        VARCHAR PRIMARY KEY,
    description VARCHAR NOT NULL
);

INSERT INTO activity_codes (code, description) VALUES
  ('01.1',  'Gajenje jednogodišnjih biljaka'),
  ('10.1',  'Prerada i konzerviranje mesa'),
  ('41.2',  'Izgradnja stambenih i nestambenih zgrada'),
  ('45.1',  'Trgovina motornim vozilima'),
  ('46.1',  'Posredovanje u trgovini'),
  ('47.1',  'Trgovina na malo u nespecijalizovanim prodavnicama'),
  ('56.1',  'Delatnost restorana i pokretnih ugostiteljskih objekata'),
  ('62.01', 'Računarsko programiranje'),
  ('62.02', 'Konsultantske delatnosti u oblasti informacione tehnologije'),
  ('64.19', 'Ostalo monetarno posredovanje'),
  ('69.1',  'Pravne delatnosti'),
  ('70.2',  'Konsultantske delatnosti u oblasti poslovanja'),
  ('85.3',  'Srednje obrazovanje'),
  ('86.1',  'Bolničke delatnosti'),
  ('96.0',  'Ostale lične uslužne delatnosti')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE companies (
    id                  BIGSERIAL PRIMARY KEY,
    name                VARCHAR NOT NULL,
    registration_number VARCHAR NOT NULL UNIQUE,
    pib                 VARCHAR NOT NULL UNIQUE,
    activity_code       VARCHAR REFERENCES activity_codes(code),
    address             VARCHAR,
    owner_client_id     BIGINT NOT NULL
);

CREATE TABLE accounts (
    id                  BIGSERIAL PRIMARY KEY,
    account_number      VARCHAR NOT NULL UNIQUE,
    account_name        VARCHAR,
    owner_id            BIGINT NOT NULL,
    employee_id         BIGINT NOT NULL,
    currency_id         BIGINT NOT NULL,
    account_type        VARCHAR NOT NULL,
    status              VARCHAR NOT NULL DEFAULT 'ACTIVE',
    balance             NUMERIC(20, 2) NOT NULL DEFAULT 0,
    available_balance   NUMERIC(20, 2) NOT NULL DEFAULT 0,
    created_date        DATE NOT NULL DEFAULT CURRENT_DATE,
    expiration_date     DATE,
    daily_limit         NUMERIC(20, 2),
    monthly_limit       NUMERIC(20, 2),
    daily_spent         NUMERIC(20, 2) NOT NULL DEFAULT 0,
    monthly_spent       NUMERIC(20, 2) NOT NULL DEFAULT 0,
    maintenance_fee     NUMERIC(20, 2) NOT NULL DEFAULT 0,
    company_id          BIGINT REFERENCES companies(id)
);
