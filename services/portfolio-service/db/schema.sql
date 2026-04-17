CREATE TABLE portfolio_entry (
    id            BIGSERIAL     PRIMARY KEY,
    user_id       BIGINT        NOT NULL,
    user_type     VARCHAR(10)   NOT NULL DEFAULT 'CLIENT',
    listing_id    BIGINT        NOT NULL,
    amount        INT           NOT NULL DEFAULT 0,
    buy_price     NUMERIC(20,6) NOT NULL DEFAULT 0,
    last_modified TIMESTAMP     NOT NULL DEFAULT NOW(),
    is_public     BOOLEAN       NOT NULL DEFAULT FALSE,
    public_amount INT           NOT NULL DEFAULT 0,
    account_id    BIGINT        NOT NULL,
    UNIQUE(user_id, listing_id)
);
