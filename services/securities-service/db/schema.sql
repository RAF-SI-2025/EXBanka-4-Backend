CREATE TABLE stock_exchanges (
    id       BIGSERIAL    PRIMARY KEY,
    name     VARCHAR(255) NOT NULL,
    acronym  VARCHAR(20)  NOT NULL,
    mic_code VARCHAR(10)  NOT NULL UNIQUE,
    polity   VARCHAR(100) NOT NULL,
    currency VARCHAR(10)  NOT NULL,
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
