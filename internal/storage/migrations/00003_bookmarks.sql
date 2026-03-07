-- Database Gateway provides access to servers with ACL for safe and restricted database interactions.
-- Copyright (C) 2024  Kirill Zhuravlev
--
-- This program is free software: you can redistribute it and/or modify
-- it under the terms of the GNU General Public License as published by
-- the Free Software Foundation, either version 3 of the License, or
-- (at your option) any later version.
--
-- This program is distributed in the hope that it will be useful,
-- but WITHOUT ANY WARRANTY; without even the implied warranty of
-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
-- GNU General Public License for more details.
--
-- You should have received a copy of the GNU General Public License
-- along with this program.  If not, see <https://www.gnu.org/licenses/>.

-- +goose Up
-- +goose StatementBegin

create table bookmarks
(
    id         uuid        not null,
    user_id    text        not null,
    target_id  text        not null,
    title      text        not null,
    query      text        not null,
    created_at timestamptz not null,

    primary key (id)
);

create index idx_bookmarks_user_target_created_at
    on bookmarks (user_id, target_id, created_at desc);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop table bookmarks;

-- +goose StatementEnd
