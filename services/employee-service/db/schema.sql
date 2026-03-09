CREATE TABLE employees (
    id              BIGINT PRIMARY KEY,
    ime             VARCHAR,
    prezime         VARCHAR,
    datum_rodjenja  DATE,
    pol             VARCHAR,
    email           VARCHAR UNIQUE,
    broj_telefona   VARCHAR,
    adresa          VARCHAR,
    username        VARCHAR,
    password        VARCHAR,
    pozicija        VARCHAR,
    departman       VARCHAR,
    aktivan         BOOLEAN,
    dozvole         TEXT[]
);

INSERT INTO employees (id, ime, prezime, datum_rodjenja, pol, email, broj_telefona, adresa, username, password, pozicija, departman, aktivan, dozvole)
SELECT 1, 'Admin', 'Admin', '1990-01-01', 'M', 'admin@exbanka.com', '', '', 'admin', 'admin', 'Administrator', 'IT', true, ARRAY['ADMIN', 'READ', 'WRITE', 'DELETE']
WHERE NOT EXISTS (SELECT 1 FROM employees WHERE username = 'admin');
