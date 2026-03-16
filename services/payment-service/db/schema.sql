CREATE TABLE payments (
    id               BIGSERIAL PRIMARY KEY,
    order_number     VARCHAR     NOT NULL UNIQUE,
    from_account     VARCHAR     NOT NULL,
    to_account       VARCHAR     NOT NULL,
    initial_amount   NUMERIC(20, 2) NOT NULL,
    final_amount     NUMERIC(20, 2) NOT NULL,
    fee              NUMERIC(20, 2) NOT NULL DEFAULT 0,
    recipient_id     BIGINT,
    payment_code     VARCHAR,
    reference_number VARCHAR,
    purpose          VARCHAR,
    timestamp        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status           VARCHAR     NOT NULL DEFAULT 'PROCESSING'
);
