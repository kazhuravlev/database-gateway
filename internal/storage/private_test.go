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

package storage //nolint:exhaustruct,testpackage

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/go-jet/jet/v2/qrm"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var errRowsAffectedFailed = errors.New("rows affected failed")

type resultStub struct {
	rowsAffected    int64
	rowsAffectedErr error
}

func (resultStub) LastInsertId() (int64, error) {
	return 0, nil
}

func (r resultStub) RowsAffected() (int64, error) {
	return r.rowsAffected, r.rowsAffectedErr
}

func TestHandleError(t *testing.T) {
	t.Parallel()

	rowsAffectedErr := errRowsAffectedFailed

	testCases := []struct {
		name    string
		err     error
		res     sql.Result
		wantErr error
	}{
		{
			name:    "no rows",
			err:     qrm.ErrNoRows,
			wantErr: ErrNotFound,
		},
		{
			name:    "duplicate key",
			err:     &pq.Error{Code: "23505"},
			wantErr: ErrIntegrityViolation,
		},
		{
			name:    "zero rows affected",
			res:     resultStub{rowsAffected: 0},
			wantErr: ErrNotFound,
		},
		{
			name: "rows affected error",
			res: resultStub{
				rowsAffectedErr: rowsAffectedErr,
			},
			wantErr: rowsAffectedErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := handleError("test operation", tc.err, tc.res)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
