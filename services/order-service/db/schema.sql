CREATE TYPE order_type_enum AS ENUM ('MARKET', 'LIMIT', 'STOP', 'STOP_LIMIT');
CREATE TYPE order_direction AS ENUM ('BUY', 'SELL');
CREATE TYPE order_status AS ENUM ('PENDING', 'APPROVED', 'DECLINED');

CREATE TABLE orders (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            BIGINT          NOT NULL,
    user_type          VARCHAR(10)     NOT NULL DEFAULT 'CLIENT',
    asset_id           BIGINT          NOT NULL,
    order_type         order_type_enum NOT NULL,
    quantity           INT             NOT NULL,
    contract_size      INT             NOT NULL DEFAULT 1,
    price_per_unit     NUMERIC(20,6)   NOT NULL DEFAULT 0,
    limit_value        NUMERIC(20,6),
    stop_value         NUMERIC(20,6),
    direction          order_direction NOT NULL,
    status             order_status    NOT NULL DEFAULT 'PENDING',
    approved_by        BIGINT,
    is_done            BOOLEAN         NOT NULL DEFAULT FALSE,
    last_modification  TIMESTAMP       NOT NULL DEFAULT NOW(),
    remaining_portions INT             NOT NULL DEFAULT 0,
    after_hours        BOOLEAN         NOT NULL DEFAULT FALSE,
    is_aon             BOOLEAN         NOT NULL DEFAULT FALSE,
    is_margin          BOOLEAN         NOT NULL DEFAULT FALSE,
    account_id         BIGINT          NOT NULL
);

CREATE TABLE order_portions (
    id        BIGSERIAL PRIMARY KEY,
    order_id  BIGINT        NOT NULL REFERENCES orders(id),
    quantity  INT           NOT NULL,
    price     NUMERIC(20,6) NOT NULL,
    filled_at TIMESTAMP     NOT NULL DEFAULT NOW()
);
