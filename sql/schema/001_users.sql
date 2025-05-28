-- +goose Up
CREATE TABLE users (
    id uuid unique not null,
    created_at timestamp not null,
    updated_at timestamp not null,
    email text not null
);

-- +goose Down
DROP TABLE users;