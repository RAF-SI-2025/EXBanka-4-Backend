CREATE TABLE currencies (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR NOT NULL,
    code        VARCHAR NOT NULL UNIQUE,
    symbol      VARCHAR NOT NULL,
    country     VARCHAR NOT NULL,
    description VARCHAR,
    status      VARCHAR NOT NULL DEFAULT 'ACTIVE'
);

INSERT INTO currencies (name, code, symbol, country, description, status) VALUES
  ('Serbian Dinar',     'RSD', 'din',  'Serbia',         'Serbian national currency',    'ACTIVE'),
  ('Euro',              'EUR', '€',    'European Union',  'EU common currency',           'ACTIVE'),
  ('Swiss Franc',       'CHF', 'Fr',   'Switzerland',    'Swiss national currency',      'ACTIVE'),
  ('US Dollar',         'USD', '$',    'United States',  'US national currency',         'ACTIVE'),
  ('British Pound',     'GBP', '£',    'United Kingdom', 'UK national currency',         'ACTIVE'),
  ('Japanese Yen',      'JPY', '¥',    'Japan',          'Japanese national currency',   'ACTIVE'),
  ('Canadian Dollar',   'CAD', 'CA$',  'Canada',         'Canadian national currency',   'ACTIVE'),
  ('Australian Dollar', 'AUD', 'AU$',  'Australia',      'Australian national currency', 'ACTIVE')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE exchange_rates (
    from_currency VARCHAR NOT NULL,
    to_currency   VARCHAR NOT NULL,
    rate          NUMERIC(20, 6) NOT NULL,
    PRIMARY KEY (from_currency, to_currency)
);

INSERT INTO exchange_rates (from_currency, to_currency, rate) VALUES
  ('RSD', 'EUR', 0.008547), ('EUR', 'RSD', 117.0),
  ('RSD', 'USD', 0.009259), ('USD', 'RSD', 108.0),
  ('RSD', 'CHF', 0.008621), ('CHF', 'RSD', 116.0),
  ('RSD', 'GBP', 0.007353), ('GBP', 'RSD', 136.0),
  ('RSD', 'JPY', 1.380000), ('JPY', 'RSD', 0.724638),
  ('RSD', 'CAD', 0.012500), ('CAD', 'RSD', 80.0),
  ('RSD', 'AUD', 0.014286), ('AUD', 'RSD', 70.0),
  ('EUR', 'USD', 1.083),    ('USD', 'EUR', 0.923),
  ('EUR', 'CHF', 1.008),    ('CHF', 'EUR', 0.992),
  ('EUR', 'GBP', 0.860),    ('GBP', 'EUR', 1.163),
  ('USD', 'CHF', 0.931),    ('CHF', 'USD', 1.074),
  ('USD', 'GBP', 0.794),    ('GBP', 'USD', 1.259),
  ('CHF', 'GBP', 0.854),    ('GBP', 'CHF', 1.170)
ON CONFLICT DO NOTHING;

-- Daily buy/sell rates vs RSD — used by exchange-service (issue #72)
CREATE TABLE daily_exchange_rates (
    id            BIGSERIAL PRIMARY KEY,
    currency_code VARCHAR NOT NULL REFERENCES currencies(code),
    buying_rate   NUMERIC(20, 6) NOT NULL,
    selling_rate  NUMERIC(20, 6) NOT NULL,
    middle_rate   NUMERIC(20, 6) NOT NULL,
    date          DATE NOT NULL DEFAULT CURRENT_DATE,
    UNIQUE (currency_code, date)
);

-- Exchange transaction history (issue #76)
CREATE TABLE exchange_transactions (
    id            BIGSERIAL PRIMARY KEY,
    client_id     BIGINT NOT NULL,
    from_account  VARCHAR NOT NULL,
    to_account    VARCHAR NOT NULL,
    from_currency VARCHAR NOT NULL,
    to_currency   VARCHAR NOT NULL,
    from_amount   NUMERIC(20, 2) NOT NULL,
    to_amount     NUMERIC(20, 2) NOT NULL,
    rate          NUMERIC(20, 6) NOT NULL,
    commission    NUMERIC(20, 2) NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status        VARCHAR NOT NULL DEFAULT 'COMPLETED'
);
