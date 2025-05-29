-- +goose Up
alter TABLE users 
add hashed_password text not null default 'unset';

-- +goose Down
alter table users
drop column hashed_password;