CREATE TABLE authorized_persons (
    id             BIGSERIAL PRIMARY KEY,
    first_name     VARCHAR NOT NULL,
    last_name      VARCHAR NOT NULL,
    date_of_birth  DATE    NOT NULL,
    gender         VARCHAR NOT NULL,
    email          VARCHAR NOT NULL,
    phone_number   VARCHAR NOT NULL,
    address        VARCHAR NOT NULL,
    account_number VARCHAR NOT NULL  -- logical FK to account_db.accounts (enforced at app level)
);

CREATE TABLE card_requests (
    id                     BIGSERIAL    PRIMARY KEY,
    request_token          VARCHAR      NOT NULL UNIQUE,
    account_number         VARCHAR      NOT NULL,
    card_name              VARCHAR      NOT NULL,
    caller_client_id       BIGINT       NOT NULL,
    for_self               BOOLEAN      NOT NULL DEFAULT true,
    authorized_person_data JSONB,                           -- non-null when for_self = false
    confirmation_code      VARCHAR      NOT NULL,
    expires_at             TIMESTAMPTZ  NOT NULL,
    used                   BOOLEAN      NOT NULL DEFAULT false
);

CREATE TABLE cards (
    id                   BIGSERIAL PRIMARY KEY,
    card_number          VARCHAR       NOT NULL UNIQUE,
    card_type            VARCHAR       NOT NULL DEFAULT 'DEBIT',  -- DEBIT | CREDIT
    card_name            VARCHAR       NOT NULL,                  -- VISA | MASTERCARD | DINACARD | AMERICAN_EXPRESS
    created_at           TIMESTAMP     NOT NULL DEFAULT NOW(),
    expiry_date          DATE          NOT NULL,
    account_number       VARCHAR       NOT NULL,                  -- logical FK to account_db.accounts (enforced at app level)
    cvv                  VARCHAR       NOT NULL,                  -- stored hashed
    card_limit           NUMERIC(20,2),
    status               VARCHAR       NOT NULL DEFAULT 'ACTIVE', -- ACTIVE | BLOCKED | DEACTIVATED
    authorized_person_id BIGINT        REFERENCES authorized_persons(id)
);
