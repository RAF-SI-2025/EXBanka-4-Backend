CREATE TABLE clients (
    id            BIGSERIAL PRIMARY KEY,
    first_name    VARCHAR     NOT NULL,
    last_name     VARCHAR     NOT NULL,
    jmbg          VARCHAR(13) NOT NULL UNIQUE,
    date_of_birth DATE        NOT NULL,
    gender        VARCHAR     NOT NULL,
    email         VARCHAR     NOT NULL UNIQUE,
    phone_number  VARCHAR     NOT NULL,
    address       VARCHAR     NOT NULL,
    username      VARCHAR     NOT NULL UNIQUE,
    password      VARCHAR,
    active        BOOLEAN     NOT NULL DEFAULT false
);
