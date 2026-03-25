CREATE TABLE payment_recipients (
    id             BIGSERIAL PRIMARY KEY,
    client_id      BIGINT    NOT NULL,
    name           VARCHAR   NOT NULL,
    account_number VARCHAR   NOT NULL,
    "order"        INT       NOT NULL DEFAULT 0
);

CREATE TABLE transfers (
    id             BIGSERIAL PRIMARY KEY,
    order_number   VARCHAR        NOT NULL UNIQUE,
    from_account   VARCHAR        NOT NULL,
    to_account     VARCHAR        NOT NULL,
    initial_amount NUMERIC(20, 2) NOT NULL,
    final_amount   NUMERIC(20, 2) NOT NULL,
    exchange_rate  NUMERIC(20, 6) NOT NULL DEFAULT 1,
    fee            NUMERIC(20, 2) NOT NULL DEFAULT 0,
    timestamp      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE payments (
    id               BIGSERIAL PRIMARY KEY,
    order_number     VARCHAR     NOT NULL UNIQUE,
    from_account     VARCHAR     NOT NULL,
    to_account       VARCHAR     NOT NULL,
    initial_amount   NUMERIC(20, 2) NOT NULL,
    final_amount     NUMERIC(20, 2) NOT NULL,
    fee              NUMERIC(20, 2) NOT NULL DEFAULT 0,
    recipient_id     BIGINT REFERENCES payment_recipients(id),
    payment_code     VARCHAR,
    reference_number VARCHAR,
    purpose          VARCHAR,
    timestamp        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status           VARCHAR     NOT NULL DEFAULT 'PROCESSING'
);
