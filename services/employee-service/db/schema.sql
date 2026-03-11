CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE employees (
    id              BIGSERIAL PRIMARY KEY,
    ime             VARCHAR,
    prezime         VARCHAR,
    datum_rodjenja  DATE,
    pol             VARCHAR,
    email           VARCHAR UNIQUE,
    broj_telefona   VARCHAR,
    adresa          VARCHAR,
    username        VARCHAR UNIQUE,
    password        VARCHAR,
    pozicija        VARCHAR,
    departman       VARCHAR,
    aktivan         BOOLEAN,
    dozvole         TEXT[]
);

CREATE INDEX IF NOT EXISTS idx_employees_ime      ON employees (ime);
CREATE INDEX IF NOT EXISTS idx_employees_prezime  ON employees (prezime);
CREATE INDEX IF NOT EXISTS idx_employees_pozicija ON employees (pozicija);

INSERT INTO employees (ime, prezime, datum_rodjenja, pol, email, broj_telefona, adresa, username, password, pozicija, departman, aktivan, dozvole)
SELECT 'Admin', 'Admin', '1990-01-01', 'M', 'admin@exbanka.com', '', '', 'admin', crypt('admin', gen_salt('bf', 10)), 'Administrator', 'IT', true, ARRAY['ADMIN', 'READ', 'WRITE', 'DELETE']
WHERE NOT EXISTS (SELECT 1 FROM employees WHERE username = 'admin');
