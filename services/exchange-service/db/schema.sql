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
