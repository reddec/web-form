-- +migrate Up

CREATE TABLE pizza
(
    ID           BIGSERIAL   NOT NULL PRIMARY KEY,
    CREATED_AT   TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    EMPLOYEE     TEXT        NOT NULL,
    PIZZA_KIND   TEXT        NOT NULL,
    EXTRA_CHEESE BOOLEAN     NOT NULL DEFAULT false
);


-- +migrate Down
DROP TABLE pizza;
