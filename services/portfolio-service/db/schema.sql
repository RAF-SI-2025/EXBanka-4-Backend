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
    account_id      BIGINT        NOT NULL,
    reserved_amount INT           NOT NULL DEFAULT 0,
    UNIQUE(user_id, user_type, listing_id)
);

CREATE TABLE watchlists (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL,
    user_type  VARCHAR(10) NOT NULL CHECK (user_type IN ('EMPLOYEE', 'CLIENT')),
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, user_type, name)
);
CREATE TABLE watchlist_items (
    id           BIGSERIAL PRIMARY KEY,
    watchlist_id BIGINT NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    listing_id   BIGINT NOT NULL,
    added_at     TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (watchlist_id, listing_id)
);
CREATE INDEX idx_watchlist_items_watchlist ON watchlist_items(watchlist_id);

CREATE TABLE dividend_payouts (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL,
    user_type        VARCHAR(10) NOT NULL CHECK (user_type IN ('EMPLOYEE', 'CLIENT')),
    stock_listing_id BIGINT NOT NULL,
    quantity         NUMERIC(20,6) NOT NULL,
    gross_amount     NUMERIC(20,6) NOT NULL,
    currency         VARCHAR(10) NOT NULL,
    tax_rsd          NUMERIC(20,6) DEFAULT 0,
    net_amount       NUMERIC(20,6) NOT NULL,
    account_id       BIGINT,
    payment_date     DATE NOT NULL,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_dividend_payouts_user ON dividend_payouts(user_id, user_type, payment_date DESC);

CREATE TABLE tax_record (
    id         BIGSERIAL     PRIMARY KEY,
    user_id    BIGINT        NOT NULL,
    user_type  VARCHAR(10)   NOT NULL DEFAULT 'CLIENT',
    amount_rsd NUMERIC(20,6) NOT NULL,
    month      INT           NOT NULL,
    year       INT           NOT NULL,
    is_paid    BOOLEAN       NOT NULL DEFAULT FALSE,
    paid_at    TIMESTAMP
);
