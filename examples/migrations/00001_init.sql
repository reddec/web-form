-- +migrate Up

CREATE TABLE shop
(
    ID            BIGSERIAL   NOT NULL PRIMARY KEY,
    CREATED_AT    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    DELIVERY_DATE TIMESTAMPTZ NOT NULL,
    BIRTHDAY      DATE        NOT NULL,
    CLIENT_ID     TEXT        NOT NULL,
    DOUGH         TEXT        NOT NULL DEFAULT 'thin',
    CHEESE        TEXT[]      NOT NULL,
    PHONE         TEXT        NOT NULL,
    EMAIL         TEXT,
    NOTIFY_SMS    BOOLEAN     NOT NULL DEFAULT FALSE,
    ZIP           INTEGER     NOT NULL,
    ADDRESS       TEXT        NOT NULL
);

-- +migrate Down
DROP TABLE shop;
