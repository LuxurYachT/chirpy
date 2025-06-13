-- +goose Up
CREATE TABLE refresh_tokens (
    token text not null,
    user_id uuid not null,
    created_at timestamp not null,
    updated_at timestamp not null,
    expires_at timestamp not null,
    revoked_at timestamp,
    primary key (token),
    foreign key (user_id)
    references users(id) on delete cascade
);

-- +goose Down
DROP TABLE refresh_tokens;