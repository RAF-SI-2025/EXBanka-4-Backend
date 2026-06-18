CREATE TABLE clients (
    id            BIGSERIAL PRIMARY KEY,
    first_name    VARCHAR     NOT NULL,
    last_name     VARCHAR     NOT NULL,
    jmbg          VARCHAR(13) NOT NULL UNIQUE,
    date_of_birth DATE        NOT NULL,
    gender        VARCHAR     NOT NULL,
    email         VARCHAR     NOT NULL UNIQUE,
    phone_number  VARCHAR     NOT NULL,
    address       VARCHAR     NOT NULL,
    username      VARCHAR     NOT NULL UNIQUE,
    password      VARCHAR,
    active                BOOLEAN     NOT NULL DEFAULT false,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    account_locked_until  TIMESTAMPTZ
);

-- Seed test client used by Cypress e2e tests (password: taraDunjic123)
INSERT INTO clients (first_name, last_name, jmbg, date_of_birth, gender, email, phone_number, address, username, password, active)
SELECT 'Tara', 'Dunjic', '2809002785018', '2002-09-28', 'F', 'dunja@banka.rs', '+381601234567', 'Bulevar oslobodjenja 1, Beograd', 'dunja', '$2a$10$KZiA1q9EmV1PlRI0/m7i7e8a2GgitlGORbgHEbb9Y9ZUNYpxyfY.u', true
WHERE NOT EXISTS (SELECT 1 FROM clients WHERE email = 'dunja@banka.rs');

-- Seed second test client for OTC testing (password: markoMarkovic123)
INSERT INTO clients (first_name, last_name, jmbg, date_of_birth, gender, email, phone_number, address, username, password, active)
SELECT 'Marko', 'Markovic', '1505001710023', '2001-05-15', 'M', 'marko.markovic.test@raf.rs', '+381641234567', 'Knez Mihailova 10, Beograd', 'markomarkovic', '$2a$10$A6F/b5e5iG4U27zS6GWCKum73MPRd23jO61BMmQf30v575ZAqLPw.', true
WHERE NOT EXISTS (SELECT 1 FROM clients WHERE email = 'marko.markovic.test@raf.rs');
