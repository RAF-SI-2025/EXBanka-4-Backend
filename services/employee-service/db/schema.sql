CREATE TABLE employees (
    id              BIGINT PRIMARY KEY,
    ime             VARCHAR,
    prezime         VARCHAR,
    datum_rodjenja  DATE,
    pol             VARCHAR,
    email           VARCHAR,
    broj_telefona   VARCHAR,
    adresa          VARCHAR,
    username        VARCHAR,
    password        VARCHAR,
    pozicija        VARCHAR,
    departman       VARCHAR,
    aktivan         BOOLEAN
);
