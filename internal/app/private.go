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
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
)

func (s *Service) findUser(fn func(user config.User) bool) (*config.User, error) {
	users, ok := s.opts.users.Provider.(config.UsersProviderConfig)
	if !ok {
		return nil, errors.New("not implemented") //nolint:err113
	}

	user := just.SliceFindFirst(users, func(_ int, user config.User) bool {
		return fn(user)
	})

	if user, ok := user.ValueOk(); ok {
		return &user, nil
	}

	return nil, fmt.Errorf("user not exists: %w", ErrNotFound)
}

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

	urlExample := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		pgCfg.User,
		pgCfg.Password,
		pgCfg.Host,
		pgCfg.Port,
		pgCfg.DB,
		just.If(pgCfg.UseSSL, "enable", "disable"),
	)
	dbpool, err := pgxpool.New(ctx, urlExample)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}
	// TODO: we cann disconnect on shutdown. But this is not so important.
	// dbpool.Close()

	s.connsMu.Lock()
	s.conns[target.ID] = dbpool
	s.connsMu.Unlock()

	return dbpool, nil
}

func (s *Service) getTargetByID(_ context.Context, uID config.UserID, tID config.TargetID) (*config.Target, error) {
	for i := range s.opts.targets {
		target := s.opts.targets[i]
		if target.ID == tID {
			if s.opts.acls.Allow(rules.ByUserID(uID.S()), rules.ByTargetID(target.ID.S())) {
				return &target, nil
			}
		}
	}

	return nil, fmt.Errorf("target not found: %w", ErrNotFound)
}
