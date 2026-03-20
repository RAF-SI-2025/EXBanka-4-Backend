CREATE TABLE activation_tokens (
    token       VARCHAR PRIMARY KEY,
    employee_id BIGINT      NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token       VARCHAR PRIMARY KEY,
    employee_id BIGINT      NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS client_activation_tokens (
    token      VARCHAR PRIMARY KEY,
    client_id  BIGINT      NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
