-- +goose Up
-- +goose StatementBegin

alter table query_results add column state text not null default 'completed';
alter table query_results alter column state drop default;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

alter table query_results drop column state;

-- +goose StatementEnd
