CREATE TABLE stock_exchanges (
    id       BIGSERIAL    PRIMARY KEY,
    name     VARCHAR(255) NOT NULL,
    acronym  VARCHAR(20)  NOT NULL,
    mic_code VARCHAR(10)  NOT NULL UNIQUE,
    polity   VARCHAR(100) NOT NULL,
    currency VARCHAR(100) NOT NULL,
    timezone VARCHAR(50)  NOT NULL
);

-- Working hours are shared by all exchanges in the same country (polity).
-- segment: 'pre_market', 'regular', 'post_market'
-- Times are stored in the exchange's local timezone (see stock_exchanges.timezone).
CREATE TABLE exchange_working_hours (
    id         BIGSERIAL    PRIMARY KEY,
    polity     VARCHAR(100) NOT NULL,
    segment    VARCHAR(20)  NOT NULL,
    open_time  TIME         NOT NULL,
    close_time TIME         NOT NULL,
    UNIQUE (polity, segment)
);

-- Holidays are shared by all exchanges in the same country (polity).
CREATE TABLE exchange_holidays (
    id           BIGSERIAL    PRIMARY KEY,
    polity       VARCHAR(100) NOT NULL,
    holiday_date DATE         NOT NULL,
    description  VARCHAR(255),
    UNIQUE (polity, holiday_date)
);

-- Global settings (single row enforced by CHECK).
CREATE TABLE settings (
    id                BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id = TRUE),
    test_mode_enabled BOOLEAN NOT NULL DEFAULT FALSE
);
INSERT INTO settings (test_mode_enabled) VALUES (FALSE);

-- ── Listings ──────────────────────────────────────────────────────────────────

CREATE TYPE listing_type AS ENUM ('STOCK', 'FOREX_PAIR', 'FUTURES_CONTRACT', 'OPTION');
CREATE TYPE liquidity_level AS ENUM ('HIGH', 'MEDIUM', 'LOW');
CREATE TYPE option_type_enum AS ENUM ('CALL', 'PUT');

-- Base listing (all subtypes share this row).
CREATE TABLE listing (
    id           BIGSERIAL     PRIMARY KEY,
    ticker       VARCHAR(20)   NOT NULL UNIQUE,
    name         VARCHAR(255)  NOT NULL,
    exchange_id  BIGINT        NOT NULL REFERENCES stock_exchanges(id),
    last_refresh TIMESTAMP,
    price        NUMERIC(20,6) NOT NULL DEFAULT 0,
    ask          NUMERIC(20,6) NOT NULL DEFAULT 0,
    bid          NUMERIC(20,6) NOT NULL DEFAULT 0,
    volume       BIGINT        NOT NULL DEFAULT 0,
    change       NUMERIC(20,6) NOT NULL DEFAULT 0,
    type         listing_type  NOT NULL
);

-- Daily OHLCV snapshot — one row per listing per calendar day.
CREATE TABLE listing_daily_price_info (
    id         BIGSERIAL     PRIMARY KEY,
    listing_id BIGINT        NOT NULL REFERENCES listing(id) ON DELETE CASCADE,
    date       DATE          NOT NULL,
    price      NUMERIC(20,6) NOT NULL,
    ask        NUMERIC(20,6) NOT NULL,
    bid        NUMERIC(20,6) NOT NULL,
    change     NUMERIC(20,6) NOT NULL DEFAULT 0,
    volume     BIGINT        NOT NULL DEFAULT 0,
    UNIQUE (listing_id, date)
);

-- Subtype: stock
CREATE TABLE listing_stock (
    listing_id         BIGINT        PRIMARY KEY REFERENCES listing(id) ON DELETE CASCADE,
    outstanding_shares BIGINT        NOT NULL DEFAULT 0,
    dividend_yield     NUMERIC(10,6) NOT NULL DEFAULT 0
);

-- Subtype: forex pair
CREATE TABLE listing_forex_pair (
    listing_id     BIGINT         PRIMARY KEY REFERENCES listing(id) ON DELETE CASCADE,
    base_currency  VARCHAR(10)    NOT NULL,
    quote_currency VARCHAR(10)    NOT NULL,
    liquidity      liquidity_level NOT NULL DEFAULT 'MEDIUM'
);

-- Subtype: futures contract
CREATE TABLE listing_futures_contract (
    listing_id      BIGINT        PRIMARY KEY REFERENCES listing(id) ON DELETE CASCADE,
    contract_size   NUMERIC(20,6) NOT NULL DEFAULT 1,
    contract_unit   VARCHAR(50)   NOT NULL DEFAULT '',
    settlement_date DATE
);

-- Subtype: option (listing_option avoids collision with SQL reserved word OPTION)
CREATE TABLE listing_option (
    listing_id         BIGINT           PRIMARY KEY REFERENCES listing(id) ON DELETE CASCADE,
    stock_listing_id   BIGINT           NOT NULL REFERENCES listing(id),
    option_type        option_type_enum NOT NULL,
    strike_price       NUMERIC(20,6)    NOT NULL DEFAULT 0,
    implied_volatility NUMERIC(10,6)    NOT NULL DEFAULT 0,
    open_interest      BIGINT           NOT NULL DEFAULT 0,
    settlement_date    DATE
);
