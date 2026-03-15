CREATE TABLE clients (
    id              BIGSERIAL PRIMARY KEY,
    first_name      VARCHAR,
    last_name       VARCHAR,
    date_of_birth   DATE,
    gender          VARCHAR,
    email           VARCHAR UNIQUE,
    phone_number    VARCHAR,
    address         VARCHAR,
    password        VARCHAR,
    linked_accounts TEXT[],
    active          BOOLEAN DEFAULT false
);
