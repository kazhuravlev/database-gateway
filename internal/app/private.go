// Database Gateway provides access to servers with ACL for safe and restricted database interactions.
// Copyright (C) 2024  Kirill Zhuravlev
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/kazhuravlev/just"
)

func (s *Service) getConnection(ctx context.Context, target config.Target) (*pgxpool.Pool, error) { //nolint:gocritic
	{
		s.connsMu.RLock()
		pool, ok := s.conns[target.ID]
		s.connsMu.RUnlock()

		if ok {
			return pool, nil
		}
	}

	s.opts.logger.Info("connect to target", slog.String("target", string(target.ID)))
	pgCfg := target.Connection

	connErr := func(reason error) error {
		return fmt.Errorf("connect to target %q (%s:%d): %w", target.ID, pgCfg.Host, pgCfg.Port, reason)
	}

	connURL := url.URL{ //nolint:exhaustruct_v5
		Scheme: "postgres",
		User:   url.UserPassword(pgCfg.User, pgCfg.Password),
		Host:   fmt.Sprintf("%s:%d", pgCfg.Host, pgCfg.Port),
		Path:   "/" + pgCfg.DB,
		RawQuery: url.Values{
			"sslmode": {just.If(pgCfg.UseSSL, "enable", "disable")},
		}.Encode(),
	}

	dbpool, err := pgxpool.New(ctx, connURL.String())
	if err != nil {
		return nil, connErr(fmt.Errorf("create db pool: %w", err))
	}

	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()

		return nil, connErr(fmt.Errorf("ping: %w", err))
	}
	// TODO: we cann disconnect on shutdown. But this is not so important.
	// dbpool.Close()

	s.connsMu.Lock()
	s.conns[target.ID] = dbpool
	s.connsMu.Unlock()

	return dbpool, nil
}

func (s *Service) getTargetByID(_ context.Context, user structs.User, tID config.TargetID) (*config.Target, *validator.DbSchema, error) {
	subjects := userSubjects(user)
	for i := range s.opts.targets {
		target := s.opts.targets[i]
		if target.ID == tID {
			if s.opts.authorizer.AllowTarget(subjects, target.ID.S()) {
				schema := validator.NewDbSchema(target.DefaultSchema, target.Tables)

				return &target, schema, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("target not found: %w", ErrNotFound)
}
