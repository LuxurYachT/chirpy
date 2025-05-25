-- +goose Up
CREATE EXTENSION pgcrypto;

-- +goose Down
DROP EXTENSION pgcrypto;