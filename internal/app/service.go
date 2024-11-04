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
	conns   map[string]*pgxpool.Pool
}

func New(opts Options) (*Service, error) {
	return &Service{opts: opts, connsMu: new(sync.RWMutex), conns: make(map[string]*pgxpool.Pool)}, nil
}

func (s *Service) AuthUser(ctx context.Context, username, password string) (string, error) {
	for i := range s.opts.cfg.Users {
		user := s.opts.cfg.Users[i]
		if user.Username == username && user.Password == password {
			return user.Username, nil
		}
	}

	return "", fmt.Errorf("user not exists: %w", ErrNotFound)
}

func (s *Service) GetUserByUsername(ctx context.Context, id string) (*config.User, error) {
	for i := range s.opts.cfg.Users {
		user := s.opts.cfg.Users[i]
		if user.Username == id {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not exists: %w", ErrNotFound)
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

func (s *Service) GetTargetByID(ctx context.Context, id string) (*config.Target, error) {
	for i := range s.opts.cfg.Targets {
		target := s.opts.cfg.Targets[i]
		if target.ID == id {
			return &target, nil
		}
	}

	return nil, fmt.Errorf("target not found: %w", ErrNotFound)
}

func (s *Service) RunQuery(ctx context.Context, username, srvID, query string) (*structs.QTable, error) {
	user, err := s.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	srv, err := s.GetTargetByID(ctx, srvID)
	if err != nil {
		return nil, fmt.Errorf("get target by id: %w", err)
	}

	acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
		return acl.Target == srvID
	})

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

	s.opts.logger.Info("connect to target", slog.String("target", target.ID))
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
