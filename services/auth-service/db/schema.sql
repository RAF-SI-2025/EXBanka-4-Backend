CREATE TABLE activation_tokens (
    token       VARCHAR PRIMARY KEY,
    employee_id BIGINT      NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);
