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
	"database/sql"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	"github.com/kazhuravlev/just"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

func handleError(msg string, err error, res sql.Result) error {
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return fmt.Errorf("%s: %w", msg, ErrNotFound)
		}

		if pqErr, ok := just.ErrAs[*pq.Error](err); ok {
			if pqErr.Code == "23505" {
				return fmt.Errorf("%s: record already exists: %w", msg, ErrIntegrityViolation)
			}
		}

		return fmt.Errorf("%s: %w", msg, err)
	}

	if res != nil {
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("%s: check affected rows: %w", msg, err)
		}

		if affected == 0 {
			return fmt.Errorf("%s: no rows affected: %w", msg, ErrNotFound)
		}
	}

	return nil
}
