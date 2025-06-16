-- +goose Up
alter TABLE users 
add is_chirpy_red bool not null default false;

-- +goose Down
alter table users
drop column is_chirpy_red;