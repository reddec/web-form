-- +migrate Up

CREATE TABLE simple
(
    ID         BIGSERIAL   NOT NULL PRIMARY KEY,
    CREATED_AT TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    NAME       TEXT        NOT NULL,
    YEAR       INTEGER     NOT NULL,
    COMMENT    TEXT
);

CREATE TABLE birthday
(
    ID         BIGSERIAL   NOT NULL PRIMARY KEY,
    CREATED_AT TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    EMPLOYEE   TEXT        NOT NULL,
    BIRTHDAY   DATE        NOT NULL
);


-- +migrate Down
DROP TABLE simple;
DROP TABLE birthday;
