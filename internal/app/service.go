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
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/kazhuravlev/just"
	"github.com/labstack/gommon/log"
)

var ErrNotFound = errors.New("not found")

type Service struct {
	opts Options

	connsMu *sync.RWMutex
	conns   map[config.TargetID]*pgxpool.Pool
}

func New(opts Options) (*Service, error) {
	return &Service{opts: opts, connsMu: new(sync.RWMutex), conns: make(map[config.TargetID]*pgxpool.Pool)}, nil
}

func (s *Service) findUser(fn func(user config.User) bool) (*config.User, error) {
	users, ok := s.opts.cfg.Users.Provider.(config.UsersProviderConfig)
	if !ok {
		return nil, errors.New("not implemented")
	}

	user := just.SliceFindFirst(users, func(_ int, user config.User) bool {
		return fn(user)
	})

	if user, ok := user.ValueOk(); ok {
		return &user, nil
	}

	return nil, fmt.Errorf("user not exists: %w", ErrNotFound)
}

func (s *Service) AuthUser(ctx context.Context, username, password string) (config.UserID, error) {
	user, err := s.findUser(func(user config.User) bool {
		return user.Username == username && user.Password == password
	})
	if err != nil {
		return "", err
	}

	return user.ID, nil
}

func (s *Service) GetUserByID(ctx context.Context, id config.UserID) (*config.User, error) {
	user, err := s.findUser(func(user config.User) bool {
		return user.ID == id
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) GetTargets(ctx context.Context) ([]structs.Server, error) {
	servers := just.SliceMap(s.opts.cfg.Targets, func(t config.Target) structs.Server {
		return structs.Server{
			ID:     t.ID,
			Type:   t.Type,
			Tables: t.Tables,
		}
	})

	return servers, nil
}

func (s *Service) GetTargetByID(ctx context.Context, id config.TargetID) (*config.Target, error) {
	for i := range s.opts.cfg.Targets {
		target := s.opts.cfg.Targets[i]
		if target.ID == id {
			return &target, nil
		}
	}

	return nil, fmt.Errorf("target not found: %w", ErrNotFound)
}

func (s *Service) GetACLs(ctx context.Context, uID config.UserID, tID config.TargetID) []config.ACL {
	var res []config.ACL
	for _, acl := range s.opts.cfg.ACLs {
		if acl.Target == tID && acl.User == uID {
			res = append(res, acl)
		}
	}

	return res
}

func (s *Service) RunQuery(ctx context.Context, userID config.UserID, srvID config.TargetID, query string) (*structs.QTable, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	srv, err := s.GetTargetByID(ctx, srvID)
	if err != nil {
		return nil, fmt.Errorf("get target by id: %w", err)
	}

	acls := s.GetACLs(ctx, user.ID, srvID)

	if err := validator.IsAllowed(srv.Tables, acls, query); err != nil {
		log.Error("err", err.Error())

		return nil, fmt.Errorf("preflight check: %w", err)
	}

	conn, err := s.getConnectionByID(ctx, *srv)
	if err != nil {
		return nil, fmt.Errorf("get connection by id: %w", err)
	}

	res, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	rows, err := pgx.CollectRows(res, func(row pgx.CollectableRow) ([]any, error) {
		return row.Values()
	})
	if err != nil {
		return nil, fmt.Errorf("collect rowsL %w", err)
	}

	cols := just.SliceMap(res.FieldDescriptions(), func(fd pgconn.FieldDescription) string {
		return fd.Name
	})

	return &structs.QTable{
		Headers: cols,
		Rows: just.SliceMap(rows, func(row []any) []string {
			return just.SliceMap(row, func(v any) string {
				return fmt.Sprint(v)
			})
		}),
	}, nil
}

func (s *Service) getConnectionByID(ctx context.Context, target config.Target) (*pgxpool.Pool, error) {
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

func (s *Service) AuthType() config.AuthType {
	return s.opts.cfg.Users.Provider.Type()
}
