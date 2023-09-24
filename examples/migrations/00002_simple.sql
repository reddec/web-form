-- +migrate Up

CREATE TABLE simple
(
    ID         BIGSERIAL   NOT NULL PRIMARY KEY,
    CREATED_AT TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    NAME       TEXT        NOT NULL,
    YEAR       INTEGER     NOT NULL,
    COMMENT    TEXT
);

-- +migrate Down
DROP TABLE simple;
