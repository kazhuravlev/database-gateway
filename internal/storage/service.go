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

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-jet/jet/v2/qrm"
	"github.com/kazhuravlev/database-gateway/internal/pgdb"
)

var (
	ErrBadRequest         = errors.New("bad request")
	ErrNotFound           = errors.New("not found")
	ErrIntegrityViolation = errors.New("integrity violation")
)

type Service struct {
	opts Options
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	return &Service{
		opts: opts,
	}, nil
}

func (s *Service) Conn(ctx context.Context) qrm.DB { //nolint:ireturn // because pgdb returns a private struct
	return pgdb.NewConn(ctx, s.opts.dbWrite)
}

func (s *Service) DoInTx(ctx context.Context, transactionCallback func(conn qrm.DB) error) error {
	dbTx, err := s.opts.dbWrite.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := transactionCallback(pgdb.NewTx(ctx, dbTx)); err != nil {
		if err := dbTx.Rollback(); err != nil {
			s.opts.logger.Error("rollback transaction",
				slog.String("error", err.Error()))
		}

		return fmt.Errorf("do operation: %w", err)
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
