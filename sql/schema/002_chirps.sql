-- +goose Up
CREATE TABLE chirps (
    id uuid not null,
    created_at timestamp not null,
    updated_at timestamp not null,
    body text not null,
    user_id uuid not null,
    primary key (id),
    foreign key (user_id)
    references users(id) on delete cascade
);

-- +goose Down
DROP TABLE chirps;