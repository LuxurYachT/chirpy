-- +goose Up
CREATE TABLE users (
    id uuid,
    created_at timestamp,
    updated_at timestamp,
    email text not null
);

-- +goose Down
DROP TABLE users;