CREATE TABLE IF NOT EXISTS otc_negotiations (
    id               BIGSERIAL PRIMARY KEY,
    ticker           VARCHAR(20)    NOT NULL,
    seller_id        BIGINT         NOT NULL,
    seller_type      VARCHAR(10)    NOT NULL DEFAULT 'CLIENT',
    buyer_id         BIGINT         NOT NULL,
    buyer_type       VARCHAR(10)    NOT NULL DEFAULT 'CLIENT',
    amount           INT            NOT NULL,
    price_per_stock  DECIMAL(18,4)  NOT NULL,
    settlement_date  DATE           NOT NULL,
    premium          DECIMAL(18,4)  NOT NULL,
    currency         VARCHAR(10)    NOT NULL,
    last_modified    TIMESTAMP      NOT NULL DEFAULT NOW(),
    modified_by_id   BIGINT,
    modified_by_type VARCHAR(10),
    status           VARCHAR(20)    NOT NULL DEFAULT 'PENDING_SELLER',
    -- cross-bank negotiation fields (NULL for intra-bank negotiations)
    buyer_routing_number   INTEGER,
    buyer_external_id      VARCHAR(100),
    seller_routing_number  INTEGER,
    seller_external_id     VARCHAR(100),
    creator_routing_number INTEGER,
    creator_external_id    VARCHAR(100),
    UNIQUE (creator_routing_number, creator_external_id)
);

-- Migration: add cross-bank columns if upgrading an existing deployment
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS buyer_routing_number   INTEGER;
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS buyer_external_id      VARCHAR(100);
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS seller_routing_number  INTEGER;
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS seller_external_id     VARCHAR(100);
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS creator_routing_number INTEGER;
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS creator_external_id    VARCHAR(100);

CREATE TABLE IF NOT EXISTS otc_contracts (
    id               BIGSERIAL PRIMARY KEY,
    negotiation_id   BIGINT NOT NULL UNIQUE REFERENCES otc_negotiations(id),
    seller_id        BIGINT NOT NULL,
    seller_type      VARCHAR(10) NOT NULL,
    buyer_id         BIGINT NOT NULL,
    buyer_type       VARCHAR(10) NOT NULL,
    ticker           VARCHAR(20) NOT NULL,
    amount           INT NOT NULL,
    strike_price     DECIMAL(18,4) NOT NULL,
    premium          DECIMAL(18,4) NOT NULL,
    currency         VARCHAR(10) NOT NULL,
    settlement_date  DATE NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS otc_saga_log (
    id           BIGSERIAL PRIMARY KEY,
    contract_id  BIGINT NOT NULL,
    step         INT NOT NULL,
    status       VARCHAR(20) NOT NULL,
    timestamp    TIMESTAMP NOT NULL DEFAULT NOW(),
    error_msg    TEXT
);

-- Globalni SAGA tracker: jedan red po toku, prati status (Running/Compensating/Completed/Compensated)
CREATE TABLE IF NOT EXISTS otc_saga (
    id           BIGSERIAL PRIMARY KEY,
    contract_id  BIGINT NOT NULL UNIQUE,
    status       VARCHAR(20) NOT NULL DEFAULT 'Running',
    current_step INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
-- Migration za postojeće deploymente
ALTER TABLE otc_saga ADD COLUMN IF NOT EXISTS current_step INT NOT NULL DEFAULT 0;
ALTER TABLE otc_saga ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT NOW();

-- Migration: drop over-broad unique constraint that blocked multiple negotiations from the same buyer
ALTER TABLE otc_negotiations DROP CONSTRAINT IF EXISTS otc_negotiations_creator_routing_number_creator_external_id_key;

-- Migration: partner_negotiation_id = lokalni ID pregovora na partner banci (za outgoing OPTION posting)
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS partner_negotiation_id BIGINT;
-- Migration: partner banks (e.g. Banka 4) return UUID strings, not integers
ALTER TABLE otc_negotiations ALTER COLUMN partner_negotiation_id TYPE TEXT USING partner_negotiation_id::TEXT;

-- Tabela za tracking incoming cross-bank 2PC transakcija koje uključuju OPTION postinge
CREATE TABLE IF NOT EXISTS otc_interbank_tx (
    id                  BIGSERIAL PRIMARY KEY,
    idem_routing_number VARCHAR(20)  NOT NULL,
    idem_key            VARCHAR(100) NOT NULL,
    tx_routing_number   VARCHAR(20),
    tx_id               VARCHAR(100),
    negotiation_id      BIGINT       NOT NULL REFERENCES otc_negotiations(id),
    tx_type             VARCHAR(20)  NOT NULL,
    stock_amount        INT          NOT NULL,
    status              VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    cached_vote         VARCHAR(10),
    created_at          TIMESTAMP    NOT NULL DEFAULT NOW(),
    UNIQUE(idem_routing_number, idem_key)
);
ALTER TABLE otc_interbank_tx ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT NOW();
ALTER TABLE otc_negotiations ADD COLUMN IF NOT EXISTS buyer_account_number VARCHAR(50);

CREATE TABLE IF NOT EXISTS otc_negotiation_history (
    id                  BIGSERIAL PRIMARY KEY,
    negotiation_id      BIGINT NOT NULL REFERENCES otc_negotiations(id),
    action              VARCHAR(20) NOT NULL,
    actor_id            BIGINT NOT NULL,
    actor_type          VARCHAR(20) NOT NULL,
    actor_name          VARCHAR(200),
    old_amount          INT,
    new_amount          INT,
    old_price_per_stock DOUBLE PRECISION,
    new_price_per_stock DOUBLE PRECISION,
    old_settlement_date VARCHAR(20),
    new_settlement_date VARCHAR(20),
    old_premium         DOUBLE PRECISION,
    new_premium         DOUBLE PRECISION,
    timestamp           TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_otc_negotiation_history_neg_id ON otc_negotiation_history(negotiation_id);
